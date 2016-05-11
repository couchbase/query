//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

/*
Note: The input slices must be sorted beforehand.
*/
func SortedStringsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if a == nil || b == nil {
		return a == nil && b == nil
	}

	for i, _ := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
