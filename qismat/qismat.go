package main

import (
    "flag"
    "log"
    "os"
    "strings"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

type Instance struct {
    Path     string
    SeqCount map[string]int // Sequentials
    CmaCount map[string]int // RF arrays created as CMAs
    ComCount map[string]int // Combinationa logic gates
    Children []*Instance
}

func (i *Instance) AddSeq(name string) {
    if _, ok := i.SeqCount[name]; ok {
        i.SeqCount[name]++
        return
    }
    i.SeqCount[name] = 1
}

func (i *Instance) AddCma(name string) {
    if _, ok := i.CmaCount[name]; ok {
        i.CmaCount[name]++
        return
    }
    i.CmaCount[name] = 1
}

func (i *Instance) AddCom(name string) {
    if _, ok := i.ComCount[name]; ok {
        i.ComCount[name]++
        return
    }
    i.ComCount[name] = 1
}

func (i *Instance) AddChild(c *Instance) {
    i.Children = append(i.Children, c)
}

func (i Instance) PrintSeq() {
    for seq, count := range i.SeqCount {
        log.Printf("%s,%s,%d", i.Path, seq, count)
    }

    for _, child := range i.Children {
        child.PrintSeq()
    }
}

func (i Instance) PrintCma() {
    for cma, count := range i.CmaCount {
        log.Printf("%s,%s,%d", i.Path, cma, count)
    }

    for _, child := range i.Children {
        child.PrintCma()
    }

}

func (i Instance) PrintCom() {
    for com, count := range i.ComCount {
        log.Printf("%s,%s,%d", i.Path, com, count)
    }

    for _, child := range i.Children {
        child.PrintCom()
    }

}

func (i Instance) IsEmpty() bool {
    numseqs := 0
    for range i.SeqCount {
        numseqs++
    }

    numcmas := 0
    for range i.CmaCount {
        numcmas++
    }

    numcoms := 0
    for range i.ComCount {
        numcoms++
    }

    if len(i.Children) == 0 && numseqs == 0 && numcmas == 0 && numcoms == 0 {
        return true
    }
    return false
}

func Load(prefix, name string) *Instance {

    prefix += "/" + name

    inst := &Instance{
        Path    : prefix,
        SeqCount: make(map[string]int),
        CmaCount: make(map[string]int),
        ComCount: make(map[string]int),
        Children: []*Instance{},
    }
    
    iter := session.DB("sart").C(cache+"_insts").Find(bson.M{"module": name}).Iter()

    var i bson.M

    for iter.Next(&i) {
        itype := i["type"].(string)
        if i["isseq"].(bool) {
            inst.AddSeq(itype)
        } else if strings.HasPrefix(itype, "m74") {
            inst.AddCma(itype)
        } else if i["isprim"].(bool) && strings.HasPrefix(itype, "ec0") {
            inst.AddCom(itype)
        } else if i["isprim"].(bool) && !strings.HasPrefix(itype, "ec0") {
            // Primitives have no children. If name does not start with ec0, it
            // means that these don't have netlists elaborated. These are most
            // likely full-custom EBBs.
            log.Println("EBB?:", itype, prefix, i["name"])
        } else {
            c := Load(prefix, itype)
            if c != nil {
                inst.AddChild(c)
            }
        }
    }

    if inst.IsEmpty() {
        return nil
    }

    return inst
}

////////////////////////////////////////////////////////////////////////////////

var session *mgo.Session
var cache   string

func main() {
    var server, top string

    flag.StringVar(&server, "server",  "localhost", "name of mongodb server")
    flag.StringVar(&cache,  "cache",   "",          "name of cache to save module info")
    flag.StringVar(&top,    "top",     "",          "name of instantiated top cell")

    flag.Parse()

    log.SetFlags(log.Lshortfile)

    if cache == "" || top == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    log.SetOutput(os.Stdout)

    // Connect to MongoDB //////////////////////////////////////////////////////

    var err error

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }

    inst := Load("", top)
    if inst != nil {
        log.SetFlags(0)
        inst.PrintSeq()
        inst.PrintCma()
        inst.PrintCom()
    } else {
        log.Println("Not found")
    }
}
