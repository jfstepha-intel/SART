package parsesp

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sart/rtl"
	"strings"
	"testing"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var session *mgo.Session
var insts *mgo.Collection
var ports *mgo.Collection
var conns *mgo.Collection

func wait() {
	time.Sleep(time.Millisecond * 5)
}

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.SetOutput(ioutil.Discard)

	var err error

	session, err = mgo.Dial("localhost")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to connect to mongodb on localhost:", err)
		os.Exit(1)
	}

	insts = session.DB("sart").C("test_insts")
	ports = session.DB("sart").C("test_ports")
	conns = session.DB("sart").C("test_conns")

	rtl.InitMgo(session, "test", true)
}

func Test1(t *testing.T) {
	New("test", strings.NewReader(
		// A very basic subckt
		`.SUBCKT test1 port
.ENDS`))

	wait()

	var result bson.M

	// Need to find one entry in the ports collection with module name test1
	err := ports.Find(bson.M{"module": "test1"}).One(&result)
	if err != nil {
		t.Errorf("Could not locate module test1 in ports: %v", err)
	}

	// The name field in the document must be 'port'
	if _, ok := result["name"]; !ok {
		t.Errorf("No name field found")
	}

	if result["name"] != "port" {
		t.Errorf("Expected port with name %q. Got: %q", "port", result["name"])
	}
}

func Test1a(t *testing.T) {
	New("test", strings.NewReader(
		// A very basic subckt with comments
		`* comment
*----
.SUBCKT test1a port
* comment
.ENDS
`))

	wait()

	var result bson.M

	// Need to find one entry in the ports collection with module name test1
	err := ports.Find(bson.M{"module": "test1a"}).One(&result)
	if err != nil {
		t.Errorf("Could not locate module test1a in ports: %v", err)
	}

	// The name field in the document must be 'port'
	if _, ok := result["name"]; !ok {
		t.Errorf("No name field found")
	}

	if result["name"] != "port" {
		t.Errorf("Expected port with name %q. Got: %q", "port", result["name"])
	}
}

func Test2(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple ports and one instance
		`.SUBCKT test2 port1 port2
Minst1 a b c
.ENDS`))

	wait()

	var result []bson.M

	// Need to ensure that two ports were recorded.
	n, err := ports.Find(bson.M{"module": "test2"}).Count()
	if err != nil {
		t.Errorf("Unable to run Count query: %v", err)
	}

	if n != 2 {
		t.Errorf("Expected to find 2 ports, got %d", n)
	}

	// Get both results and ensure that the positions and names are correct
	err = ports.Find(bson.M{"module": "test2"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module test2 in ports: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected to find 2 ports, got %d", len(result))
	}

	for i, r := range result {
		if r["pos"] != i {
			t.Errorf("Expected pos field to be '%d', got '%d'", i, r["pos"])
		}
		if r["name"] != fmt.Sprintf("port%d", i+1) {
			t.Errorf("Expected name field to be %q, got %q", fmt.Sprintf("port%d", i+1), r["name"])
		}
	}

	// There should also be one entry in the insts collection that matches test2
	err = insts.Find(bson.M{"module": "test2"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test2 docs in insts: %v", err)
	}

	// There should be exactly one instance
	if len(result) != 1 {
		t.Errorf("Expected to find 1 instance, got %d", len(result))
	}

	// With this name
	if result[0]["name"] != "Minst1" {
		t.Errorf("Expected to find name:Minst1, got %v", result)
	}

	// And this type
	if result[0]["type"] != "c" {
		t.Errorf("Expected to find type:c, got %v", result)
	}

	// There should also be two docs in the conn collection matching
	// module:test2
	err = conns.Find(bson.M{"module": "test2"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test2 docs in conns: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected to find 2 connections, got %d", len(result))
	}
}

func Test2a(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple ports separated by a line break
		`
.SUBCKT test2a port1 port2
+ port3
Minst1 a b c
.ENDS`))

	wait()

	var result []bson.M

	// Get all the port documents that match this test name
	err := ports.Find(bson.M{"module": "test2a"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module test2a in ports: %v", err)
	}

	// There must exactly be 3 ports
	if len(result) != 3 {
		t.Errorf("Expected to find 3 ports, got %d", len(result))
	}

	// Ensure their names match, too
	for i, r := range result {
		exp := fmt.Sprintf("port%d", i+1)
		if r["name"] != exp {
			t.Errorf("Expected port name to be %q, got %q", exp, r["name"])
		}
	}
}

func Test2b(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple ports separated by a line break immediately after the
		// module name, and with multiple line breaks
		`
.SUBCKT test2b 
+port1 port2
+ port3
Minst1 a b c moduletype
.ENDS`))

	wait()

	var result []bson.M

	// Get all the port documents that match this test name
	err := ports.Find(bson.M{"module": "test2b"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module test2b in ports: %v", err)
	}

	// There must exactly be 3 ports
	if len(result) != 3 {
		t.Errorf("Expected to find 3 ports, got %d", len(result))
	}

	// Ensure their names match, too
	for i, r := range result {
		exp := fmt.Sprintf("port%d", i+1)
		if r["name"] != exp {
			t.Errorf("Expected port name to be %q, got %q", exp, r["name"])
		}
	}
}

