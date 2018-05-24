package set

import (
    "fmt"
    "sort"
)

type Set map[string]struct{}

func New(elements ...interface{}) Set {
    set := make(Set)
    for _, e := range elements {
        set.Add(e.(string))
    }
    return set
}

func (set Set) Add(str string) {
    set[str] = struct{}{}
}

func (set Set) Has(str string) bool {
    if _, ok := set[str]; ok {
        return true
    }
    return false
}

func (set Set) List() (elements []string) {
    for element := range set {
        elements = append(elements, element)
    }
    return
}

func (set Set) Sort() (elements []string) {
    elements = set.List()
    sort.Strings(elements)
    return
}

func (a Set) Not(b Set) (c Set) {
    c = make(Set)
    for _, e := range a.List() {
        if !b.Has(e) {
            c.Add(e)
        }
    }
    return
}

func (set Set) String() (str string) {
    return fmt.Sprintf("%d", len(set.List()))
}
