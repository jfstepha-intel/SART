package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "os"
    "regexp"
    "sort"
    "strconv"
    "strings"

    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
)

var level int
var depth int

type Table map[string]WidthMap

func (t Table) Add(key string) {
    if _, found := t[key]; !found {
        t[key] = make(WidthMap)
    }
}

func (t Table) Print(prefix string) {
    keys := []string{}
    for key := range t {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    for _, key := range keys {
        // log.Println(key)
        // log.Println(t[key])
        t[key].Print(prefix)
    }
}

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

var seqre = regexp.MustCompile(`fa0[fl]`)
var rfsre = regexp.MustCompile(`fa7`)
var sramre = regexp.MustCompile(`x74hdc1bit2x2`)

var seq, rfs, com *log.Logger

func (m WidthMap) Print(prefix string) {
    keys := []string{}
    for key := range m {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    for _, key := range keys {
        str := fmt.Sprintf("%s %s %0.3f",
                           strings.Replace(prefix, "tnt__a0_18ww10d4__", "", -1),
                           strings.Replace(key,    "tnt__a0_18ww10d4__", "", -1),
                           m[key])
        switch {
            case seqre.MatchString(key):
                seq.Println(str)
            case sramre.MatchString(key): fallthrough
            case rfsre.MatchString(key):
                rfs.Println(str)
            default:
                com.Println(str)
        }
    }
}

func (m WidthMap) String() (str string) {
    keys := []string{}
    for key := range m {
        keys = append(keys, key)
    }
    sort.Strings(keys)
    for _, key := range keys {
        str += fmt.Sprintf("%s %0.3f\n", key, m[key])
    }
    // str += fmt.Sprintf("%0.3f,", m["n"])
    // str += fmt.Sprintf("%0.3f,", m["nsvt"])
    // str += fmt.Sprintf("%0.3f,", m["p"])
    // str += fmt.Sprintf("%0.3f,",  m["psvt"])
    // str += fmt.Sprintf("%0.3f,",  m["navt"])
    // str += fmt.Sprintf("%0.3f",  m["pavt"])
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

        // log.Println(key, value)
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
        log.Printf("%s%s,%v", prefix, name, widths)
    }

    level--
    return
}

var rfre = regexp.MustCompile(`fa7m`)
var sram = regexp.MustCompile(`x74hdc1bit2x2`)

func Traverse(path, name string) (table Table) {
    level++
    if level < 4 {
        // log.Println(path, name)
    }
    table = make(Table)

    // stdcell := false
    // if strings.HasPrefix(name, "tnt__a0_18ww10d4__fa0") {
    //     log.Println("stdcell")
    //     stdcell = true
    // }

    blackbox := false
    if  strings.HasPrefix(name, "tnt__a0_18ww10d4__fa0l") ||
        strings.HasPrefix(name, "tnt__a0_18ww10d4__fa0f") ||
        rfre.MatchString(name) ||
        sram.MatchString(name) ||
        false {
        // strings.HasPrefix(name, "tnt__a0_18ww10d4__fa7m") {
        // strings.HasPrefix(name, "tnt__a0_18ww10d4__m74p0rf") {
        // log.Println("blackbox")
        blackbox = true
    }

    table.Add(path)

    switch {
    case blackbox:
        // log.Println("Sequential", path, name)
        table[path].Add(name, 1)

    // case stdcell   :
    //     log.Println("StdCell", path, name)
    //     key := name+"/"+iname

    default:
        // log.Println("default branch")
        path += "/" + strings.TrimPrefix(name, "tnt__a0_18ww10d4__")
        table.Add(path)
        // log.Println("Traverse", path, name, stdcell)
        iter := session.DB("sart").C(cache+"_insts").Find(bson.M{"module": name}).Iter()
        var result bson.M

        // count := 0
        for iter.Next(&result) {
            // module := result["module"].(string)
            iname  := result["name"].(string)
            itype  := result["type"].(string)
            key := name+"/"+iname

            // log.Println(iname, itype, key)

            if XtorWidths.Has(key) {
                typ := itype
                // log.Println("found leaf", itype, XtorWidths[key])
                table[path].Add(typ, XtorWidths[key])
            } else {

                // log.Println("found subcell")

                subtable := Traverse(path, itype)
                // if stdcell {
                    for subname := range subtable {
                        // log.Println("got back", subname)
                        for k, v := range subtable[subname] {
                            // log.Println(subname, k, v)
                            // table.Add(subname)
                            // log.Println(path, k, v)
                            table[path].Add(k, v)
                        }
                    }
                // } else {
                //     log.Fatal("should not get here")
                // }
            }
        }
    }

    level --
    return
}

var session *mgo.Session
var cache   string

func main() {
    var server, top, hpath string

    flag.StringVar(&server, "server",  "localhost", "name of mongodb server")
    flag.StringVar(&cache,  "cache",   "",          "name of cache to save module info")
    flag.StringVar(&top,    "top",     "",          "name of instantiated top cell")
    flag.StringVar(&hpath,  "hier",    "",          "path to file with hierarchies to report")
    flag.IntVar(&depth,     "depth",   3,           "max depth to print output")

    flag.Parse()

    log.SetFlags(log.Lshortfile)
    log.SetFlags(0)

    if cache == "" || (top == "" && hpath == "") {
        flag.PrintDefaults()
        log.Fatal("Insufficient arguments")
    }

    if hpath != "" && top != "" {
        log.Fatal("Can't specify both -hier and -top")
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

    sfile, err := os.Create("seq.csv")
    if err != nil { log.Fatal(err) }
    seq = log.New(sfile, "", 0)

    rfile, err := os.Create("rfs.csv")
    if err != nil { log.Fatal(err) }
    rfs = log.New(rfile, "", 0)

    cfile, err := os.Create("com.csv")
    if err != nil { log.Fatal(err) }
    com = log.New(cfile, "", 0)

    if top != "" {
        table := Traverse("", top)
        table.Print(top)
        return
    }

    hfile, err := os.Open(hpath)
    if err != nil {
        log.Fatal(err)
    }

    scanner := bufio.NewScanner(hfile)

    for scanner.Scan() {
        line := scanner.Text()
        if len(line) == 0 {
            continue
        }

        parts := strings.Split(line, "/")
        top := parts[len(parts)-1]
        // log.Println(parts, top)

        table := Traverse("", top)
        table.Print(line)
    }
}