func Test3(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple instances
		`
.SUBCKT test3 port1 port2
Minst1 a b c type1
Minst2 a b c d type2
.ENDS`))

	wait()

	var result []bson.M

	err := insts.Find(bson.M{"module": "test3"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test3 docs in insts: %v", err)
	}

	// There should be exactly two instances
	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}

	// Ensure their names match, too
	for i, r := range result {
		exp := fmt.Sprintf("Minst%d", i+1)
		if r["name"] != exp {
			t.Errorf("Expected instance name to be %q, got %q", exp, r["name"])
		}

		exp = fmt.Sprintf("type%d", i+1)
		if r["type"] != exp {
			t.Errorf("Expected instance type to be %q, got %q", exp, r["type"])
		}
	}
}

func Test4(t *testing.T) {
	New("test", strings.NewReader(
		// Multiple basic subckts
		`
.SUBCKT test4 port1 port2
Minst1 a b c
Minst2 a b c d
.ENDS

.SUBCKT test4a port1 port2
Minst1 a b c
Minst2 a b c d
.ENDS

`))

	wait()

	var result []bson.M

	err := insts.Find(bson.M{"module": "test4"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test4 docs in insts: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}

	err = insts.Find(bson.M{"module": "test4a"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test4a docs in insts: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}
}

func Test5(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with empty port specifications
		`
.SUBCKT test5 port1 port2
* INPUT:
* OUTPUT:
* INOUT:
*************
Minst1 a b c type1
Minst2 a b c d type2
.ENDS`))

	wait()

	var result []bson.M

	err := insts.Find(bson.M{"module": "test5"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test5 docs in insts: %v", err)
	}

	// There should be exactly two instances
	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}

	// Ensure their names match, too
	for i, r := range result {
		exp := fmt.Sprintf("Minst%d", i+1)
		if r["name"] != exp {
			t.Errorf("Expected instance name to be %q, got %q", exp, r["name"])
		}

		exp = fmt.Sprintf("type%d", i+1)
		if r["type"] != exp {
			t.Errorf("Expected instance type to be %q, got %q", exp, r["type"])
		}
	}

}

func Test5a(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with valid port specifiers and line breaks
		`
.SUBCKT test5a port1 port2
* INPUT: a b c d
* OUTPUT:a b c d
*+ e f
* INOUT: a b c d
*+ e f
* +g
*************
Minst1 a b c type1
Minst2 a b c d type2
.ENDS`))
	
	wait()

	var result []bson.M

	// In this test we're just ensuring that the tokens in the comments section
	// got parsed correctly, made it past the instantiationas and wrapped up
	// correctly. We're not necessarily concerned about the port directions
	// just yet.

	err := insts.Find(bson.M{"module": "test5a"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test5a docs in insts: %v", err)
	}

	// There should be exactly two instances
	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}

	// Ensure their names match, too
	for i, r := range result {
		exp := fmt.Sprintf("Minst%d", i+1)
		if r["name"] != exp {
			t.Errorf("Expected instance name to be %q, got %q", exp, r["name"])
		}

		exp = fmt.Sprintf("type%d", i+1)
		if r["type"] != exp {
			t.Errorf("Expected instance type to be %q, got %q", exp, r["type"])
		}
	}
}

