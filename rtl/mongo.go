package rtl

import (
    "log"
    "sync"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

var mgosession *mgo.Session

const db = "sart"

var collection, portcoll, instcoll, conncoll, propcoll string

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

    portcoll = cname + "_ports"
    instcoll = cname + "_insts"
    conncoll = cname + "_conns"
    propcoll = cname + "_props"

    var err error

    if drop {
        dropCollection(portcoll)
        dropCollection(instcoll)
        dropCollection(conncoll)
        dropCollection(propcoll)
    }

    // Each port in a module must have a unique name
    n := mgosession.DB(db).C(portcoll)
    err = n.EnsureIndex(mgo.Index{ Key: []string{"module", "name"}, Unique: true })
    if err != nil { log.Fatal(err) }

    // Each instance in a module must have a unique name
    i := mgosession.DB(db).C(instcoll)
    err = i.EnsureIndex(mgo.Index{ Key: []string{"module", "name"}, Unique: true })
    if err != nil { log.Fatal(err) }

    // Index the type as well because there will be update queries using type
    // as selector
    err = i.EnsureIndex(mgo.Index{ Key: []string{"type"} })
    if err != nil { log.Fatal(err) }

    // Each formal name of an instance connection in a module must be unique
    c := mgosession.DB(db).C(conncoll)
    err = c.EnsureIndex(mgo.Index{ Key: []string{"module", "iname", "pos"}, Unique: true })
    if err != nil { log.Fatal(err) }

    // Index the itype as well because there will be update queries using itype
    // as selector
    err = c.EnsureIndex(mgo.Index{ Key: []string{"itype"} })
    if err != nil { log.Fatal(err) }

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

func (m *Module) Save() {
    for _, port := range m.Ports {
        jobs <- insertjob{portcoll, port}
    }

    for _, inst := range m.Insts {
        jobs <- insertjob{instcoll, inst}
    }

    for _, conns := range m.Conns {
        for _, conn := range conns {
            jobs <- insertjob{conncoll, conn}
        }
    }

    for _, props := range m.Props {
        for _, prop := range props {
            jobs <- insertjob{propcoll, prop}
        }
    }
}

func (m *Module) Load() {
    // ports collection, query and iterator
    wc := mgosession.DB(db).C(portcoll)
    wq := wc.Find(bson.M{"module": m.Name})
    wi := wq.Iter()

    var result bson.M

    for wi.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        var port Port
        err = bson.Unmarshal(bytes, &port)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        m.AddPort(&port)
    }

    // instance collection, query and iterator
    ic := mgosession.DB(db).C(instcoll)
    iq := ic.Find(bson.M{"module": m.Name})
    ii := iq.Iter()

    for ii.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        var inst Inst
        err = bson.Unmarshal(bytes, &inst)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["name"], err)
        }

        m.AddInst(&inst)
    }

    // connection collection, query and iterator
    cc := mgosession.DB(db).C(conncoll)
    cq := cc.Find(bson.M{"module": m.Name})
    ci := cq.Iter()

    for ci.Next(&result) {
        bytes, err := bson.Marshal(result)
        if err !=nil {
            log.Fatalf("Unable to marshal. module:%q iname:%q err:%v",
                       result["module"], result["iname"], err)
        }

        var conn Conn
        err = bson.Unmarshal(bytes, &conn)
        if err != nil {
            log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
                       result["module"], result["iname"], err)
        }

        m.AddConn(&conn)
    }
}

// InstNames returns a map with name-value pairs corresponding to the name and
// type of the instantiations in a module. Each pair will need a subnet built.
// The implementation of this method is a simple aggregation pipeline on the
// conns collection.
//
// References:
// 1. https://stackoverflow.com/questions/11973725/how-to-efficiently-perform-distinct-with-multiple-keys
// 2. https://docs.mongodb.com/manual/reference/operator/aggregation/group
// 3. https://godoc.org/labix.org/v2/mgo#Collection.Pipe
//
func (m Module) InstNames() map[string]string {
    insts := make(map[string]string)

    c := mgosession.DB(db).C(conncoll)

    // Setup the aggregation pipeline
    pipe := c.Pipe([]bson.M{
        // Filter to pick only the connections that apply to this module
        bson.M{"$match"  : bson.M{"module": m.Name}},
        // Need only name type fields
        bson.M{"$project": bson.M{"name": 1, "type": 1}},
        // Next find distinct name-type pairs
        bson.M{"$group"  : bson.M{"_id": bson.M{"name": "$name", "type": "$type"}}},
    })

    // Run and gather all results
    var result []bson.M
    err := pipe.All(&result)
    if err != nil {
        log.Fatal(err)
    }

    // The structure of the returned documents is thus: 
    // { "_id" : { "name" : "irep61", "type" : "sncclnt_ec0bfm202al1n02x5" } }
    for _, val := range result {
        doc := val["_id"].(bson.M)
        insts[doc["name"].(string)] = doc["type"].(string)
    }

    return insts
}

func LoadModule(top string) *Module {
    m := NewModule(top)
    m.Load()
    return m
}
