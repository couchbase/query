//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package datastore provides a common datastore abstraction over storage
engines, such as Couchbase server, cloud, mobile, file, 3rd-party
databases and storage engines, etc.

The logical hierarchy for the query language is datastore -> namespace -> bucket
-> document. Indexes are also attached to buckets.

TODO: This hierarchy should be revisited and aligned with long-term
plans before query Beta / GA.

*/
package datastore

import (
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

// log channel for the datastore lifecycle
const CHANNEL = "DATASTORE"

// Datastore represents a cluster or single-node server.
type Datastore interface {
	Id() string                                            // Id of this datastore
	URL() string                                           // URL to this datastore
	NamespaceIds() ([]string, errors.Error)                // Ids of the namespaces contained in this datastore
	NamespaceNames() ([]string, errors.Error)              // Names of the namespaces contained in this datastore
	NamespaceById(id string) (Namespace, errors.Error)     // Find a namespace in this datastore using the namespace's Id
	NamespaceByName(name string) (Namespace, errors.Error) // Find a namespace in this datastore using the namespace's name
}

// Namespace represents a logical boundary that is within a datastore and above
// a keyspace. In the query language, a namespace is only used as a namespace
// to qualify keyspace names. No assumptions are made about namespaces and
// isolation, resource management, or any other concerns.
type Namespace interface {
	DatastoreId() string                                 // Id of the datastore that contains this namespace
	Id() string                                          // Id of this namespace
	Name() string                                        // Name of this namespace
	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this namespace
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this namespace
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this namespace using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this namespace using the keyspace's name
}

// Keyspace is a map of key-value entries (typically key-document, but
// also key-counter, key-blob, etc.). Keys are unique within a
// keyspace.
type Keyspace interface {
	NamespaceId() string                                             // Id of the namespace that contains this keyspace
	Id() string                                                      // Id of this keyspace
	Name() string                                                    // Name of this keyspace
	Count() (int64, errors.Error)                                    // Number of key-value entries in this keyspace
	Indexer(name IndexType) (Indexer, errors.Error)                  // Index provider by name, e.g. VIEW or GSI
	Indexers() ([]Indexer, errors.Error)                             // List of index providers
	IndexByPrimary() (PrimaryIndex, errors.Error)                    // Returns the server-recommended primary index
	CreatePrimaryIndex(using IndexType) (PrimaryIndex, errors.Error) // Create or return a primary index on this keyspace
	CreateIndex(name string, equalKey, rangeKey expression.Expressions,
		where expression.Expression, using IndexType) (Index, errors.Error) // Create a secondary index on this keyspace

	// These methods have been moved to Indexer and will be removed from here
	IndexIds() ([]string, errors.Error)            // Ids of the indexes defined on this keyspace
	IndexNames() ([]string, errors.Error)          // Names of the indexes defined on this keyspace
	IndexById(id string) (Index, errors.Error)     // Find an index on this keyspace using the index's id
	IndexByName(name string) (Index, errors.Error) // Find an index on this keyspace using the index's name
	Indexes() ([]Index, errors.Error)              // Returns all the indexes defined on this keyspace

	// Used by both SELECT and DML statements
	Fetch(keys []string) ([]AnnotatedPair, errors.Error) // Bulk key-value fetch from this keyspace
	//Fetch(keys []string, projection, filter expression.Expression) ([]AnnotatedPair, errors.Error) // Bulk key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts []Pair) ([]Pair, errors.Error)     // Bulk key-value insert into this keyspace
	Update(updates []Pair) ([]Pair, errors.Error)     // Bulk key-value updates into this keyspace
	Upsert(upserts []Pair) ([]Pair, errors.Error)     // Bulk key-value upserts into this keyspace
	Delete(deletes []string) ([]string, errors.Error) // Bulk key-value deletes from this keyspace

	Release() // Release any resources held by this object
}

// Key-value pair
type Pair struct {
	Key   string
	Value value.Value
}

// Key-value pair
type AnnotatedPair struct {
	Key   string
	Value value.AnnotatedValue
}

// Globally accessible Datastore instance
var _DATASTORE Datastore

func SetDatastore(datastore Datastore) {
	_DATASTORE = datastore
}

func GetDatastore() Datastore {
	return _DATASTORE
}

func GetKeyspace(namespace, keyspace string) (Keyspace, error) {
	datastore := GetDatastore()

	ns, err := datastore.NamespaceByName(namespace)
	if err != nil {
		return nil, err
	}

	return ns.KeyspaceByName(keyspace)
}