func Test6(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with line breaks in instantiations
		`
.SUBCKT test6 port1 port2
*************
Minst1 a b c
+ d e f mtype
Minst2 a b c d
.ENDS`))

	wait()

	var result []bson.M

	err := insts.Find(bson.M{"module": "test6"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate module:test6 docs in insts: %v", err)
	}

	// There should be exactly two instances
	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}

	// Query the connections. there should be exactly 6 of type mtype and Minst1
	err = conns.Find(bson.M{"module": "test6", "iname": "Minst1", "itype": "mtype"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate {module:test6, iname:Minst1} docs in insts: %v", err)
	}

	if len(result) != 6 {
		t.Errorf("Expected to find 6 instance, got %d", len(result))
	}

	// And 3 of type d for Minst2
	err = conns.Find(bson.M{"module": "test6", "iname": "Minst2", "itype": "d"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate {module:test6, iname:Minst2} docs in insts: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected to find 3 instance, got %d", len(result))
	}
}

func Test7(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with properties in instantiations
		`
.SUBCKT test7 port1 port2
*************
Minst1 a b c prop1=0
Minst2 a b c d
.ENDS`))

	wait()

	var result []bson.M

	err := conns.Find(bson.M{"module": "test7", "iname": "Minst1", "itype": "c"}).All(&result)
	if err != nil {
		t.Errorf("Could not locate {module:test7, iname:Minst1} docs in insts: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected to find 2 instance, got %d", len(result))
	}
}

func Test7a(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with properties in instantiations and line breaks
		`
.SUBCKT test7a port1 port2
*************
Minst1 a b c prop1=0 prop2="string"
Minst2 a b c d prop3=42
+ prop4=""
.ENDS`))
}

func Test7b(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with properties in instantiations and line breaks and between ids
		`
.SUBCKT test7b port1 port2
*************
Minst1 a b c prop1=0 prop2="string"
Minst2 a b prop=33 c d prop3=42
+ prop4=""
.ENDS`))
}

func Test7c(t *testing.T) {
	New("test", strings.NewReader(
		// subckt with properties in instantiations and line breaks and between ids
		`
.SUBCKT e8xlres0a0e1basxn2hnx n11 n22 
* INPUT: 
* OUTPUT: 
* INOUT: n11 n22 


************************
R+i_resa n11 n22 N=2 e8xrm0m1a_prim LEVEL=2 
.ENDS  e8xlres0a0e1basxn2hnx`))
}

func Test8(t *testing.T) {
	New("test", strings.NewReader(
		// other rare directives
		`
.PARAM param="1"
.param param=42
.GLOBAL vss
.SUBCKT test8 port1 port2
.connect a b
Minst1 a b c
.ENDS
.end`))
}

func Test9(t *testing.T) {
	New("test", strings.NewReader(
		// other rare directives
		`
.PARAM param="1"
.param param=42
.GLOBAL vss
.SUBCKT test9 port1 port2
.connect a b
Minst1 a b c prop=2
.ENDS
.end`))
}

func Test10(t *testing.T) {
	New("test", strings.NewReader(
		`
.SUBCKT test5 port1 port2
* INPUT:
* OUTPUT:
* INOUT:

*** R 2 of 4 ***  .C_O_N_N_E_C_T vccmodule_nom c2u_v_sense_pwr_mod_untimed_zynfwh

*************
Minst1 a b c
Minst2 a b c d
Xshim5p vccmodule_nom vss grtshim5p 
R2 vccmodule_nom c2u_v_sense_pwr_mod_untimed_zynfwh rm11m10 SPACER=20e-9 SPACEL=20e-9 L=1193e-9 W=472e-9
.ENDS`))
}
