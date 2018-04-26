package rtl

import (
    "log"
    "sync"
    "gopkg.in/mgo.v2"
)

var mgosession *mgo.Session

const db = "sart"

var collection, nodecoll, instcoll string

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

// Syncronizers

func DoneMgo() {
    close(jobs)
}

func WaitMgo() {
    wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////

func InitMgo(s *mgo.Session, cname string) {
    mgosession = s.Copy()
    collection = cname

    nodecoll = cname + "_nodes"
    instcoll = cname + "_insts"

    var err error

    deleteCollection(nodecoll)
    deleteCollection(instcoll)

    n := mgosession.DB(db).C(nodecoll)
    err = n.EnsureIndex(mgo.Index{
        Key: []string{"module", "name"},
        Unique: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    i := mgosession.DB(db).C(instcoll)
    err = i.EnsureIndex(mgo.Index{
        Key: []string{"module", "name", "formal"},
        Unique: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Initialize worker pool for insert jobs
    jobs = make(chan insertjob, 100)
    for i := 0; i < MaxMgoThreads; i++ {
        wg.Add(1)
        go worker()
    }
}

// func EmptyCache() {
//     c := cache()
//     err := c.DropCollection()
//     if err != nil {
//         log.Println(err)
//     }
//     n := mgosession.DB(db).C(nodecoll)
//     i := mgosession.DB(db).C(instcoll)
// }

func deleteCollection(coll string) {
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
    for _, n := range m.Nodes {
        jobs <- insertjob{nodecoll, n}
    }

    for _, i := range m.Insts {
        jobs <- insertjob{instcoll, i}
    }
}
