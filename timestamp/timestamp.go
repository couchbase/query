//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
