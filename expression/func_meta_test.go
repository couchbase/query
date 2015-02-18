package expression

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/couchbase/query/util"
)

// Define the pattern for UUIDs - RFC 4122, version 4
var parseUUIDRegex = regexp.MustCompile(hexPattern)

const hexPattern = `^(urn\:uuid\:)?[\{(\[]?([A-Fa-f0-9]{8})-?([A-Fa-f0-9]{4})-?([1-5][A-Fa-f0-9]{3})-?([A-Fa-f0-9]{4})-?([A-Fa-f0-9]{12})[\]\})]?$`

func TestNewV4(t *testing.T) {
	u, err := util.UUID()
	if err != nil {
		t.Errorf("Unexpected error getting UUID: %s", err.Error())
	}
	if !parseUUIDRegex.MatchString(u) {
		t.Errorf("Expected string representation to be valid, given: %s", u)
	}
	fmt.Printf("\t UUID:  %s \n", u)
}

func TestNewV4_eval(t *testing.T) {
	uu := NewUuid()
	u, _ := uu.Evaluate(nil, nil)

	if !parseUUIDRegex.MatchString(u.Actual().(string)) {
		t.Errorf("Expected string representation to be valid, given: %s", u.Actual().(string))
	}

	fmt.Printf("\t UUID:  %v \n", u.Actual())

}
