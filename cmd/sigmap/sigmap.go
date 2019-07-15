package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"sart/netlist"
	"sart/rtl"
	"sort"

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

func findOneMatchModule(module string) {
	q := nodesCollection.Find(bson.M{"module": bson.RegEx{module, ""}})

	var result bson.M

	err := q.One(&result)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found at least one node that matched -module regex %q", module)
	log.Println(bson2node(result))
}

func findOneMatchName(name string) {
	q := nodesCollection.Find(bson.M{"name": bson.RegEx{name, ""}})

	var result bson.M

	err := q.One(&result)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found at least one node that matched -name regex %q", name)
	log.Println(bson2node(result))
}

func findDistinctMatches(module, name string) {
	q := nodesCollection.Find(bson.M{"name": bson.RegEx{name, ""}})

	var result []string

	err := q.Distinct("name", &result)
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(result)

	log.Printf("Unique node names that match -name regex %q", name)
	for i, n := range result {
		log.Printf("%4d %q", i+1, n)
	}

	// Find distinct modules that contain nodes that match the -name regex
	err = q.Distinct("module", &result)
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(result)

	log.Printf("Unique module names that match -name regex %q", name)
	for i, n := range result {
		log.Printf("%4d %q", i+1, n)
	}

	// Find distinct types of nodes that match the -name regex
	err = q.Distinct("type", &result)
	if err != nil {
		log.Fatal(err)
	}

	sort.Strings(result)

	log.Printf("Unique types that match -name regex %q", name)
	for i, n := range result {
		log.Printf("%4d %q", i+1, n)
	}
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
		// findOneMatchModule(module)
	}

	if name != "" {
		// findOneMatchName(name)
		findDistinctMatches(module, name)
	}

}
