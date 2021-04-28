package main

import (
	"flag"
	"log"
	"os"
	"sart/rtl"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var indent string = "|   "

func Tree(prefix string, level int, top, name string) {
	if upto > 0 && level > upto {
		return
	}

	log.Printf("%s%s (%s)", prefix, top, name)
	q := session.DB("sart").C(cache + "_insts").Find(bson.M{"module": top})

	iter := q.Iter()

	var inst bson.M
	for iter.Next(&inst) {
		if !inst["isprim"].(bool) {
			Tree(prefix+indent, level+1, inst["type"].(string), inst["name"].(string))
		}
	}
}

var server, cache, top string
var session *mgo.Session
var upto int

func main() {
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")
	flag.StringVar(&cache, "cache", "", "name of cache to save module info (req.)")
	flag.StringVar(&top, "top", "", "name of top cell to explore (req.)")
	flag.IntVar(&upto, "upto", 1, "depth to which hierarchy is sought. -1 for full hierarchy")

	flag.Parse()

	log.SetFlags(log.Lshortfile)
	log.SetFlags(0)

	if cache == "" || top == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	var err error
	session, err = mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}
	rtl.InitMgo(session, cache, false)

	log.SetOutput(os.Stdout)

	Tree("", 0, top, top)
}
