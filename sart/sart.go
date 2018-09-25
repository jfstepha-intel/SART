package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"sart/netlist"
	"sart/rtl"

	"gopkg.in/mgo.v2"
)

func main() {
	var cache, top, ace, logp, server string

	var debug, nobuild, nowalk bool

	flag.StringVar(&cache, "cache", "", "name of cache from which to fetch module info.")
	flag.StringVar(&top, "top", "", "name of topcell on which to run sart")
	flag.StringVar(&ace, "ace", "", "path to ace structs file")
	flag.StringVar(&logp, "log", "", "path to file where log messages should be redirected")
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")

	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.BoolVar(&nobuild, "nobuild", false, "use to skip netlist build step")
	flag.BoolVar(&nowalk, "nowalk", false, "use to skip netlist walk steps")

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

	if nobuild {
		netlist.InitMgo(session, cache, false)
	} else {
		netlist.InitMgo(session, cache, true)

		log.Println("Building netlist..")

		start = time.Now()
		// netlist.New("", top, top)
		nl := netlist.New("", top, top, 0)
		log.Println(nl)

		netlist.DoneMgo()
		netlist.WaitMgo()
		log.Println("Netlist built. Elapsed:", time.Since(start))
	}

	if nowalk {
		return
	}

	start = time.Now()
	n := netlist.NewNetlist(top)
	n.Load()
	log.Println("Netlist loaded. Elapsed:", time.Since(start))
	log.Println(n)

	log.Println("Starting walks..")
	changed := n.Walk()
	log.Println(changed)

	for changed > 0 {
		changed = n.Walk()
		log.Println(changed)
	}

	n.Stats("")
}
