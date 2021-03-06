package netlist

import (
	"log"
	"sart/ace"
	"sart/bitfield"
	"sync"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var mgosession *mgo.Session

const db = "sart"

var nodecoll, linkcoll, snetcoll string

////////////////////////////////////////////////////////////////////////////////
// Worker pool for insert jobs

const MaxMgoThreads = 8

var wg sync.WaitGroup

type insertjob struct {
	col string
	doc interface{}
}

var jobs chan insertjob

func worker() {
	s := mgosession.Copy()

	for job := range jobs {
		c := s.DB(db).C(job.col)
		err := c.Insert(job.doc)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

// Synchronizers

func DoneMgo() {
	close(jobs)
}

func WaitMgo() {
	wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////

var updateJobs chan *Node
var updateWg sync.WaitGroup

func updateWorker() {
	s := mgosession.Copy()
	c := s.DB(db).C(nodecoll)

	for node := range updateJobs {
		sel := bson.M{"module": node.Parent, "name": node.Name}
		err := c.Update(sel, node)
		if err != nil {
			log.Fatal(err)
		}
	}
	updateWg.Done()
}

func UpdateWait() {
	close(updateJobs)
	updateWg.Wait()
}

////////////////////////////////////////////////////////////////////////////////

func InitMgo(s *mgo.Session, cname string, drop bool) {
	mgosession = s.Copy()

	nodecoll = cname + "_nnodes"
	linkcoll = cname + "_nlinks"
	snetcoll = cname + "_nsnets"

	var err error

	if drop {
		dropCollection(nodecoll)
		dropCollection(linkcoll)
		dropCollection(snetcoll)
	}

	n := mgosession.DB(db).C(nodecoll)
	err = n.EnsureIndex(mgo.Index{Key: []string{"module", "name"}, Unique: true})
	if err != nil {
		log.Fatal(err)
	}

	l := mgosession.DB(db).C(linkcoll)
	err = l.EnsureIndex(mgo.Index{Key: []string{"module"}})
	if err != nil {
		log.Fatal(err)
	}

	b := mgosession.DB(db).C(snetcoll)
	err = b.EnsureIndex(mgo.Index{Key: []string{"module", "name"}, Unique: true})
	if err != nil {
		log.Fatal(err)
	}

	// Initialize worker pool for insert jobs
	jobs = make(chan insertjob, 100)
	for i := 0; i < MaxMgoThreads; i++ {
		wg.Add(1)
		go worker()
	}

	updateJobs = make(chan *Node, 100)
	for i := 0; i < MaxMgoThreads; i++ {
		updateWg.Add(1)
		go updateWorker()
	}
}

func dropCollection(coll string) {
	c := mgosession.DB(db).C(coll)
	err := c.DropCollection()
	if err != nil {
		log.Println(err)
	}
}

func MarkAceNodes(acestructs []ace.AceStruct) (reset, marked int) {
	s := mgosession.Copy()
	c := s.DB(db).C(nodecoll)

	maxace := len(acestructs)

	// Reset the ACE information of all nodes that had changed. ////////////////

	bf := bitfield.New(maxace)
	sel := bson.M{
		// Select if read or write port ACE terms has a non-zero character
		// because if a node was never marked with an ACE value during a walk,
		// its ACE terms string will be all 0s.
		"$or": []bson.M{
			bson.M{"rpace": bson.RegEx{"[^0]", ""}},
			bson.M{"wpace": bson.RegEx{"[^0]", ""}},
		},
	}
	upd := bson.M{
		// Update it with an empty bitfield of the required size.
		"$set": bson.M{
			"isace":  false,
			"walked": false,
			"rpace":  bf,
			"wpace":  bf,
		},
	}

	ci, err := c.UpdateAll(sel, upd)
	if err != nil {
		log.Fatal(err)
	}
	reset = ci.Updated

	// Mark ACE nodes //////////////////////////////////////////////////////////

	// The index of the ACE struct in the array will be the bit to set in the
	// bitfield to indicate its contribution to the pAVF equation.
	for i, s := range acestructs {
		rpbf := bitfield.New(maxace)
		wpbf := bitfield.New(maxace)
		rpbf.Set(i)
		wpbf.Set(i)

		sel := bson.M{}

		if s.Selector.Module != "" {
			sel["module"] = bson.RegEx{s.Selector.Module, ""}
		}
		if s.Selector.Name != "" {
			sel["name"] = bson.RegEx{s.Selector.Name, ""}
		}

		upd := bson.M{
			"$set": bson.M{
				"isace": true,
				"rpace": rpbf,
				"wpace": wpbf,
			},
		}

		ci, err := c.UpdateAll(sel, upd)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("(%d/%d) Marked %d nodes ACE with %v", i+1, maxace,
			ci.Updated, s)
		marked += ci.Updated
	}

	return
}

func (n *Netlist) Save() {
	for _, node := range n.Nodes {
		jobs <- insertjob{nodecoll, node}
	}

	// Links is a map of right-nodes indexed using the fullname of the
	// left-node. It is sufficient to push just the fullname of the rnode into
	// mongo as during retrieval, the right-node-fullname can be used to locate
	// the node which should already have been loaded.
	for lfullname, rnodes := range n.Links {
		for _, rnode := range rnodes {
			doc := bson.M{
				"module":    n.Name,
				"lfullname": lfullname,
				"rfullname": rnode.Fullname(),
			}
			jobs <- insertjob{linkcoll, doc}
		}
	}

	for _, subnet := range n.Subnets {
		doc := bson.M{
			"module": n.Name,
			"name":   subnet.Name,
		}
		jobs <- insertjob{snetcoll, doc}
	}
}

func (n *Netlist) Update() (count int) {
	for _, node := range n.Nodes {
		// If a node that is not ace has been touched by an ACE value, update
		// to reflect this in mongo.
		if !node.IsAce && (!node.RpAce.AllUnset() || !node.WpAce.AllUnset()) {
			updateJobs <- node
			count++
		}
	}

	for _, subnet := range n.Subnets {
		count += subnet.Update()
	}

	return
}

func (n *Netlist) Load() {
	n.LoadNodes(0)
	n.LoadLinks(0)
}

func (n *Netlist) LoadNodes(level int) {
	// log.Printf("Nodes Load (%d) %q", level, n.Name)
	var result bson.M

	// nodes collection, query and iterator
	nc := mgosession.DB(db).C(nodecoll)
	nq := nc.Find(bson.M{"module": n.Name})
	ni := nq.Iter()

	for ni.Next(&result) {
		bytes, err := bson.Marshal(result)
		if err != nil {
			log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
				result["module"], result["name"], err)
		}

		var node Node
		err = bson.Unmarshal(bytes, &node)
		if err != nil {
			log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
				result["module"], result["name"], err)
		}

		n.AddNode(&node)
	}

	// Use parallel loader for subnet nodes.
	loader := NewNodeLoader(level + 1)

	// subnet collection, query and iterator
	sc := mgosession.DB(db).C(snetcoll)
	sq := sc.Find(bson.M{"module": n.Name}).Select(bson.M{"_id": 0, "module": 0})
	si := sq.Iter()

	for si.Next(&result) {
		fullname := result["name"].(string)
		subnet := NewNetlist(fullname)
		n.Subnets[fullname] = subnet

		// This is effectively subnet.LoadNodes() except that this will run in
		// parallel through a worker pool.
		loader.Add(subnet)
	}

	// Indicate that there will be no more load jobs and wait till all subnets
	// are loaded. If we don't wait the links being loaded next will not have
	// all the nodes needed to get hooked up.
	loader.Done()
	loader.Wait()

	// log.Printf("Nodes Done (%d) %q", level, n.Name)
}

func (n *Netlist) LoadLinks(level int) {
	// log.Printf("Links Load (%d) %q", level, n.Name)
	var result bson.M

	// link collection, query and iterator
	lc := mgosession.DB(db).C(linkcoll)
	lq := lc.Find(bson.M{"module": n.Name}).Select(bson.M{"_id": 0})
	li := lq.Iter()

	for li.Next(&result) {
		lfullname := result["lfullname"].(string)
		rfullname := result["rfullname"].(string)
		lnode := n.LocateNode(lfullname)
		rnode := n.LocateNode(rfullname)

		if lnode == nil {
			log.Fatalf("Could not locate lnode %q in netlist %q", lfullname, n.Name)
		}

		if rnode == nil {
			log.Fatalf("Could not locate rnode %q in netlist %q", rfullname, n.Name)
		}

		n.Connect(lnode, rnode)
	}

	// Use parallel loader for subnet links.
	loader := NewLinkLoader(level + 1)

	for _, subnet := range n.Subnets {
		// This is effectively subnet.LoadLinks() except that this will run in
		// parallel through a worker pool.
		loader.Add(subnet)
	}

	loader.Done()
	loader.Wait()

	// log.Printf("Links Done (%d) %q", level, n.Name)
}

////////////////////////////////////////////////////////////////////////////////

type Loader interface {
	LoadNodes(int)
	LoadLinks(int)
}

type NetlistLoader struct {
	loadjobs chan Loader
	wg       sync.WaitGroup
}

func NewNetlistLoader() *NetlistLoader {
	l := &NetlistLoader{
		loadjobs: make(chan Loader, 1000),
	}
	return l
}

func NewNodeLoader(level int) *NetlistLoader {
	l := NewNetlistLoader()

	for i := 0; i < MaxMgoThreads; i++ {
		l.wg.Add(1)
		go func() {
			for job := range l.loadjobs {
				job.LoadNodes(level)
			}
			l.wg.Done()
		}()
	}

	return l
}

func NewLinkLoader(level int) *NetlistLoader {
	l := NewNetlistLoader()

	for i := 0; i < MaxMgoThreads; i++ {
		l.wg.Add(1)
		go func() {
			for job := range l.loadjobs {
				job.LoadLinks(level)
			}
			l.wg.Done()
		}()
	}

	return l
}

// Add a job to the loader
func (l *NetlistLoader) Add(job Loader) {
	l.loadjobs <- job
}

// Done is to be invoked when there are no more jobs for NetlistLoader. It
// closes the internal channel used for scheduling and synchronizing.
func (l *NetlistLoader) Done() {
	close(l.loadjobs)
}

// Wait waits till all the load jobs have completed. Wait is to be invoked
// after Done is invoked to indicate that there will be no more jobs for
// NetlistLoader.
func (l *NetlistLoader) Wait() {
	l.wg.Wait()
}
