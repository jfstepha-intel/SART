package main

import (
	"flag"
	"log"
	"os"
	"sart/ace"
	"sart/netlist"
	"sart/rtl"
	"time"

	mgo "gopkg.in/mgo.v2"
)

func NetTree(prefix string, level int, n *netlist.Netlist) {
	if level > 2 {
		return
	}
	log.Printf("%s%s %v", prefix, n.Shortname(), n.Stats(acestructs, 0, 0))
	for s := range n.Subnets {
		NetTree(prefix+"|   ", level+1, n.Subnets[s])
	}
}

var acestructs []ace.AceStruct

func main() {
	var cache, top, server, acepath string
	var session *mgo.Session

	flag.StringVar(&cache, "cache", "", "name of cache from which to fetch netlist")
	flag.StringVar(&top, "top", "", "name of top cell to start traversing")
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")
	flag.StringVar(&acepath, "ace", "", "path to ace structs file (req.)")

	flag.Parse()

	log.SetFlags(log.Lshortfile)

	if cache == "" || top == "" || acepath == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	file, err := os.Open(acepath)
	if err != nil {
		log.Fatal(err)
	}

	acestructs = ace.Load(file)
	log.Printf("Found %d ACE structs.", len(acestructs))

	session, err = mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}
	rtl.InitMgo(session, cache, false)

	netlist.InitMgo(session, cache, false)

	n := netlist.NewNetlist(top)
	log.Printf("Loading netlist %s..", top)
	start := time.Now()
	n.Load()
	log.Println("Done. Time elapsed:", time.Since(start))

	log.SetOutput(os.Stdout)

	NetTree("", 0, n)
}
