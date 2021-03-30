//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
