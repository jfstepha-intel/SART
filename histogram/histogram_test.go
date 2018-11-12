package histogram

import (
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	h := New()
	if h == nil {
		t.Errorf("Expecting a non-nil histogram. Got nil.")
	}
}

func ExampleAdd() {
	h := New()

	h.Add(1)
	h.Add(1)
	h.Add(2)
	h.Add(2)
	h.Add(3)
	h.Add("hello")
	h.Add("hello")
	h.Add("hello")
	h.Add("hello")

	fmt.Println(h)

	// Unordered output:
	// 1: 2
	// 2: 2
	// 3: 1
	// hello: 4
}

func ExampleMerge() {
	h := New()

	h.Add(1)
	h.Add(1)
	h.Add(2)
	h.Add(2)
	h.Add(3)
	h.Add("hello")
	h.Add("hello")
	h.Add("hello")
	h.Add("hello")

	w := New()

	w.Add("hello")
	w.Add(4)

	h.Merge(w)

	fmt.Println(h)

	// Unordered output:
	// 1: 2
	// 2: 2
	// 3: 1
	// 4: 1
	// hello: 5
}
