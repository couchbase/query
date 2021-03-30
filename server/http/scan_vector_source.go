//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package http

import (
	"strings"

	"github.com/couchbase/query/timestamp"
)

// Implements timestamp.ScanVectorSource.
type singleScanVectorSource struct {
	scan_vector timestamp.Vector
}

func (this *singleScanVectorSource) ScanVector(namespace_id string, keyspace_name string) timestamp.Vector {
	return this.scan_vector
}

func (this *singleScanVectorSource) Type() int32 {
	return timestamp.ONE_VECTOR
}

// Implements timestamp.ScanVectorSource.
// Visible because it is used in a test.
type ZeroScanVectorSource struct {
	empty scanVectorEntries
}

func (this *ZeroScanVectorSource) ScanVector(namespace_id string, keyspace_name string) timestamp.Vector {
	// Always return a vector of 0 entries.
	return &this.empty
}

func (this *ZeroScanVectorSource) Type() int32 {
	return timestamp.NO_VECTORS
}

type fullyQualifiedKeyspace struct {
	namespace string
	keyspace  string
}

// Implements timestamp.ScanVectorSource.
type multipleScanVectorSource struct {
	vector_map map[fullyQualifiedKeyspace]timestamp.Vector
}

func (this *multipleScanVectorSource) ScanVector(namespace_id string, keyspace_name string) timestamp.Vector {
	ns := fullyQualifiedKeyspace{namespace: namespace_id, keyspace: keyspace_name}
	ret, found := this.vector_map[ns]
	if found {
		return ret
	} else {
		return &scanVectorEntries{}
	}
}

func (this *multipleScanVectorSource) Type() int32 {
	return timestamp.VECTOR_MAP
}

func newMultipleScanVectorSource(default_namespace string, vector_map map[string]timestamp.Vector) *multipleScanVectorSource {
	full_map := make(map[fullyQualifiedKeyspace]timestamp.Vector)
	for k, v := range vector_map {
		// They input keys may be of form "keyspace" or "namespace:keyspace".
		if strings.Contains(k, ":") {
			parts := strings.SplitN(k, ":", 2)
			new_key := fullyQualifiedKeyspace{namespace: parts[0], keyspace: parts[1]}
			full_map[new_key] = v
		} else {
			new_key := fullyQualifiedKeyspace{namespace: default_namespace, keyspace: k}
			full_map[new_key] = v
		}
	}
	return &multipleScanVectorSource{vector_map: full_map}
}
