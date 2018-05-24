package main

import (
    "bufio"
    "flag"
    "log"
    "os"
    "sync"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"

    "sart/set"
)

func updateworker(wg *sync.WaitGroup, jobs <-chan string) {
    s := session.Copy()
    c := s.DB("sart").C(cache+"_conns")
    for prim := range jobs {
        var results []interface{}
        err := c.Find(bson.M{"itype": prim}).Distinct("formal", &results)
        if err != nil {
            log.Fatal(err)
        }

        for _, result := range results {
            formal := result.(string)
            if outnames.Has(formal) {
                log.Println(prim, formal)
                _, err = c.UpdateAll(
                    bson.M{"itype": prim, "formal": formal}, // Selector
                    bson.M{"$set": bson.M{"isout": true}},
                )
                if err != nil {
                    log.Fatal(err)
                }
            }
        }
    }
    wg.Done()
}

////////////////////////////////////////////////////////////////////////////////

var cache string

var session *mgo.Session

var outnames = set.New()

////////////////////////////////////////////////////////////////////////////////

func main() {
    var outs, server string
    var err error

    flag.StringVar(&outs,   "outs",   "",          "path to file containing names of primitive-outputs.")
    flag.StringVar(&cache,  "cache",  "",          "name of cache from which to fetch module info.")
    flag.StringVar(&server, "server", "localhost", "name of mongodb server")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    // log.SetFlags(0)

    if cache == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }

    log.SetOutput(os.Stdout)

    ////////////////////////////////////////////////////////////////////////////

    jobs := make(chan string, 100)

    var wg sync.WaitGroup

    for i := 0; i < 4; i++ {
        wg.Add(1)
        go updateworker(&wg, jobs)
    }

    ////////////////////////////////////////////////////////////////////////////

    // Load output names
    file, err := os.Open("primouts.txt")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        name := scanner.Text()
        outnames.Add(name)
    }

    ////////////////////////////////////////////////////////////////////////////

    // Get names of primitives.

    var results []interface{}
    err = session.DB("sart").C(cache+"_insts").Find(bson.M{"isprim":true}).Distinct("type", &results)
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        jobs <- result.(string)
    }
    close(jobs)

    wg.Wait()
}
