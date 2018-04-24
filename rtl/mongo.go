package rtl

import (
    "log"
    "gopkg.in/mgo.v2"
)

var mgosession *mgo.Session

const db = "sart"

var collection string

func InitMgo(s *mgo.Session, cname string) {
    mgosession = s.Copy()
    collection = cname

    c := cache()
    err := c.EnsureIndex(mgo.Index{
        Key: []string{"name"},
        Unique: true,
    })
    if err != nil {
        log.Fatal(err)
    }
}

func EmptyCache() {
    c := cache()
    err := c.DropCollection()
    if err != nil {
        log.Fatal(err)
    }
}

func cache() *mgo.Collection {
    s := mgosession.Copy()
    c := s.DB(db).C(collection)
    return c
}

func (m *Module) Save() {
    c := cache()
    err := c.Insert(m)
    if err != nil {
        log.Fatal(err)
    }
}
