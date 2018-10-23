package bitfield

import "testing"

func TestNew(t *testing.T) {
	f := New(1)
	if f == nil {
		t.Errorf("Expecting valid pointer to BitField for requested non-zero size")
	}
}

// func TestZero(t *testing.T) {
// 	f := New(0)
// 	if f != nil {
// 		t.Errorf("Expecting nil pointer to BitField when zero size BitField is requested")
// 	}
// }

func TestSize(t *testing.T) {
	testcases := []struct {
		size int
		expl int
	}{
		// 0 is not allowed
		{1, 1},
		{2, 1},
		{7, 1},
		{8, 1},
		{9, 2},
		{15, 2},
		{16, 2},
		{17, 3},
	}

	for _, testcase := range testcases {
		f := New(testcase.size)
		l := f.length()
		if l != testcase.expl {
			t.Errorf("Expecting byte length %d for requested size %d. Got %d",
				testcase.expl, testcase.size, l)
		}
	}
}

func TestLocate(t *testing.T) {
	// Size is really immaterial here. locate only does math. Tecnically does
	// not need to be a method.
	f := New(1)

	testcases := []struct {
		pos     int
		exp_byt int
		exp_bit uint8
	}{
		{0, 0, 0},
		{1, 0, 1},
		{2, 0, 2},
		{7, 0, 7},
		{8, 1, 0},
		{15, 1, 7},
		{16, 2, 0},
		{100, 12, 4},
	}

	for _, testcase := range testcases {
		byt, bit := f.locate(testcase.pos)
		if byt != testcase.exp_byt || bit != testcase.exp_bit {
			t.Errorf("Expected byt:%d, bit:%d for pos:%d. Got byt:%d, bit:%d",
				testcase.exp_byt, testcase.exp_bit, testcase.pos, byt, bit)
		}
	}
}

func TestSet(t *testing.T) {
	testcases := []struct {
		pos      int
		exp_byte uint8
	}{
		{0, 0x1},
		{1, 0x2},
		{2, 0x4},
		{3, 0x8},
		{4, 0x10},
		{5, 0x20},
		{6, 0x40},
		{7, 0x80},
		{8, 0x1},
		{9, 0x2},
		{15, 0x80},
	}

	for _, testcase := range testcases {
		f := New(15)

		f.Set(testcase.pos)

		byt, _ := f.locate(testcase.pos)

		if f.Fields[byt] != testcase.exp_byte {
			t.Errorf("Exepecting byte 0x%x for pos:%d. Got 0x%x", testcase.exp_byte,
				testcase.pos, f.Fields[byt])
		}
	}
}

func TestSetMultiple(t *testing.T) {
	testcases := []struct {
		pos      []int
		exp_word uint16
	}{
		{[]int{0}, 0x1},
		{[]int{0, 1}, 0x3},
		{[]int{0, 1, 2}, 0x7},
		{[]int{0, 1, 2, 8}, 0x107},
		{[]int{0, 1, 2, 8, 15}, 0x8107},
	}

	for _, testcase := range testcases {
		f := New(16)
		// for _, p := range testcase.pos {
		f.Set(testcase.pos...)
		// }

		word := uint16(f.Fields[1])<<8 | uint16(f.Fields[0])

		if word != testcase.exp_word {
			t.Errorf("Expecting 0x%x for %v. Got 0x%x", testcase.exp_word,
				testcase.pos, word)
		}
	}
}

func TestUnset(t *testing.T) {
	testcases := []struct {
		setpos   []int
		unsetpos []int
		exp_word uint16
	}{
		{[]int{0, 1, 2, 8}, []int{1, 2}, 0x101},
		{[]int{0, 1, 2, 8, 15}, []int{8}, 0x8007},
	}

	for _, testcase := range testcases {
		f := New(20)

		f.Set(testcase.setpos...)

		f.Unset(testcase.unsetpos...)

		word := uint16(f.Fields[1])<<8 | uint16(f.Fields[0])

		if word != testcase.exp_word {
			t.Errorf("Expecting 0x%x for set:%v and unset:%v. Got 0x%x",
				testcase.exp_word, testcase.setpos, testcase.unsetpos, word)
		}
	}
}

func TestTest(t *testing.T) {
	testcases := []struct {
		positions []int
	}{
		{[]int{0}},
		{[]int{0, 1}},
		{[]int{0, 1, 2, 3, 4, 5, 6, 7}},
		{[]int{0, 1, 8}},
		{[]int{0, 1, 8, 17, 23}},
	}

	for _, testcase := range testcases {
		f := New(20)
		// for _, p := range testcase.positions {
		f.Set(testcase.positions...)
		// }

		setpositions := f.Test()

		if len(setpositions) != len(testcase.positions) {
			t.Errorf("Lengths mismatch. Expecting %d. Got %d",
				len(testcase.positions), len(setpositions))
		}

		for i, p := range setpositions {
			if testcase.positions[i] != p {
				t.Errorf("Expecting pos %d to be set. Appears to be unset.")
			}
		}
	}
}

func TestAllUnset(t *testing.T) {
	for _, testcase := range []struct {
		bits2set []int
		exp      bool
	}{
		{[]int{}, true},
		{[]int{0}, false},
		{[]int{0, 1}, false},
		{[]int{0, 8}, false},
		{[]int{8}, false},
	} {
		f := New(20)

		f.Set(testcase.bits2set...)

		if f.AllUnset() != testcase.exp {
			t.Errorf("Expecting AllUnset() to return %v when bits are set %v",
				testcase.exp, testcase.bits2set)
		}
	}
}

func TestString(t *testing.T) {
	for _, test := range []struct {
		bits []int
		exp string
	} {
		{[]int{0}, "010000"},
		{[]int{1}, "020000"},
		{[]int{1, 2}, "060000"},
		{[]int{1, 2, 8}, "060100"},
	} {
		f := New(20)

		f.Set(test.bits...)

		if f.String() != test.exp {
			t.Errorf("Expecting %q. Got %q", test.exp, f.String())
		}
	}
}
