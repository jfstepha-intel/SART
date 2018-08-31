package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"gopkg.in/mgo.v2"

	"sart/netlist"
	"sart/rtl"
)

func main() {
	var cache, top, ace, logp, server string

    var debug bool

	flag.StringVar(&cache, "cache", "", "name of cache from which to fetch module info.")
	flag.StringVar(&top, "top", "", "name of topcell on which to run sart")
	flag.StringVar(&ace, "ace", "", "path to ace structs file")
	flag.StringVar(&logp, "log", "", "path to file where log messages should be redirected")
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")

	flag.BoolVar(&debug, "debug", false, "enable debug mode")

	flag.Parse()

	log.SetFlags(0)
	if debug {
		log.SetFlags(log.Lshortfile)
	}

	if cache == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	session, err := mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}
	rtl.InitMgo(session, cache, false)

	var logw io.Writer
	if logp != "" {
		var err error
		logw, err = os.Create(logp)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		logw = os.Stdout
	}
	log.SetOutput(logw)

	var start time.Time

	netlist.InitMgo(session, cache, true)

	start = time.Now()
	// netlist.New("", top, top)
	nl := netlist.New("", top, top)
	log.Println(nl)

	netlist.DoneMgo()
	netlist.WaitMgo()
	log.Println("Netlist built. Elapsed:", time.Since(start))

	start = time.Now()
	n := netlist.NewNetlist(top)
	n.Load()
	log.Println("Netlist loaded. Elapsed:", time.Since(start))
	log.Println(n)

	log.Println("Starting walks..")
	changed1 := n.Walk()
	log.Println(changed1)

	n.Stats("")
}
