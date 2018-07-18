package main

import (
    "flag"
    "log"
    "io/ioutil"
    "os"
    "strings"
    "sync"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"

    "sart/parse"
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

        if strings.HasSuffix(path, ".sp") {
            parsesp.New(path, file)
        } else {
            parse.New(path, file)
        }

        file.Close()
    }
    wg.Done()
}

func updateWorker(wg *sync.WaitGroup, jobs <-chan string) {
    sess := session.Copy()
    inst := sess.DB("sart").C(cache+"_insts")
    conn := sess.DB("sart").C(cache+"_conns")
    for itype := range jobs {
        _, err := inst.UpdateAll(
            bson.M{"type": itype}, // Selector
            bson.M{"$set": 
                bson.M{"isprim": true},
            },
        )
        if err != nil {
            log.Fatal(err)
        }

        _, err = conn.UpdateAll(
            bson.M{"itype": itype}, // Selector
            bson.M{"$set": 
                bson.M{"isprim": true},
            },
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
    var path, server string 
    var threads int

    flag.StringVar(&path,   "path",    "",          "path to folder with netlist files")
    flag.StringVar(&server, "server",  "localhost", "name of mongodb server")
    flag.StringVar(&cache,  "cache",   "",          "name of cache to save module info")
    flag.IntVar(&threads,   "threads", 2,           "number of parallel threads to spawn")

    flag.Parse()

    if path == "" || cache == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    log.SetFlags(log.Lshortfile)

    // Connect to MongoDB //////////////////////////////////////////////////////

    var err error

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }
    rtl.InitMgo(session, cache, true)

    log.SetOutput(os.Stdout)

    // Setup inputs, waitgroup and worker threads //////////////////////////////

    files, err := ioutil.ReadDir(path)
    if err != nil {
        log.Fatal(err)
    }

    var parsewg sync.WaitGroup
    parsejobs  := make(chan string, 100)

    for i := 0; i < threads; i++ {
        go parseWorker(&parsewg, parsejobs)
        parsewg.Add(1)
    }

    // Loop over files and add to parsers pool /////////////////////////////////

    count := 0
    total := len(files)
    for _, file := range files {
        filename := file.Name()
        count++
        
        if  !strings.HasSuffix(filename, ".v")  &&
            !strings.HasSuffix(filename, ".vg") && 
            !strings.HasSuffix(filename, ".sp") {
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

    return

    ////////////////////////////////////////////////////////////////////////////
    // At this point all available information in the input netlists have been
    // parsed, sliced and diced into wires, insts and conns collections in the
    // mongo database. Next we need to mark all the instantiations for which a
    // module definition was not found as primitives.
    ////////////////////////////////////////////////////////////////////////////

    // These are the modules for which a definition is available -- defined
    // modules
    var defnmodules []interface{}
    err = session.DB("sart").C(cache + "_wires").Find(nil).Distinct("module", &defnmodules)
    if err != nil {
        log.Fatal(err)
    }

    // These are modules that have been instantiated at least once --
    // instantiated modules
    var instmodules []interface{}
    err = session.DB("sart").C(cache + "_insts").Find(nil).Distinct("type", &instmodules)
    if err != nil {
        log.Fatal(err)
    }

    // Create sets out of these lists
    defn := set.New(defnmodules...)
    inst := set.New(instmodules...)

    // Setup worker pool for update queries ////////////////////////////////////

    var updatewg sync.WaitGroup
    updatejobs := make(chan string, 100)

    for i := 0; i < threads; i++ {
        go updateWorker(&updatewg, updatejobs)
        updatewg.Add(1)
    }

    // Instantiated, but not defined implies primitives or EBBs. Either way
    // they'll have to be treated as primitives as there is no information on
    // them to go by.

    prims := inst.Not(defn).Sort()
    total  = len(prims)
    count  = 0

    // Loop over each primitive that was found and add to the updaters pool
    for _, prim := range prims {
        updatejobs <- prim
        count ++
        log.Printf("prim: (%d/%d) %s", count, total, prim)
    }
    close(updatejobs)

    updatewg.Wait()

    ////////////////////////////////////////////////////////////////////////////

    log.Println("Marking sequentials..")

    clog, err := session.DB("sart").C(cache+"_insts").UpdateAll(
        // everything that starts with ec0f or ec0l
        bson.M{"type": bson.RegEx{"^ec0[fl]", ""}}, // Selector interface
        bson.M{"$set": bson.M{"isseq": true}},      // Updater  interface
    )

    if err != nil {
        log.Fatal(err)
    }

    log.Println("Done. Found:", clog.Matched, "; Updated:", clog.Updated)

    ////////////////////////////////////////////////////////////////////////////

    log.Println("Marking outputs..")

    outs := []string{
        "carry",
        "clkout",
        "o",
        "o1",
        "out0",
        "so",
        "sum",
    }

    for _, out := range outs {
        clog, err := session.DB("sart").C(cache+"_conns").UpdateAll(
            bson.M{"formal": out, "isprim": true},
            bson.M{"$set": bson.M{"isout": true}},
        )

        if err != nil {
            log.Fatal(err)
        }
        log.Printf("out: %s Found: %d, Updated: %d", out, clog.Matched, clog.Updated)
    }

    log.Println("Done.")
}
