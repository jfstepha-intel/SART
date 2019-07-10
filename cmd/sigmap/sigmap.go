package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sart/netlist"
	"sart/rtl"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const sartdb = "sart"

// Redefine Node locally so that Stringer can be overridden.
type Node netlist.Node

func (n Node) String() string {
	json, err := json.MarshalIndent(n, "", "    ")
	if err != nil {
		log.Fatal("Unable to MarshalIndent")
	}
	return string(json)
}

func bson2node(val bson.M) (node Node) {
	bytes, err := bson.Marshal(val)
	if err != nil {
		log.Fatalf("Unable to marshal. module:%q name:%q err:%v",
			val["module"], val["name"], err)
	}

	err = bson.Unmarshal(bytes, &node)
	if err != nil {
		log.Fatalf("Unable to umarshal. module:%q name:%q err:%v",
			val["module"], val["name"], err)
	}

	return
}

func findOneModule(module string) {
	q := nodesCollection.Find(bson.M{"module": bson.RegEx{module, ""}})

	var result bson.M

	err := q.One(&result)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found at least one node that matched module regex %q", module)
	log.Println(bson2node(result))
}

var server, cache string
var session *mgo.Session

var nodesCollection *mgo.Collection

func main() {
	var module, name string

	flag.StringVar(&server, "server", "localhost", "name of mongodb server")
	flag.StringVar(&cache, "cache", "", "name of cache to search in (req.)")
	flag.StringVar(&module, "module", "", "regex to match module")
	flag.StringVar(&name, "name", "", "regex to match name")

	flag.Parse()

	log.SetFlags(log.Lshortfile)

	if cache == "" {
		flag.PrintDefaults()
		log.Fatal("-E- Insufficient arguments. -cache is required.")
	}

	if module == "" && name == "" {
		flag.PrintDefaults()
		log.Fatal("-E- Insufficient arguments. Please specify either -module or -name")
	}

	var err error
	session, err = mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}
	rtl.InitMgo(session, cache, false)

	nodesCollection = session.DB(sartdb).C(cache + "_nnodes")

	// Every log message up to here will go to stderr
	log.SetOutput(os.Stdout)

	if module != "" {
		findOneModule(module)
	}

}
