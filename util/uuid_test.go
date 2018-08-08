//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"testing"
)

func TestUUID(t *testing.T) {
	u1, _ := UUID()
	u2, _ := UUID()

	v5u1, _ := UUIDV5(u1, u2)
	v5u2, _ := UUIDV5(u1, u2)

	if v5u1 != v5u2 {
		t.Errorf("Expected %v twice, got %v instead", v5u1, v5u2)
	}
}
