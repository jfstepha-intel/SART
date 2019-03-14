package ace

import (
	"encoding/json"
	"io"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Regex struct {
	Module string `json:"module"`
	Name   string `json:"name"`
}

type AceStruct struct {
	Selector Regex   `json:"sel"`
	Rpavf    float64 `json:"rpavf"`
	Wpavf    float64 `json:"wpavf"`
}

func Load(reader io.Reader) (acestructs []AceStruct) {
	err := json.NewDecoder(reader).Decode(&acestructs)
	if err != nil {
		log.Fatal(err)
	}
	return
}
