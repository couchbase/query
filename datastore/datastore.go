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
	"net/http"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

// log channel for the datastore lifecycle
const CHANNEL = "DATASTORE"

// Datastore represents a cluster or single-node server.
type Datastore interface {
	Id() string                                                                           // Id of this datastore
	URL() string                                                                          // URL to this datastore
	NamespaceIds() ([]string, errors.Error)                                               // Ids of the namespaces contained in this datastore
	NamespaceNames() ([]string, errors.Error)                                             // Names of the namespaces contained in this datastore
	NamespaceById(id string) (Namespace, errors.Error)                                    // Find a namespace in this datastore using the namespace's Id
	NamespaceByName(name string) (Namespace, errors.Error)                                // Find a namespace in this datastore using the namespace's name
	Authorize(*Privileges, Credentials, *http.Request) (AuthenticatedUsers, errors.Error) // Perform authorization and return nil if successful
	SetLogLevel(level logging.Level)                                                      // Set log level of in-process indexers
	Inferencer(name InferenceType) (Inferencer, errors.Error)                             // Schema inference provider by name, e.g. INF_DEFAULT
	Inferencers() ([]Inferencer, errors.Error)                                            // List of schema inference providers
	UserInfo() (value.Value, errors.Error)                                                // The users, and their roles. JSON data.
	GetUserInfoAll() ([]auth.User, errors.Error)                                          // Get information about all the users.
	PutUserInfo(u *auth.User) errors.Error                                                // Set information for a specific user.
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

// A subset of execution.Context that is useful at the datastore level.
type QueryContext interface {
	Credentials() Credentials
	AuthenticatedUsers() []string
}

type QueryContextImpl struct {
}

func (ci *QueryContextImpl) Credentials() Credentials {
	return make(Credentials, 0)
}

func (ci *QueryContextImpl) AuthenticatedUsers() []string {
	return make([]string, 0, 16)
}

var NULL_QUERY_CONTEXT = &QueryContextImpl{}

// Keyspace is a map of key-value entries (typically key-document, but
// also key-counter, key-blob, etc.). Keys are unique within a
// keyspace.
type Keyspace interface {
	NamespaceId() string                              // Id of the namespace that contains this keyspace
	Id() string                                       // Id of this keyspace
	Name() string                                     // Name of this keyspace
	Count(context QueryContext) (int64, errors.Error) // Number of key-value entries in this keyspace
	Indexer(name IndexType) (Indexer, errors.Error)   // Indexer provider by name, e.g. VIEW or GSI; "" returns default Indexer
	Indexers() ([]Indexer, errors.Error)              // List of index providers

	// Used by both SELECT and DML statements
	Fetch(keys []string, context QueryContext) ([]value.AnnotatedPair, []errors.Error) // Bulk key-value fetch from this keyspace
	//Fetch(keys []string, projection, filter expression.Expression) ([]value.AnnotatedPair, errors.Error) // Bulk key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value insert into this keyspace
	Update(updates []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value updates into this keyspace
	Upsert(upserts []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value upserts into this keyspace
	Delete(deletes []string, context QueryContext) ([]string, errors.Error) // Bulk key-value deletes from this keyspace

	Release() // Release any resources held by this object
}

// Globally accessible Datastore instance
var _DATASTORE Datastore
var _SYSTEMSTORE Datastore

func SetDatastore(datastore Datastore) {
	_DATASTORE = datastore
}

func GetDatastore() Datastore {
	return _DATASTORE
}

func SetSystemstore(systemstore Datastore) {
	_SYSTEMSTORE = systemstore
}

func GetSystemstore() Datastore {
	return _SYSTEMSTORE
}

func GetKeyspace(namespace, keyspace string) (Keyspace, errors.Error) {
	var datastore Datastore

	if namespace == "#system" {
		datastore = GetSystemstore()
	} else {
		datastore = GetDatastore()
	}

	if datastore == nil {
		return nil, errors.NewError(nil, "Datastore not set.")
	}

	ns, err := datastore.NamespaceByName(namespace)
	if err != nil {
		return nil, err
	}

	return ns.KeyspaceByName(keyspace)
}
