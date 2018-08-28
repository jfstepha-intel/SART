package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sart/rtl"
	"strings"

	mgo "gopkg.in/mgo.v2"
)

var uprefix = "tnt__a0_18ww02d6__"

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
	sname := strings.TrimPrefix(m.Name, uprefix)
	for k, v := range m.Flops {
		fmt.Fprintf(SEQ, "%s/%s,%s,%d\n", prefix, sname, strings.TrimPrefix(k, uprefix), v)
	}
	for k, v := range m.Latch {
		fmt.Fprintf(SEQ, "%s/%s,%s,%d\n", prefix, sname, strings.TrimPrefix(k, uprefix), v)
	}
	for k, v := range m.Regfs {
		fmt.Fprintf(REG, "%s/%s,%s,%d\n", prefix, sname, strings.TrimPrefix(k, uprefix), v)
	}
	for k, v := range m.Embbs {
		fmt.Fprintf(REG, "%s/%s,%s,%d\n", prefix, sname, strings.TrimPrefix(k, uprefix), v)
	}

	// For combinational logic, report sum of widths for each transistor type
	widths := make(map[string]float64)
	for cell, count := range m.Combs {
		for _, prop := range props[cell] {
			device := prop.Itype
			width := prop.Fval * float64(count)
			widths[device] += width
		}
	}
	for device, width := range widths {
		fmt.Fprintf(COM, "%s/%s,%s,%0.3f\n", prefix, sname, device, width)
	}
}

type ModuleTable map[string]*Module

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

func (t ModuleTable) Add(m *Module) {
	t[m.Name] = m
}

func (t ModuleTable) Print(top, prefix string) {
	var m *Module
	var ok bool

	if m, ok = t[top]; !ok {
		return
	}

	stop := strings.TrimPrefix(top, uprefix)
	if blackboxes.Has(stop) {
		a := NewModule(m.Module)

		t.Accumulate(top, a)
		a.Print(prefix)
		return
	}

	m.Print(prefix)

	prefix += "/" + stop
	for _, inst := range m.Insts {
		t.Print(inst.Type, prefix)
	}
}

var LUT ModuleTable

var dfxre = regexp.MustCompile("dfx")

func Count(m *rtl.Module, prefix string) {
	// log.Printf("%s%s", prefix, m.Name)
	// prefix += "|   "

	x := NewModule(m)

	for _, inst := range m.Insts {
		// log.Printf("%s%s %s", prefix, inst.Type, inst.Name)

		itype := strings.TrimPrefix(inst.Type, uprefix)

		switch {
		// case dfxre.MatchString(itype):
		// 	continue
		case strings.HasPrefix(itype, "m74"):
			x.Embbs[inst.Type]++
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
		case primparents.Has(inst.Type):
			x.Combs[inst.Type]++
			continue
		default:
			i := rtl.NewModule(inst.Type)
			i.Load()
			Count(i, prefix+"|   ")
		}
	}

	LUT[m.Name] = x
	log.Printf("%s%s", prefix, m.Name)
}

var SEQ, REG, COM io.Writer

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

	LoadWidths(session, cache)
	LoadPrimParents(session, cache)

	m := rtl.NewModule(top)

	m.Load()

	LUT = make(ModuleTable)

	Count(m, "")

	SEQ, err = os.Create("seq.csv")
	REG, err = os.Create("reg.csv")
	COM, err = os.Create("com.csv")

	log.Println("Report")
	LUT.Print(top, "")
}
