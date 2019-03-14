package ace

import (
	"strings"
	"testing"
)

func TestOne(t *testing.T) {
	str := `[ { "sel": {"module": "ip73"}, "rpavf": 0.005, "wpavf": 1.0, "comment": "this" } ]`
	a := Load(strings.NewReader(str))
	if len(a) != 1 {
		t.Errorf("Expecting a slice with 1 AceStruct. Got %d", len(a))
	}
}

func TestTwo(t *testing.T) {
	str := `[
	{ "sel": {"module": "ip73"}, "rpavf": 0.005, "wpavf": 1.0, "comment": "this" },
	{
		"sel": {"module": "ip73", "name": "abc", "comment": "this is a comment"},
		"rpavf": 0.005,
		"wpavf": 1.0,
		"comment": "this"
	}
]`
	a := Load(strings.NewReader(str))
	if len(a) != 2 {
		t.Errorf("Expecting a slice with 2 AceStruct. Got %d", len(a))
	}
}

func TestThree(t *testing.T) {
	str := `
[

{
	"sel": {
		"module": "parcpmssmb\/Xcpmssmb\/Xpke\/Xu_pke_unit128_rf_mems\/Xpke_unit128_wrap_mem_pke_banka0_mem_shell_128x72\/Xram_row_0_col_0\/Xip743rfshpm1r1w128x72c1p1$",
		"name": "idinp0"
	},
	"rpavf": 0.0,
	"wpavf": 0.0,
	"comment": "Ignore pke slices other than 0"
},

{
	"sel": {
		"module": "parcpmssmb\/Xcpmssmb\/Xpke\/Xu_pke_unit128_rf_mems\/Xpke_unit128_wrap_mem_pke_banka0_mem_shell_128x72\/Xram_row_0_col_0\/Xip743rfshpm1r1w128x72c1p1$",
		"name": "odoutp0"
	},
	"rpavf": 0.0,
	"wpavf": 0.0,
	"comment": "Ignore pke slices other than 0"
}

]
`

	a := Load(strings.NewReader(str))
	if len(a) != 2 {
		t.Errorf("Expecting a slice with 2 AceStruct. Got %d", len(a))
	}
}
