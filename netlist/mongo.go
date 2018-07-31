package netlist

import (
    "log"
    "sync"

    "gopkg.in/mgo.v2"
    // "gopkg.in/mgo.v2/bson"
)

var mgosession *mgo.Session

const db = "sart"

var collection, portcoll, instcoll, conncoll string

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
    // instcoll = cname + "_insts"
    // conncoll = cname + "_conns"

    // var err error

    if drop {
        dropCollection(portcoll)
        // dropCollection(instcoll)
        // dropCollection(conncoll)
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
}

func (n *Netlist) Load() {
}
