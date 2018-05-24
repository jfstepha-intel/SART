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

////////////////////////////////////////////////////////////////////////////////

type MgoRunner interface {
    Run(*mgo.Session)
}

////////////////////////////////////////////////////////////////////////////////

type MgoJob struct {
    Collection string
    SeqName    string
}

// MgoJob implements MgoRunner
func (j MgoJob) Run(s *mgo.Session) {
    c := s.DB("sart").C(j.Collection)
    _, err := c.UpdateAll(
        bson.M{"type": j.SeqName},
        bson.M{"$set": bson.M{"isseq": true}},
    )
    if err != nil {
        log.Fatal(err)
    }
}

////////////////////////////////////////////////////////////////////////////////

func updateworker(wg *sync.WaitGroup, jobs <-chan MgoRunner) {
    s := session.Copy()
    for job := range jobs {
        job.Run(s)
    }
    wg.Done()
}

var cache string

var session *mgo.Session

func main() {
    var prims, server string
    var err error

    flag.StringVar(&cache,  "cache",  "",          "name of cache from which to fetch module info.")
    flag.StringVar(&prims,  "prims",  "",          "path to file with names of sequentials")
    flag.StringVar(&server, "server", "localhost", "name of mongodb server")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

    if cache == "" || prims == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }

    log.SetOutput(os.Stdout)

    ////////////////////////////////////////////////////////////////////////////

    jobs := make(chan MgoRunner, 100)
    var wg sync.WaitGroup

    for i := 0; i < 4; i++ {
        wg.Add(1)
        go updateworker(&wg, jobs)
    }

    ////////////////////////////////////////////////////////////////////////////

    file, err := os.Open(prims)
    if err != nil {
        log.Fatal(err)
    }

    seqs := set.New()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        seqs.Add(scanner.Text())
    }

    count := 0
    total := len(seqs.List())

    for _, seq := range seqs.List() {
        count++
        log.Printf("(%d/%d) %s", count, total, seq)
        jobs <- MgoJob{cache+"_insts", seq}
    }

    close(jobs)

    ////////////////////////////////////////////////////////////////////////////

    wg.Wait()
}
