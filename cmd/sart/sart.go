package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"sart/ace"
	"sart/netlist"
	"sart/rtl"

	"gopkg.in/mgo.v2"
)

func main() {
	var cache, top, acepath, logp, server string

	var debug, nobuild, nowalk bool

	// Command line switches ///////////////////////////////////////////////////

	flag.StringVar(&cache, "cache", "", "name of cache from which to fetch module info. (req.)")
	flag.StringVar(&top, "top", "", "name of topcell on which to run sart")
	flag.StringVar(&acepath, "ace", "", "path to ace structs file (req.)")
	flag.StringVar(&logp, "log", "", "path to file where log messages should be redirected")
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")

	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	flag.BoolVar(&nobuild, "nobuild", false, "use to skip netlist build step")
	flag.BoolVar(&nowalk, "nowalk", false, "use to skip netlist walk steps")

	flag.Parse()

	// Set log flags ///////////////////////////////////////////////////////////

	log.SetFlags(0)
	if debug {
		log.SetFlags(log.Lshortfile)
	}

	// Check for minimum arguments /////////////////////////////////////////////

	if cache == "" || acepath == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	// Connect to mongo and initialize package rtl's mongo connection //////////

	session, err := mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}

	rtl.InitMgo(session, cache, false)

	// If a log file is specified redirect log messages to it; stdout otherwise

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

	// Load ACE structs file ///////////////////////////////////////////////////

	file, err := os.Open(acepath)
	if err != nil {
		log.Fatal(err)
	}

	acestructs := ace.Load(file)
	log.Printf("Found %d ACE structs.", len(acestructs))

	// Build netlist if needed, otherwise simply initialize package netlist's
	// mongo connection so that a netlist can be loaded

	var start time.Time

	if nobuild {
		netlist.InitMgo(session, cache, false)
	} else {
		netlist.InitMgo(session, cache, true)

		log.Println("Building netlist..")

		start = time.Now()
		nl := netlist.New("", top, top, len(acestructs), 0)
		log.Println(nl)

		netlist.DoneMgo()
		netlist.WaitMgo()
		log.Println("Netlist built. Elapsed:", time.Since(start))
	}

	// By this point, the netlist has been built and saved to mongo. Next ACE
	// nodes need to be marked before starting walks. This has to be done
	// before loading the netlist from mongo. This is a necessary step unless
	// walks are not needed, i.e. when -nowalk is specified.

	// Reset nodes and mark ACE nodes in mongo /////////////////////////////////

	if !nowalk {
		log.Println("Reseting nodes and marking ACE nodes..")
		start = time.Now()
		r, m := netlist.MarkAceNodes(acestructs)
		log.Printf("%d nodes reset and %d ACE nodes marked. Elapsed: %v", r, m,
			time.Since(start))
	}

	// Load netlist from mongo /////////////////////////////////////////////////

	log.Println("Loading netlist..")

	start = time.Now()
	n := netlist.NewNetlist(top)
	n.Load()
	log.Println("Netlist loaded. Elapsed:", time.Since(start))
	log.Println(n)

	// Start walks /////////////////////////////////////////////////////////////

	if !nowalk {
		log.Println("Starting walks..")
		start = time.Now()
		changed := n.Walk()

		for changed > 0 {
			changed = n.Walk()
		}
		log.Println("Walks complete. Elapsed:", time.Since(start))

		// Update mongo with latest ACE info ///////////////////////////////////

		log.Println("Updating nodes..")
		start = time.Now()
		updated := n.Update()
		netlist.UpdateWait()
		log.Printf("Nodes updated: %d. Elapsed: %v", updated, time.Since(start))
	}

	// Print stats and quit ////////////////////////////////////////////////////

	log.Println(n.Stats(acestructs, 0, 0))
}
