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
	"arrp_ardftutc",
	"rordym",

	// fpap
	"fpap_fpdftutc",

	// fpdp
	"fpdp_frfp",

	// icup
	"ictlb",
	"icup_iddftutc",

	// idbp
	"idbp_idb",

	// iecp
	"iecp_exdftutc",
	"iecp_exprf0",
	"iecp_exprf1",
	"iecp_exprf2",

	// mectp
	"agutlb",
	"dcstd",
	"dctag",
	"mectp_agdftutc",

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

	"dv15idvtallps",
	"dfxoddi",

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

	// analog blocks to black box
	"afscaip",
	"apdesdresistor",
	"apdl2c6pg",
	"bgcaip",
	"ckblspinecustom",
	"ckgclkphasedet",
	"dpllcaip",
	"lpldocaip",
	"pgatecaip",
	"pgmodh",
	"pgmodv",
	"tadcd",
	"thbackendcaip",
	"thdiodecaip",
	"thremotecaip",
	"trcpath",
}

var blackboxes StringSet

func init() {
	blackboxes = make(StringSet)

	for _, str := range bblist {
		blackboxes.Add(str)
	}
}
