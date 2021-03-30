//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package util

import (
	"testing"
	"time"
)

type testVal struct {
	duration  time.Duration
	start     Qualifier
	end       Qualifier
	precision int
	capped    bool
	expected  string
}

var tests = [...]testVal{

	// invalid qualifiers
	{time.Duration(1), HOUR, DAY, 9, false, ""},
	{time.Duration(1), YEAR, DAY, 9, false, ""},

	// negative
	{time.Duration(-1), HOUR, FRACTION, 9, false, "-0:00:00.000000001"},

	// uncapped
	{time.Duration(1), HOUR, FRACTION, 9, false, "0:00:00.000000001"},
	{time.Duration(14582000000001), HOUR, FRACTION, 9, false, "4:03:02.000000001"},
	{time.Duration(532982000000001), DAY, FRACTION, 9, false, "6 04:03:02.000000001"},
	{time.Duration(532982000000001), HOUR, FRACTION, 9, false, "148:03:02.000000001"},
	{time.Duration(34214400000000000), YEAR, MONTH, 9, false, "1-01"},

	// capped
	{time.Duration(1), HOUR, FRACTION, 9, true, "00:00:00.000000001"},
	{time.Duration(14582000000001), HOUR, FRACTION, 9, true, "04:03:02.000000001"},
	{time.Duration(532982000000001), HOUR, FRACTION, 9, true, "24:00:00.000000000"},
	{time.Duration(34214400000000000), YEAR, MONTH, 9, true, "0001-01"},

	// smaller fraction
	{time.Duration(14582000000001), HOUR, FRACTION, 5, true, "04:03:02.00000"},
}

func TestInterval(t *testing.T) {

	for _, test := range tests {
		res := ToQualifiedInterval(test.duration, test.start, test.end, test.precision, test.capped)
		if res != test.expected {
			if res == "" {
				res = "nothing"
			}
			expected := test.expected
			if expected == "" {
				expected = "nothing"
			}
			t.Errorf("Expected %v, got %v", expected, res)
		}
	}
}
