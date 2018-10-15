package main

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

var typespecs []*TypeSpec

func LoadSpec(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	filebytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(filebytes, &typespecs)
	if err != nil {
		log.Fatal(err)
	}

	for _, ts := range typespecs {
		ts.regex = regexp.MustCompile(ts.Regex)
	}
}

func MatchType(itype string) string {
	for _, ts := range typespecs {
		if ts.regex.MatchString(itype) {
			return ts.Type
		}
	}

	if primparents.Has(itype) {
		return "Comb"
	}

	return "Unknown"
}
