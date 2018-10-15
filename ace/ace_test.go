package ace

import (
	"strings"
	"testing"
)

func TestWhitespace(t *testing.T) {
	str := `

  
	`
	a := Load(strings.NewReader(str))
	if len(a) != 0 {
		t.Errorf("Expecting to get a slice with 0 AceStruct. Got %d", len(a))
	}
}

func TestPoundComments(t *testing.T) {
	a := Load(strings.NewReader("# Comment"))
	if len(a) != 0 {
		t.Errorf("Expecting to get a slice with 0 AceStruct. Got %d", len(a))
	}
}

func TestSlashComments(t *testing.T) {
	a := Load(strings.NewReader("// Comment"))
	if len(a) != 0 {
		t.Errorf("Expecting to get a slice with 0 AceStruct. Got %d", len(a))
	}
}

func TestLineWithSpace(t *testing.T) {
	str := `module, Xauto_vector, 0.014188, 0.01378`
	a := Load(strings.NewReader(str))
	if len(a) != 1 {
		t.Errorf("Expecting to get a slice with 1 AceStruct. Got %d", len(a))
	}
}

func TestMultiline(t *testing.T) {
	str := `  // Comment

module, Xauto_vector_1,0.014188,0.01378

	name,Xauto_vector_2,0.018747,0.01861

  module,Xauto_vector_3,0.018747,0.01861
	# another comment
	`
	a := Load(strings.NewReader(str))
	if len(a) != 3 {
		t.Errorf("Expecting to get a slice with 3 AceStruct. Got %d", len(a))
	}
}
