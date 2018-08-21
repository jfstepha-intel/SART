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
	// arrp
	"ariqd",
	"ariqm",
	"arratmf",
	"arratmi",
	"rordym",

	// icup
	"ictlb",
	"iddftutc",

	// idbp
	"idbp_idb",

	// iecp
	"exdftutc",
	"iecp_exprf0",
	"iecp_exprf1",
	"iecp_exprf2",

	// mectp
	"agdftutc",
	"agutlb",
	"dcstd",
	"dctag",

	// medp
	"dcdata",
	"dcfill",
	"dcldrotate",
	"dcstrotate",

	// msup
	"mspatch",

	// pgp-s
	"pgcorebotp",
	"pgcoretopp",

	// bus
	"bbcp",
	"bpip",
	"busbotp",
	"bustopp",
	"bxqp",
	"filp",
	"infrp",
	"pllcorep",
	"pllifcp",

	// l2
	"l2hi2mpartp",
	"l2lo2mpartp",
	"l2pgramp",
}

var blackboxes StringSet

func init() {
	blackboxes = make(StringSet)

	for _, str := range bblist {
		blackboxes.Add(str)
	}
}
