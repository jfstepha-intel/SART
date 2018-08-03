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

    flag.StringVar(&cache,  "cache",  "",          "name of cache from which to fetch module info.")
    flag.StringVar(&top,    "top",    "",          "name of topcell on which to run sart")
    flag.StringVar(&ace,    "ace",    "",          "path to ace structs file")
    flag.StringVar(&logp,   "log",    "",          "path to file where log messages should be redirected")
    flag.StringVar(&server, "server", "localhost", "name of mongodb server")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

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

    netlist.InitMgo(session, cache, true)

    start := time.Now()
    log.Println(netlist.New("", top, top))

    netlist.DoneMgo()
    netlist.WaitMgo()
    log.Println("Elapsed:", time.Since(start))

    start = time.Now()
    n := netlist.NewNetlist(top)
    n.Load()
    log.Println("Elapsed:", time.Since(start))
    log.Println(n)

    log.Println("Starting walks..")
    n.WalkDown()
}
