/*
Copyright 2014-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package expression

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/couchbase/query/util"
)

// Define the pattern for UUIDs - RFC 4122, version 4
var parseUUIDRegex = regexp.MustCompile(hexPattern)

const hexPattern = `^(urn\:uuid\:)?[\{(\[]?([A-Fa-f0-9]{8})-?([A-Fa-f0-9]{4})-?([1-5][A-Fa-f0-9]{3})-` +
	`?([A-Fa-f0-9]{4})-?([A-Fa-f0-9]{12})[\]\})]?$`

func TestNewBase64Decode(t *testing.T) {
	inEx := NewConstant([]interface{}{1, 2, 3})
	encEx := NewBase64Encode(inEx)
	rv, err := encEx.Evaluate(nil, nil)
	if err != nil {
		t.Errorf("Error %v returned by Base64Encode", err)
	}
	decEx := NewBase64Decode(encEx)
	rv, err = decEx.Evaluate(nil, nil)
	if inEx.Value().Collate(rv) != 0 {
		t.Errorf("Mismatch: received %v expected %v", rv, inEx.Value().Actual())
	}
}

func TestNewV4(t *testing.T) {
	u, err := util.UUIDV4()
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
