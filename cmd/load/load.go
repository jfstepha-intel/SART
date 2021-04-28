package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	// "sart/parse"
	"sart/parsesp"
	"sart/rtl"
	"sart/set"
)

func parseWorker(wg *sync.WaitGroup, jobs <-chan string) {
	for path := range jobs {
		file, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}

		parsesp.New(path, file)

		file.Close()
	}
	wg.Done()
}

func updateWorker(wg *sync.WaitGroup, jobs <-chan string) {
	sess := session.Copy()
	inst := sess.DB("sart").C(cache + "_insts")
	conn := sess.DB("sart").C(cache + "_conns")
	for itype := range jobs {
		_, err := inst.UpdateAll(
			bson.M{"type": itype}, // Selector
			bson.M{"$set": bson.M{"isprim": true}},
		)
		if err != nil {
			log.Fatal(err)
		}

		_, err = conn.UpdateAll(
			bson.M{"itype": itype}, // Selector
			bson.M{"$set": bson.M{"isprim": true}},
		)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

type connTypeUpdateJob struct {
	Module string
	Pos    int
	Type   string
}

func connTypeUpdateWorker(wg *sync.WaitGroup, jobs <-chan connTypeUpdateJob) {
	sess := session.Copy()
	conn := sess.DB("sart").C(cache + "_conns")
	for job := range jobs {
		_, err := conn.UpdateAll(
			bson.M{"itype": job.Module, "pos": job.Pos}, // Selector
			bson.M{"$set": bson.M{"type": job.Type}},
		)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

func prmpUpdateWorker(wg *sync.WaitGroup, jobs <-chan string) {
	sess := session.Copy()
	inst := sess.DB("sart").C(cache + "_insts")
	for job := range jobs {
		_, err := inst.UpdateAll(
			bson.M{"module": job}, // Selector
			bson.M{"$set": bson.M{"isprimparent": true}},
		)
		if err != nil {
			log.Fatal(err)
		}
		_, err = inst.UpdateAll(
			bson.M{"type": job}, // Selector
			bson.M{"$set": bson.M{"isprim": true}},
		)
		if err != nil {
			log.Fatal(err)
		}
	}
	wg.Done()
}

var session *mgo.Session
var cache string

func main() {
	var path, server, seqre string
	var threads int
	var noparse, qonly bool

	flag.StringVar(&path, "path", "", "path to folder with netlist files (req.)")
	flag.StringVar(&server, "server", "localhost", "name of mongodb server")
	flag.StringVar(&cache, "cache", "", "name of cache to save module info (req.)")
	flag.StringVar(&seqre, "seqre", "ec0[fl]", "regular expression to mark sequential cells e.g. 'ec0[fl]'")
	flag.IntVar(&threads, "threads", 4, "number of parallel threads to spawn")
	flag.BoolVar(&noparse, "noparse", false, "include to skip parse step")
	flag.BoolVar(&qonly, "qismatonly", false, "include to skip sart steps")

	flag.Parse()

	log.SetFlags(log.Lshortfile)

	if path == "" || cache == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	// Connect to MongoDB //////////////////////////////////////////////////////

	var err error

	session, err = mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}

	// Set the connection timeout to a large number because for really large
	// netlists the host may run out of memory and might need to swap before
	// requests can be fulfilled. For these, the default of 1 minute is clearly
	// insufficient. Ref:
	// https://stackoverflow.com/questions/24652587/i-o-timeout-with-mgo-and-mongodb
	session.SetSocketTimeout(3 * time.Hour)

	rtl.InitMgo(session, cache, !noparse)

	log.SetOutput(os.Stdout)

	var count, total int

	if !noparse {
		// Setup inputs, waitgroup and worker threads //////////////////////////////

		files, err := ioutil.ReadDir(path)
		if err != nil {
			log.Fatal(err)
		}

		var parsewg sync.WaitGroup
		parsejobs := make(chan string, 100)

		for i := 0; i < threads; i++ {
			go parseWorker(&parsewg, parsejobs)
			parsewg.Add(1)
		}

		// Loop over files and add to parsers pool /////////////////////////////////

		count = 0
		total = len(files)
		for _, file := range files {
			filename := file.Name()
			count++

			if !strings.HasSuffix(filename, ".sp") {
				continue
			}

			fpath := path + "/" + filename
			parsejobs <- fpath

			log.Printf("load: (%d/%d) %s", count, total, filename)
		}

		// No more parse jobs
		close(parsejobs)
		parsewg.Wait()

		rtl.DoneMgo() // Signal no more mongo insert jobs
		rtl.WaitMgo() // Wait for all insert jobs to complete
	}

	if qonly {
		return
	}

	////////////////////////////////////////////////////////////////////////////
	// At this point all available information in the input netlists have been
	// parsed, sliced and diced into wires, insts and conns collections in the
	// mongo database. Next we need to mark all the instantiations for which a
	// module definition was not found as primitives.
	////////////////////////////////////////////////////////////////////////////

	log.Println("Marking primitives..")

	// In the instance collection, a list of all distinct types is the universe
	// of everything that has been instantiated at least once.
	var allmodules []interface{}
	err = session.DB("sart").C(cache+"_insts").Find(nil).Distinct("type", &allmodules)
	if err != nil {
		log.Fatal(err)
	}

	// These are modules that have instantiations inside them. I.e there is a
	// SUBCKT definition
	var instmodules []interface{}
	err = session.DB("sart").C(cache+"_insts").Find(nil).Distinct("module", &instmodules)
	if err != nil {
		log.Fatal(err)
	}

	// Create sets out of these lists
	allm := set.New(allmodules...)
	inst := set.New(instmodules...)

	// Setup worker pool for update queries ////////////////////////////////////

	var updatewg sync.WaitGroup
	updatejobs := make(chan string, 100)

	for i := 0; i < threads; i++ {
		go updateWorker(&updatewg, updatejobs)
		updatewg.Add(1)
	}

	// Remove the non-empty modules from the universe to identify primitives.
	prims := allm.Not(inst).Sort()
	total = len(prims)
	count = 0

	// Loop over each primitive that was found and add to the updaters pool
	for _, prim := range prims {
		updatejobs <- prim
		count++
		log.Printf("prim: (%d/%d) %s", count, total, prim)
	}
	close(updatejobs)

	updatewg.Wait()

	////////////////////////////////////////////////////////////////////////////

	log.Println("Marking primitive parents..")

	// These are modules that have 'X' instantiations inside them.
	var primparents []interface{}
	err = session.DB("sart").C(cache+"_insts").Find(bson.M{"name": bson.RegEx{"^X", ""}}).Distinct("module", &primparents)
	if err != nil {
		log.Fatal(err)
	}

	var prmpwg sync.WaitGroup
	prmpjobs := make(chan string, 100)

	for i := 0; i < threads; i++ {
		go prmpUpdateWorker(&prmpwg, prmpjobs)
		prmpwg.Add(1)
	}

	// Remove the modules with X instantiations inside them from the universe
	// to identify primitive parents.
	prmps := allm.Not(set.New(primparents...)).List()
	total = len(prmps)
	count = 0

	// Loop over each primitive parent that was found and add to the updaters
	// pool
	for _, prmp := range prmps {
		prmpjobs <- prmp
		count++
		log.Printf("prmp: (%d/%d) %s", count, total, prmp)
	}
	close(prmpjobs)

	prmpwg.Wait()

	////////////////////////////////////////////////////////////////////////////

	log.Println("Marking sequentials..")

	clog, err := session.DB("sart").C(cache+"_insts").UpdateAll(
		// everything that starts with ec0f or ec0l
		bson.M{"type": bson.RegEx{seqre, ""}}, // Selector interface
		bson.M{"$set": bson.M{"isseq": true}}, // Updater  interface
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Done. Found:", clog.Matched, "; Updated:", clog.Updated)

	////////////////////////////////////////////////////////////////////////////
	// Next, the instance connections' type field need to be updated to reflect
	// the direction -- input, output or inout. This information was captured
	// when the module (subckt) definition of each instance was discovered. At
	// this point it is already saved in mongo. It has to be done in this
	// manner -- a two-pass sort of way, beacuse all module (subckt)
	// definitions are not held in memory at the discovery phase.

	// Setup worker pool for conn type update queries //////////////////////////

	var connTypeUpdateWg sync.WaitGroup
	connTypeUpdateJobs := make(chan connTypeUpdateJob, 100)

	for i := 0; i < threads; i++ {
		go connTypeUpdateWorker(&connTypeUpdateWg, connTypeUpdateJobs)
		connTypeUpdateWg.Add(1)
	}

	////////////////////////////////////////////////////////////////////////////

	log.Println("Marking conn outputs and inouts..")

	findq := session.DB("sart").C(cache + "_ports").Find(
		bson.M{
			"$or": []bson.M{
				bson.M{"type": "OUTPUT"},
				bson.M{"type": "INOUT"},
			},
		},
	)

	total, err = findq.Count()
	if err != nil {
		log.Fatal(err)
	}

	// As there are approximately 2x inputs relative to outputs and inouts
	// combiner, by default an rtl.Conn.Type is initialized with 'INPUT'. Find
	// all ports that are OUTPUTs and INOUTs (and their positions), iterate
	// over them and update all *connections* that match that condition.
	iter := findq.Select(bson.M{"_id": 0}).Iter()

	var result bson.M

	count = 0
	outcount := 0
	inocount := 0
	for iter.Next(&result) {
		module := result["module"].(string)
		pos := result["pos"].(int)
		ctype := result["type"].(string)

		connTypeUpdateJobs <- connTypeUpdateJob{
			Module: module,
			Pos:    pos,
			Type:   ctype,
		}

		switch ctype {
		case "OUTPUT":
			outcount++
		case "INOUT":
			inocount++
		}

		count++
		log.Printf("conn: (%d/%d) %d\t%s %s", count, total, pos, ctype, module)
	}

	close(connTypeUpdateJobs)
	connTypeUpdateWg.Wait()
	log.Printf("Done. Updated %d outputs and %d inouts", outcount, inocount)

	sess := session.Copy()
	conn := sess.DB("sart").C(cache + "_conns")

	xtors := []string{
		"n",
		"nsvt",
		"nx",
		"nxhvt",
		"nxsvt",
		"p",
		"psvt",
		"px",
		"pxhvt",
		"pxsvt",
	}

	for _, xtor := range xtors {
		sel := bson.M{"itype": xtor, "pos": 0}
		set := bson.M{"$set": bson.M{"type": "OUTPUT"}}

		ci, err := conn.UpdateAll(sel, set)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Updated %d outputs in prim %q", ci.Updated, xtor)
	}
}
