package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "sort"
    "strconv"
    "strings"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

var level int
var depth int

type WidthMap map[string]float64

func (m WidthMap) AddUq(key string, val float64) {
    if m.Has(key) {
        log.Fatalln(key, "exists")
    }
    m[key] = val
}

func (m WidthMap) Add(key string, val float64) {
    m[key] += val
}

func (m WidthMap) Has(key string) bool {
    if _, found := m[key]; found {
        return true
    }
    return false
}

func (m WidthMap) String() (str string) {
    keys := []string{}
    for key := range m {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    for _, key := range keys {
        str += fmt.Sprintf("%s:%0.3f ", key, m[key])
    }
    return
}

var XtorWidths WidthMap

func LoadXtorWidths() {
    iter := session.DB("sart").C(cache+"_props").Find(bson.M{"key": "W"}).Select(bson.M{"_id":0}).Iter()

    var result bson.M

    for iter.Next(&result) {
        module := result["module"].(string)
        iname  := result["iname"].(string)
        val    := result["val"].(string)

        key := module + "/" + iname

        if !strings.HasSuffix(val, "u") {
            log.Fatal(val)
        }

        value, err := strconv.ParseFloat(strings.TrimSuffix(val, "u"), 64)
        if err != nil {
            log.Fatal(err)
        }

        XtorWidths.AddUq(key, value)
    }
}

func Load(prefix, name string) (widths WidthMap) {
    // log.Printf("%s%s", prefix, name)
    widths = make(WidthMap)

    iter := session.DB("sart").C(cache+"_insts").Find(bson.M{"module": name}).Iter()
    var result bson.M

    // count := 0
    for iter.Next(&result) {
        module := result["module"].(string)
        iname  := result["name"].(string)
        itype  := result["type"].(string)

        lkey := module + "/" + iname

        if XtorWidths.Has(lkey) {
            // log.Printf("%sFound:%s(%f), Adding to:%s", prefix, lkey, XtorWidths[lkey], itype)
            widths.Add(itype, XtorWidths[lkey])
        } else {
            sname := prefix + strings.TrimPrefix(name, "tnt__a0_18ww10d4__") + "/"
            level++
            lw := Load(sname, itype)
            for k, v := range lw {
                widths.Add(k, v)
            }
        }
    }

    if depth < 0 || level < depth {
        name = strings.TrimPrefix(name, "tnt__a0_18ww10d4__")
        log.Printf("%s%s %v", prefix, name, widths)
    }

    level--
    return
}

var session *mgo.Session
var cache   string

func main() {
    var server, top string

    flag.StringVar(&server, "server",  "localhost", "name of mongodb server")
    flag.StringVar(&cache,  "cache",   "",          "name of cache to save module info")
    flag.StringVar(&top,    "top",     "",          "name of instantiated top cell")
    flag.IntVar(&depth,     "depth",   3,           "max depth to print output")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

    if cache == "" || top == "" {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    log.SetOutput(os.Stdout)

    XtorWidths = make(WidthMap)

    // Connect to MongoDB //////////////////////////////////////////////////////

    var err error

    session, err = mgo.Dial(server)
    if err != nil {
        log.Fatal(err)
    }

    LoadXtorWidths()

    Load("", top)
    // widths := Load("", top)
    // log.Println(widths)
}
