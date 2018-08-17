package main

import (
	"flag"
	"log"
	"os"
	"sart/rtl"
	"strings"

	mgo "gopkg.in/mgo.v2"
)

type Module struct {
	*rtl.Module
	Flops map[string]int
	Latch map[string]int
	Regfs map[string]int
	Embbs map[string]int
	Combs map[string]int
}

func NewModule(r *rtl.Module) *Module {
	m := &Module{
		Module: r,
		Flops:  make(map[string]int),
		Latch:  make(map[string]int),
		Regfs:  make(map[string]int),
		Embbs:  make(map[string]int),
		Combs:  make(map[string]int),
	}
    return m
}

func (m Module) Print(prefix string) {
	for k, v := range m.Flops {
		log.Printf("%sFLOP: %s %s %d", prefix, m.Name, k, v)
	}
	for k, v := range m.Latch {
		log.Printf("%sLATC: %s %s %d", prefix, m.Name, k, v)
	}
	for k, v := range m.Regfs {
		log.Printf("%sREGF: %s %s %d", prefix, m.Name, k, v)
	}
	for k, v := range m.Embbs {
		log.Printf("%sEMBB: %s %s %d", prefix, m.Name, k, v)
	}
	// for k, v := range m.Combs {
	// 	log.Printf("%sCOMB: %s %d", prefix, k, v)
	// }
}

func (t ModuleTable) Accumulate(top string, a *Module) {
	var m *Module
	var ok bool

	if m, ok = t[top]; !ok {
		return
	}

	for k, v := range m.Flops {
		a.Flops[k] += v
	}
	for k, v := range m.Latch {
		a.Latch[k] += v
	}
	for k, v := range m.Regfs {
		a.Regfs[k] += v
	}
	for k, v := range m.Embbs {
		a.Embbs[k] += v
	}
	for k, v := range m.Combs {
		a.Combs[k] += v
	}

	for _, inst := range m.Insts {
		t.Accumulate(inst.Type, a)
	}
}

type ModuleTable map[string]*Module

func (t ModuleTable) Add(m *Module) {
	t[m.Name] = m
}

func (t ModuleTable) Print(top, prefix string) {
	var m *Module
	var ok bool

	if m, ok = t[top]; !ok {
		return
	}

	stop := strings.TrimPrefix(top, "tnt__a0_18ww10d4__")
	if blackboxes.Has(stop) {
		a := NewModule(m.Module)
		//// a := &Module{
		//// 	Module: m.Module,
		//// 	Flops:  make(map[string]int),
		//// 	Latch:  make(map[string]int),
		//// 	Regfs:  make(map[string]int),
		//// 	Embbs:  make(map[string]int),
		//// 	Combs:  make(map[string]int),
		//// }

		t.Accumulate(top, a)
		a.Print(prefix)
		return
	}

	m.Print(prefix)

	prefix += "|   "
	for _, inst := range m.Insts {
		t.Print(inst.Type, prefix)
	}
}

var LUT ModuleTable

func Count(m *rtl.Module, prefix string) {
	log.Printf("%s%s", prefix, m.Name)
	prefix += "|   "

	x := NewModule(m)
	//// x := &Module{
	//// 	Module: m,
	//// 	Flops:  make(map[string]int),
	//// 	Latch:  make(map[string]int),
	//// 	Regfs:  make(map[string]int),
	//// 	Embbs:  make(map[string]int),
	//// 	Combs:  make(map[string]int),
	//// }

	for _, inst := range m.Insts {
		// log.Printf("%s%s %s", prefix, inst.Type, inst.Name)

		itype := strings.TrimPrefix(inst.Type, "tnt__a0_18ww10d4__")

		switch {
		case strings.HasPrefix(itype, "dfxoddi"):
			continue
		case strings.HasPrefix(itype, "ckdfxcoredop"):
			continue
		case strings.HasPrefix(itype, "m74"):
			log.Printf("%s%s", prefix, itype)
			continue
		case strings.HasPrefix(itype, "fa0f"):
			x.Flops[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa0l"):
			x.Latch[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa0"):
			x.Combs[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa7m"):
			x.Regfs[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa7f"):
			x.Flops[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa7l"):
			x.Latch[inst.Type]++
			continue
		case strings.HasPrefix(itype, "fa7"):
			x.Combs[inst.Type]++
			continue
		}

		i := rtl.NewModule(inst.Type)
		i.Load()
		Count(i, prefix)
	}

	LUT[m.Name] = x
}

func main() {
	var server, cache, top string

	flag.StringVar(&top, "top", "", "name of topcell to report")
	flag.StringVar(&cache, "cache", "", "name of mongo cache to retrieve module info from")
	flag.StringVar(&server, "server", "localhost", "name of mongo server (optional)")

	flag.Parse()

	if top == "" || cache == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments.")
	}

	session, err := mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}

	rtl.InitMgo(session, cache, false)

	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)

	m := rtl.NewModule(top)

	m.Load()

	LUT = make(ModuleTable)

	Count(m, "")

	log.Println("Report")
	LUT.Print(top, "")
}
