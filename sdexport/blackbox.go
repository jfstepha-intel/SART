package main

type StringSet map[string]struct{}

func (s StringSet) Add(str string) {
	s[str] = struct{}{}
}

func (s StringSet) Has(str string) bool {
	if _, found := s[str]; found {
		return true
	}
	return false
}

var bblist = []string{
	"ariqd",
	"ariqm",
	"arratmf",
	"arratmi",
	"rordym",
}

var blackboxes StringSet

func init() {
	blackboxes = make(StringSet)

	for _, str := range bblist {
		blackboxes.Add(str)
	}
}
