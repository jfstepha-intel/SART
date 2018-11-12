package ace

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type AceStruct struct {
	Field string
	Regex string
	Rpavf float64
	Wpavf float64
}

func Load(reader io.Reader) (acestructs []AceStruct) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Skip whitespace lines
		if line == "" {
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Split by comma and expect to find exactly 3 parts
		parts := strings.Split(line, ",")
		if len(parts) != 4 {
			log.Fatal("Expecting 4 parts in ACE struct", line, parts)
		}

		field := strings.TrimSpace(parts[0])
		regex := strings.TrimSpace(parts[1])

		// Do a basic check to see that the regex is valid -- i.e. at least
		// compilable by Go
		_, err := regexp.Compile(regex)
		if err != nil {
			log.Fatal(err)
		}

		// Convert the number strings into float type before passing on
		rpace, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		if err != nil {
			log.Fatal(err)
		}

		wpace, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
		if err != nil {
			log.Fatal(err)
		}

		acestruct := AceStruct{field, regex, rpace, wpace}
		acestructs = append(acestructs, acestruct)
	}

	return
}
