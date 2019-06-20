package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sart/rtl"
	"strings"
	"time"

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

	// Add up any transistors at this level.
	for _, prop := range props[m.Name] {
		// log.Println(prop)
		device := prop.Itype
		width := prop.Fval
		widths[device] += width
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
		if debug {
			log.Printf("%s%s %s", prefix, inst.Type, inst.Name)
		}

		t := MatchType(inst.Type)

		switch t {
		case "EBB":
			x.Embbs[inst.Type]++

		case "Reg":
			x.Regfs[inst.Type]++

		case "Flop":
			x.Flops[inst.Type]++

		case "Latch":
			x.Latch[inst.Type]++

		case "Comb":
			x.Combs[inst.Type]++

		case "Unknown":
			i := rtl.NewModule(inst.Type)
			i.Load()
			Count(i, prefix+"|   ")

		default:
			log.Fatalf("Function MatchType returned unknown type %q", t)
		}
	}

	LUT[m.Name] = x
	if !debug {
		log.Printf("%s%s", prefix, m.Name)
	}
}

var SEQ, REG, COM io.Writer

var debug bool

func main() {
	var server, cache, top, bbpath, tspec string

	flag.StringVar(&top, "top", "", "name of topcell to report")
	flag.StringVar(&cache, "cache", "", "name of mongo cache to retrieve module info from")
	flag.StringVar(&server, "server", "localhost", "name of mongo server (optional)")
	flag.StringVar(&bbpath, "bb", "", "name of file with list of names to blackbox")
	flag.StringVar(&tspec, "tspec", "", "path to json file with type specifications")

	flag.BoolVar(&debug, "debug", false, "turn on verbose mode for debug")

	flag.Parse()

	if top == "" || cache == "" {
		flag.PrintDefaults()
		log.Fatal("Insufficient arguments.")
	}

	if bbpath != "" {
		file, err := os.Open(bbpath)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				blackboxes.Add(line)
			}
		}
	}

	if tspec == "" {
		flag.PrintDefaults()
		log.Fatal("-tspec is required")
	} else {
		LoadSpec(tspec)
	}

	session, err := mgo.Dial(server)
	if err != nil {
		log.Fatal(err)
	}

	rtl.InitMgo(session, cache, false)

	log.SetFlags(0)
	if debug {
		log.SetFlags(log.Lshortfile)
	}
	log.SetOutput(os.Stdout)

	LoadWidths(session, cache)
	LoadPrimParents(session, cache)

	m := rtl.NewModule(top)

	m.Load()

	LUT = make(ModuleTable)

	start := time.Now()
	Count(m, "")
	log.Println("Finished counting. Time elapsed:", time.Since(start))

	SEQ, err = os.Create(cache + "_seq.csv")
	REG, err = os.Create(cache + "_reg.csv")
	COM, err = os.Create(cache + "_com.csv")

	log.Println("Report")
	LUT.Print(top, "")
}
