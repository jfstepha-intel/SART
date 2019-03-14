package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sart/typespecs"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Instance struct {
	Path     string
	SeqCount map[string]int // Sequentials
	CmaCount map[string]int // RF arrays created as CMAs
	RegCount map[string]int // RF bits
	ComCount map[string]int // Combinationa logic gates
	Children []*Instance
}

func (i *Instance) AddReg(name string) {
	if _, ok := i.RegCount[name]; ok {
		i.RegCount[name]++
		return
	}
	i.RegCount[name] = 1
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

func (i Instance) PrintReg(file io.Writer) {
	for reg, count := range i.RegCount {
		fmt.Fprintf(file, "%s,%s,%d\n", i.Path, reg, count)
	}

	for _, child := range i.Children {
		child.PrintReg(file)
	}
}

func (i Instance) PrintSeq(file io.Writer) {
	for seq, count := range i.SeqCount {
		fmt.Fprintf(file, "%s,%s,%d\n", i.Path, seq, count)
	}

	for _, child := range i.Children {
		child.PrintSeq(file)
	}
}

func (i Instance) PrintCma(file io.Writer) {
	for cma, count := range i.CmaCount {
		fmt.Fprintf(file, "%s,%s,%d\n", i.Path, cma, count)
	}

	for _, child := range i.Children {
		child.PrintCma(file)
	}

}

func (i Instance) PrintCom(file io.Writer) {
	for com, count := range i.ComCount {
		fmt.Fprintf(file, "%s,%s,%d\n", i.Path, com, count)
	}

	for _, child := range i.Children {
		child.PrintCom(file)
	}

}

func (i Instance) IsEmpty() bool {
	numregs := 0
	for range i.RegCount {
		numregs++
	}

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

	if len(i.Children) == 0 && numregs == 0 && numseqs == 0 && numcmas == 0 && numcoms == 0 {
		return true
	}
	return false
}

func Load(prefix, name string) *Instance {

	prefix += "/" + name

	inst := &Instance{
		Path:     prefix,
		SeqCount: make(map[string]int),
		CmaCount: make(map[string]int),
		RegCount: make(map[string]int),
		ComCount: make(map[string]int),
		Children: []*Instance{},
	}

	iter := session.DB("sart").C(cache + "_insts").Find(bson.M{"module": name}).Iter()

	var i bson.M

	for iter.Next(&i) {
		itype := i["type"].(string)
		switch ts.Match(itype) {
		case "Reg":
			inst.AddReg(itype)
		case "Flop":
			if !i["isseq"].(bool) {
				log.Printf("Classified as flop: %s", itype)
			}
			inst.AddSeq(itype)
		case "Latch":
			if !i["isseq"].(bool) {
				log.Printf("Classified as latch: %s", itype)
			}
			inst.AddSeq(itype)
		case "Comb":
			// log.Println("Comb:", itype)
			inst.AddCom(itype)
		case "Cma":
			// log.Println("Cma:", itype)
			inst.AddCma(itype)
		default:
			if i["isprim"].(bool) {
				log.Println("EBB?:", itype, prefix, i["name"])
				break
			}

			c := Load(prefix, itype)
			if c != nil {
				inst.AddChild(c)
			}
		}
		// if i["isseq"].(bool) {
		// 	inst.AddSeq(itype)
		// } else if ts.Match(itype) == "EBB" {
		// 	inst.AddCma(itype)
		// } else if i["isprim"].(bool) && ts.Match(itype) == "Comb" {
		// 	inst.AddCom(itype)
		// } else if i["isprim"].(bool) && !strings.HasPrefix(itype, "ec0") {
		// 	// Primitives have no children. If name does not start with ec0, it
		// 	// means that these don't have netlists elaborated. These are most
		// 	// likely full-custom EBBs.
		// 	log.Println("EBB?:", itype, prefix, i["name"])
		// } else {
		//	c := Load(prefix, itype)
		//	if c != nil {
		//		inst.AddChild(c)
		//	}
		// }
	}

	if inst.IsEmpty() {
		return nil
	}

	return inst
}

////////////////////////////////////////////////////////////////////////////////

var session *mgo.Session
var cache string

var ts typespecs.TypeSpecs

func main() {
	var server, top, tspec string

	flag.StringVar(&server, "server", "localhost", "name of mongodb server")
	flag.StringVar(&cache, "cache", "", "name of cache to save module info")
	flag.StringVar(&top, "top", "", "name of instantiated top cell")
	flag.StringVar(&tspec, "tspec", "", "path to json file with type specifications")

	flag.Parse()

	log.SetFlags(log.Lshortfile)

	if cache == "" || top == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments")
	}

	if tspec == "" {
		flag.PrintDefaults()
		log.Fatal("-tspec is required")
	} else {
		ts = typespecs.New(tspec)
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

		file, err := os.Create(top + ".csv")
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		fmt.Fprintln(file, "RegisterFiles")
		inst.PrintReg(file)
		fmt.Fprintln(file, "Sequentials")
		inst.PrintSeq(file)
		fmt.Fprintln(file, "CMAs")
		inst.PrintCma(file)
		fmt.Fprintln(file, "Combinational Logic")
		inst.PrintCom(file)
	} else {
		log.Println("Not found")
	}
}
