//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*
Package timestmap provides logical vector timestamps.
*/
package timestamp

const (
	// Scan vectors not specified at all.
	NO_VECTORS = 1
	// Client gave us one vector, without saying what scan vector it is for. (scan_vector)
	ONE_VECTOR = 2
	// Client gave us a map of from keyspaces to scan vectors.
	VECTOR_MAP = 3
)

type Vector interface {
	Entries() []Entry // Non-zero entries; all missing entries are zero
}

type Entry interface {
	Position() uint32 // vbucket/partition index (0-based)
	Guard() string    // vbucket/partition validation UUID
	Value() uint64    // Logical sequence number
}

type ScanVectorSource interface {
	Type() int32
	ScanVector(namespace_id string, keyspace_name string) Vector
}
