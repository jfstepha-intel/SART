package rtl

import (
    "log"
    "sync"
    "gopkg.in/mgo.v2"
)

var mgosession *mgo.Session

const db = "sart"

var collection string

////////////////////////////////////////////////////////////////////////////////
// Worker pool for insert jobs

const MaxMgoThreads = 8

var wg sync.WaitGroup

type insertjob struct {
    collection string
    doc interface{}
}

var jobs chan interface{}

func worker() {
    s := mgosession.Copy()
    c := s.DB(db).C(collection)

    for doc := range jobs {
        err := c.Insert(doc)
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

    EmptyCache()

    c := cache()
    err := c.EnsureIndex(mgo.Index{
        Key: []string{"module", "name"},
        Unique: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Initialize worker pool for insert jobs
    jobs = make(chan interface{}, 100)
    for i := 0; i < MaxMgoThreads; i++ {
        wg.Add(1)
        go worker()
    }
}

func EmptyCache() {
    c := cache()
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
        jobs <- n
    }
}
