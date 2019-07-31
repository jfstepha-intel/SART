package parsesp

import (
	"io/ioutil"
	"log"
	"os"
	"sart/rtl"
	"strings"
	"testing"

	mgo "gopkg.in/mgo.v2"
)

func init() {
	log.SetFlags(log.Lshortfile)
	log.SetOutput(os.Stdout)
	log.SetOutput(ioutil.Discard)

	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Fatal(err)
	}

	rtl.InitMgo(session, "test", true)
}

func Test1(t *testing.T) {
	New("test", strings.NewReader(
		// A very basic subckt
		`.SUBCKT test1 port
.ENDS`))
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
}

func Test2(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple ports and one instance
		`.SUBCKT test2 port1 port2
Minst1 a b c
.ENDS`))
}

func Test2a(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple ports separated by a line break
		`
.SUBCKT test2a port1 port2
+ port3
Minst1 a b c
.ENDS`))
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
}

func Test3(t *testing.T) {
	New("test", strings.NewReader(
		// Basic subckt with multiple instances
		`
.SUBCKT test3 port1 port2
Minst1 a b c
Minst2 a b c d
.ENDS`))
}

func Test4(t *testing.T) {
	New("test", strings.NewReader(
		// Multiple basic subckts
		`
.SUBCKT test4 port1 port2
Minst1 a b c
Minst2 a b c d
.ENDS

.SUBCKT test4 port1 port2
Minst1 a b c
Minst2 a b c d
.ENDS

`))
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
Minst1 a b c
Minst2 a b c d
.ENDS`))
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
Minst1 a b c
Minst2 a b c d
.ENDS`))
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
.SUBCKT test7a port1 port2
*************
Minst1 a b c prop1=0 prop2="string"
Minst2 a b prop=33 c d prop3=42
+ prop4=""
.ENDS`))
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
