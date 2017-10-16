//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package curlTest

import (
	"os"
	"strings"
	"testing"
)

// Basic test on cobering indexes
func TestCover(t *testing.T) {

	val := os.Getenv("GSI_TEST")
	if strings.ToLower(val) == "true" {
		runMatch("case_curl.json", start_cs(), t)
	}
}
