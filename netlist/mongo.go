package netlist

import (
    "log"
    "sync"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"

    "sart/rtl"
)

var mgosession *mgo.Session

const db = "sart"

var collection, portcoll, nodecoll, linkcoll, snetcoll string

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

func InitMgo(s *mgo.Session, cname string, drop bool) {
    mgosession = s.Copy()
    collection = cname

    portcoll = cname + "_nports"
    nodecoll = cname + "_nnodes"
    linkcoll = cname + "_nlinks"
    snetcoll = cname + "_nsnets"

    // var err error

    if drop {
        dropCollection(portcoll)
        dropCollection(nodecoll)
        dropCollection(linkcoll)
        dropCollection(snetcoll)
    }

    // TODO create indexes

    // Initialize worker pool for insert jobs
    jobs = make(chan insertjob, 100)
    for i := 0; i < MaxMgoThreads; i++ {
        wg.Add(1)
        go worker()
    }
}

func dropCollection(coll string) {
    c := mgosession.DB(db).C(coll)
    err := c.DropCollection()
    if err != nil {
        log.Println(err)
    }
}

func cache() *mgo.Collection {
    s := mgosession.Copy()
    c := s.DB(db).C(collection)
    return c
}

func (n *Netlist) Save() {
    for _, port := range n.Ports {
        jobs <- insertjob{portcoll, port}
    }

    for _, node := range n.Nodes {
        jobs <- insertjob{nodecoll, node}
    }

    for _, link := range n.Links {
        jobs <- insertjob{linkcoll, link}
    }

    for _, subnet := range n.Subnets {
        doc := bson.M {
            "module": n.Name,
            "subnet": subnet.Name,
        }
        jobs <- insertjob{snetcoll, doc}
    }
}

func (n *Netlist) Load() {
    // ports collection, query and iterator
    pc := mgosession.DB(db).C(portcoll)

    // Sort by pos to ensure port ordering.
    pq := pc.Find(bson.M{"module": n.Name}).Sort("pos")
    pi := pq.Iter()

    var result bson.M

    for pi.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        var port rtl.Port
        err = bson.Unmarshal(bytes, &port)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        n.Ports = append(n.Ports, &port)
    }

    // nodes collection, query and iterator
    nc := mgosession.DB(db).C(nodecoll)
    nq := nc.Find(bson.M{"module": n.Name})
    ni := nq.Iter()

    for ni.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        var node Node
        err = bson.Unmarshal(bytes, &node)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        fullname := node.Parent + "/" + node.Name
        n.Nodes[fullname] = &node
    }

    // link collection, query and iterator
    lc := mgosession.DB(db).C(linkcoll)
    lq := lc.Find(bson.M{"module": n.Name})
    li := lq.Iter()

    for li.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        var link Link
        err = bson.Unmarshal(bytes, &link)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        n.Links = append(n.Links, link)
    }

    // subnet collection, query and iterator
    sc := mgosession.DB(db).C(snetcoll)
    sq := sc.Find(bson.M{"module": n.Name}).Select(bson.M{"_id":0, "module":0})
    si := sq.Iter()

    for si.Next(&result) {
        subnetname := result["subnet"].(string)
        subnet := NewNetlist(subnetname)
        n.Subnets[subnetname] = subnet
        subnet.Load()
    }
}
