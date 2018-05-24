package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "sync"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"

    "sart/set"
)

////////////////////////////////////////////////////////////////////////////////

func updateworker(wg *sync.WaitGroup, jobs <-chan string) {
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

////////////////////////////////////////////////////////////////////////////////

var cache string

var session *mgo.Session

func main() {
    var server string
    var err error

    flag.StringVar(&cache,  "cache",  "",          "name of cache from which to fetch module info.")
    flag.StringVar(&server, "server", "localhost", "name of mongodb server")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

    if cache == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }

    log.SetOutput(os.Stdout)

    // These are the modules for which a definition is available -- defined modules
    var defnmodules []interface{}
    err = session.DB("sart").C(cache + "_nodes").Find(nil).Distinct("module", &defnmodules)
    if err != nil {
        log.Fatal(err)
    }

    // These are modules that have been instantiated at least once -- instantiated modules
    var instmodules []interface{}
    err = session.DB("sart").C(cache + "_insts").Find(nil).Distinct("type", &instmodules)
    if err != nil {
        log.Fatal(err)
    }

    ////////////////////////////////////////////////////////////////////////////

    var wg sync.WaitGroup
    jobs := make(chan string, 100)

    for i := 0; i < 4; i++ {
        wg.Add(1)
        go updateworker(&wg, jobs)
    }

    ////////////////////////////////////////////////////////////////////////////

    out, err := os.Create("uniqprims.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer out.Close()

    ////////////////////////////////////////////////////////////////////////////

    defn := set.New(defnmodules...)
    inst := set.New(instmodules...)

    prims := inst.Not(defn).Sort()
    total := len(prims)
    count := 0

    // Instantiated, but not defined implies primitives
    for _, prim := range prims {
        jobs <- prim
        count ++
        log.Printf("prim: (%d/%d) %s", count, total, prim)
        fmt.Fprintln(out, prim)
    }
    close(jobs)
    log.Println()

    ////////////////////////////////////////////////////////////////////////////

    // Defined, but not instantiated could imply that these are candidates for
    // topcell; or they are simply not instantiated.
    for _, top := range defn.Not(inst).Sort() {
        log.Println("top?", top)
    }

    wg.Wait()
}
