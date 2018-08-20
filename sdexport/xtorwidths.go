package main

import (
	"log"
	"sart/rtl"
	"strconv"
	"strings"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Prop struct {
	rtl.Prop
	Fval float64
}

type PropMap map[string][]Prop

func (m PropMap) Add(prop Prop) {
	m[prop.Parent] = append(m[prop.Parent], prop)
}

var props PropMap

func LoadWidths(session *mgo.Session, cache string) {
	iter := session.DB("sart").C(cache + "_props").Find(bson.M{"key": "W"}).Select(bson.M{"_id": 0}).Iter()

	var result bson.M

	for iter.Next(&result) {
		// module := result["module"].(string)
		// itype := result["itype"].(string)
		// val := result["val"].(string)

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

//// func LoadXtorWidths(session *mgo.Session, cache string) {
//// 	iter := session.DB("sart").C(cache + "_props").Find(bson.M{"key": "W"}).Select(bson.M{"_id": 0}).Iter()
////
//// 	var result bson.M
////
//// 	for iter.Next(&result) {
//// 		module := result["module"].(string)
//// 		iname := result["iname"].(string)
//// 		val := result["val"].(string)
////
//// 		key := module + "/" + iname
////
//// 		if !strings.HasSuffix(val, "u") {
//// 			log.Fatal(val)
//// 		}
////
//// 		value, err := strconv.ParseFloat(strings.TrimSuffix(val, "u"), 64)
//// 		if err != nil {
//// 			log.Fatal(err)
//// 		}
////
//// 		// log.Println(key, value)
//// 		XtorWidths.AddUq(key, value)
//// 	}
//// }

func init() {
	props = make(PropMap)
}
