package parse

import (
    "gopkg.in/mgo.v2"
)

var mgosession *mgo.Session

func SetMongoSession(s *mgo.Session) {
    mgosession = s.Copy()
}
