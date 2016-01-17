//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

// Quick FNV1a hash to distribute strings across required cache buckets
// Using it instead of hash/fnv to avoid poinless memory allocation
func HashString(id string, hashes int) int {
	var h uint = 2166136261
	for _, c := range []byte(id) {
		h ^= uint(c)
		h *= 16777619
	}
	return int(h % uint(hashes))
}
