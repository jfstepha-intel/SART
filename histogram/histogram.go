package histogram

import (
	"fmt"
)

type Histogram map[interface{}]int

func New() Histogram {
	return make(Histogram)
}

func (h Histogram) Add(obs interface{}) {
	h[obs]++
}

func (h Histogram) Merge(w Histogram) {
	for bin, count := range w {
		h[bin] += count
	}
}

func (h Histogram) String() (str string) {
	for bin, count := range h {
		str += fmt.Sprintf("%v: %d ", bin, count)
	}
	str = strings.TrimSuffix(str, " ")
	return
}
