package main

import (
	"log"
	"sart/rtl"
	"sart/set"
	"strconv"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Extend rtl.Prop to hold an interpreted value
type Prop struct {
	rtl.Prop
	Fval float64
}

type PropMap map[string][]Prop

func (m PropMap) Add(prop Prop) {
	m[prop.Parent] = append(m[prop.Parent], prop)
}

func (m PropMap) Print() {
	for key, val := range m {
		log.Println(key, val)
	}
}

var props PropMap

func LoadWidths(session *mgo.Session, cache string) {
	iter := session.DB("sart").C(cache + "_props").Find(bson.M{"key": "W"}).Select(bson.M{"_id": 0}).Iter()

	var result bson.M

	for iter.Next(&result) {
		bytes, err := bson.Marshal(result)
		if err != nil {
			log.Fatalf("Unable to marshal. module:%q iname:%q err:%v",
				result["module"], result["iname"], err)
		}

		var prop rtl.Prop
		err = bson.Unmarshal(bytes, &prop)
		if err != nil {
			log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
				result["module"], result["iname"], err)
		}

		fval, err := strconv.ParseFloat(strings.TrimSuffix(prop.Val, "u"), 64)
		if err != nil {
			log.Fatal(err)
		}

		props.Add(Prop{prop, fval})
	}
}

var primparents set.Set

func LoadPrimParents(session *mgo.Session, cache string) {
	var ppresults []interface{}
	err := session.DB("sart").C(cache+"_insts").Find(bson.M{"isprimparent": true}).Distinct("module", &ppresults)
	if err != nil {
		log.Fatal(err)
	}

	for _, primparent := range ppresults {
		primparents.Add(primparent.(string))
	}
}

func init() {
	props = make(PropMap)

	primparents = set.New()
}
