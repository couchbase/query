//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package curlTest

import (
	"os"
	"strings"
	"testing"
)

// Basic test on cobering indexes
func TestCover(t *testing.T) {

	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}
	runMatch("case_curl.json", false, false, start_cs(), t)
}
