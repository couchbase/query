//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

// Implements timestamp.ScanVectorSource.
// Visible because it is used in a test.
type ZeroScanVectorSource struct {
	empty scanVectorEntries
}

func (this *ZeroScanVectorSource) ScanVector(namespace_id string, keyspace_name string) timestamp.Vector {
	// Always return a vector of 0 entries.
	return &this.empty
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
