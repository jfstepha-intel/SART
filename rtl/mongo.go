package rtl

import (
    "gopkg.in/mgo.v2"
)

var mgosession *mgo.Session

const db = "sart"

var collection string

func InitMgo(s *mgo.Session, c string) {
    mgosession = s.Copy()
    collection = c
}
