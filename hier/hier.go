package main

import (
    "flag"
    "log"
    "os"
    // "strings"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"

    "sart/rtl"
)

type SafeCounterMap struct {
    block    string
    counters map[string]int
}

func Print(prefix string, level int, top string) {
    if upto > 0 && level > upto {
        return
    }

    // shorttop := top
    // if strings.HasPrefix(top, "tnt__a0_18ww10d4__") {
        // shorttop := 
        // shorttop = top
    // }

    // prefix += "/" + strings.TrimPrefix(top, "tnt__a0_18ww10d4__")
    prefix += "/" + top

    log.Println(prefix)

    // log.Printf("%s%s", prefix, shorttop)

    q := session.DB("sart").C(cache+"_insts").Find(bson.M{"module": top})

    iter := q.Iter()

    var inst bson.M
    for iter.Next(&inst) {
        if !inst["isprim"].(bool) {
            Print(prefix, level+1, inst["type"].(string))
        }
    }
}

var server, cache, top string 
var session *mgo.Session
// var threads int

var upto int

func main() {
    flag.StringVar(&server, "server",  "localhost", "name of mongodb server")
    flag.StringVar(&cache,  "cache",   "",          "name of cache to save module info")
    flag.StringVar(&top,    "top",     "",          "name of top cell to explore")
    flag.IntVar(&upto,      "upto",    1,           "depth to which hierarchy is sought. -1 for full hierarchy")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

    if cache == "" || top == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    var err error
    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }
    rtl.InitMgo(session, cache, false)

    log.SetOutput(os.Stdout)

    Print("", 0, top)
}
