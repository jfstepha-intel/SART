package queue

import (
	"testing"
)

func TestPushLen(t *testing.T) {
	testcases := []struct {
		inp []int
		exp []int
	}{
		{[]int{}, []int{}},
		{[]int{1}, []int{1}},
		{[]int{1, 2}, []int{1, 2}},
		{[]int{1, 2, 3}, []int{1, 2, 3}},
	}

	for i, tc := range testcases {
		q := New()

		for _, v := range tc.inp {
			q.Push(v)
		}

		if q.Len() != len(tc.exp) {
			t.Errorf("Test %d: Expected length of %d. Got %d.", i, tc.exp, q.Len())
		}

		values := q.Values()

		if len(values) != len(tc.exp) {
			t.Errorf("Test %d: Expected length of %d. Got %d.", i, tc.exp, len(values))
		}

		for j, v := range values {
			if tc.exp[j] != v {
				t.Errorf("Test %d: Expected %v. Got %v.", i, tc.exp, values)
			}
		}
	}
}

func TestPop(t *testing.T) {
	testcases := []struct {
		inp []int
		exp []int
		val interface{}
	}{
		{[]int{}, []int{}, nil},
		{[]int{1}, []int{}, 1},
		{[]int{2, 2}, []int{2}, 2},
		{[]int{1, 2, 3}, []int{2, 3}, 1},
	}

	for i, tc := range testcases {
		q := New()

		for _, v := range tc.inp {
			q.Push(v)
		}

		v := q.Pop()

		if v != tc.val {
			t.Errorf("Test %d: Expected length of %d. Got %d.", i, tc.exp, q.Len())
		}

		if q.Len() != len(tc.exp) {
			t.Errorf("Test %d: Expected length of %d. Got %d.", i, tc.exp, q.Len())
		}

		values := q.Values()

		if len(values) != len(tc.exp) {
			t.Errorf("Test %d: Expected length of %d. Got %d.", i, tc.exp, len(values))
		}

		for j, v := range values {
			if tc.exp[j] != v {
				t.Errorf("Test %d: Expected %v. Got %v.", i, tc.exp, values)
			}
		}
	}
}

func TestEmpty(t *testing.T) {
	q := New()

	if !q.Empty() {
		t.Errorf("Expecting empty queue. Got non-empty.")
	}

	q.Push(1)

	if q.Empty() {
		t.Errorf("Expecting non-empty queue. Got empty.")
	}

	q.Pop()

	if !q.Empty() {
		t.Errorf("Expecting empty queue. Got non-empty.")
	}
}
