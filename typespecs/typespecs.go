package typespecs

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"regexp"
)

type TypeSpec struct {
	Type  string `json:"type"`
	Regex string `json:"regex"`
	regex *regexp.Regexp
}

type TypeSpecs []*TypeSpec

func New(path string) TypeSpecs {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	filebytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	var ts TypeSpecs

	err = json.Unmarshal(filebytes, &ts)
	if err != nil {
		log.Fatal(err)
	}

	for _, ts := range ts {
		ts.regex = regexp.MustCompile(ts.Regex)
	}

	return ts
}

func (t TypeSpecs) Match(itype string) string {
	for _, ts := range t {
		if ts.regex.MatchString(itype) {
			return ts.Type
		}
	}

	return "Unknown"
}
