//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package catalog provides a common catalog abstraction over storage
engines, such as Couchbase server, cloud, mobile, file, 3rd-party
databases and storage engines, etc.

The logical hierarchy for the query language is site -> namespace -> bucket
-> document. Indexes are also attached to buckets.

TODO: This hierarchy should be revisited and aligned with long-term
plans before query Beta / GA.

*/
package catalog

import (
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// log channel for the catalog lifecycle
const CHANNEL = "CATALOG"

// Site represents a cluster or single-node server.
type Site interface {
	Id() string                                            // Id of this site
	URL() string                                           // URL to this site
	NamespaceIds() ([]string, errors.Error)                // Ids of the namespaces contained in this site
	NamespaceNames() ([]string, errors.Error)              // Names of the namespaces contained in this site
	NamespaceById(id string) (Namespace, errors.Error)     // Find a namespace in this site using the namespace's Id
	NamespaceByName(name string) (Namespace, errors.Error) // Find a namespace in this site using the namespace's name
}

// Namespace represents a logical boundary that is within a site and above
// a keyspace. In the query language, a namespace is only used as a namespace
// to qualify keyspace names. No assumptions are made about namespaces and
// isolation, resource management, or any other concerns.
type Namespace interface {
	SiteId() string                                      // Id of the site that contains this namespace
	Id() string                                          // Id of this namespace
	Name() string                                        // Name of this namespace
	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this namespace
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this namespace
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this namespace using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this namespace using the keyspace's name
}

// Keyspace is a collection of key-value entries (typically
// key-document, but also key-counter, key-blob, etc.).
type Keyspace interface {
	NamespaceId() string                              // Id of the namespace that contains this keyspace
	Id() string                                       // Id of this keyspace
	Name() string                                     // Name of this keyspace
	Count() (int64, errors.Error)                     // Number of key-value entries in this keyspace
	IndexIds() ([]string, errors.Error)               // Ids of the indexes defined on this keyspace
	IndexNames() ([]string, errors.Error)             // Names of the indexes defined on this keyspace
	IndexById(id string) (Index, errors.Error)        // Find an index on this keyspace using the index's id
	IndexByName(name string) (Index, errors.Error)    // Find an index on this keyspace using the index's name
	IndexByPrimary() (PrimaryIndex, errors.Error)     // Returns the server-recommended primary index
	Indexes() ([]Index, errors.Error)                 // Returns all the indexes defined on this keyspace
	CreatePrimaryIndex() (PrimaryIndex, errors.Error) // Create or return a primary index on this keyspace

	CreateIndex(name string, equalKey, rangeKey expression.Expressions, using IndexType) (Index, errors.Error) // Create a secondary index on this keyspace

	// Used by both SELECT and DML statements
	Fetch(keys []string) ([]Pair, errors.Error)      // Bulk key-value fetch from this keyspace
	FetchOne(key string) (value.Value, errors.Error) // Single key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts []Pair) ([]Pair, errors.Error) // Bulk key-value insert into this keyspace
	Update(updates []Pair) ([]Pair, errors.Error) // Bulk key-value updates into this keyspace
	Upsert(upserts []Pair) ([]Pair, errors.Error) // Bulk key-value upserts into this keyspace
	Delete(deletes []string) errors.Error         // Bulk key-value deletes from this keyspace

	Release() // Release any query engine resources held by this object
}

// Key-value pair
type Pair struct {
	Key   string
	Value value.Value
}
