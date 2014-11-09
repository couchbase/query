package expression

import (
	"fmt"
	"github.com/twinj/uuid"
	"regexp"
	"testing"
)

//Copied from twinj uuit_test [https://github.com/twinj/uuid]

var parseUUIDRegex = regexp.MustCompile(hexPattern)

const hexPattern = `^(urn\:uuid\:)?[\{(\[]?([A-Fa-f0-9]{8})-?([A-Fa-f0-9]{4})-?([1-5][A-Fa-f0-9]{3})-?([A-Fa-f0-9]{4})-?([A-Fa-f0-9]{12})[\]\})]?$`

func init() {
	uuid.SwitchFormat(uuid.CleanHyphen)
}

func TestNewV4(t *testing.T) {
	u := uuid.NewV4()
	if u.Version() != 4 {
		t.Errorf("Expected correct version %d, but got %d", 4, u.Version())
	}
	if u.Variant() != uuid.ReservedRFC4122 {
		t.Errorf("Expected RFC4122 variant %x, but got %x", uuid.ReservedRFC4122, u.Variant())
	}
	if !parseUUIDRegex.MatchString(u.String()) {
		t.Errorf("Expected string representation to be valid, given: %s", u.String())
	}
	fmt.Printf("\t UUID:  %s \n", u.String())
}

func TestNewV4_eval(t *testing.T) {
	uu := NewUuid()
	u, _ := uu.Evaluate(nil, nil)

	if !parseUUIDRegex.MatchString(u.Actual().(string)) {
		t.Errorf("Expected string representation to be valid, given: %s", u.Actual().(string))
	}

	fmt.Printf("\t UUID:  %v \n", u.Actual())

}
