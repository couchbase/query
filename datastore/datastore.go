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
	Id() string                                                                                          // Id of this datastore
	URL() string                                                                                         // URL to this datastore
	Info() Info                                                                                          // Secondary information about this datastore
	NamespaceIds() ([]string, errors.Error)                                                              // Ids of the namespaces contained in this datastore
	NamespaceNames() ([]string, errors.Error)                                                            // Names of the namespaces contained in this datastore
	NamespaceById(id string) (Namespace, errors.Error)                                                   // Find a namespace in this datastore using the namespace's Id
	NamespaceByName(name string) (Namespace, errors.Error)                                               // Find a namespace in this datastore using the namespace's name
	Authorize(*auth.Privileges, auth.Credentials, *http.Request) (auth.AuthenticatedUsers, errors.Error) // Perform authorization and return nil if successful
	CredsString(*http.Request) string                                                                    // return name from credentials in http request
	SetLogLevel(level logging.Level)                                                                     // Set log level of in-process indexers
	Inferencer(name InferenceType) (Inferencer, errors.Error)                                            // Schema inference provider by name, e.g. INF_DEFAULT
	Inferencers() ([]Inferencer, errors.Error)                                                           // List of schema inference providers
	StatUpdater() (StatUpdater, errors.Error)                                                            // Statistics Updater
	UserInfo() (value.Value, errors.Error)                                                               // The users, and their roles. JSON data.
	GetUserInfoAll() ([]User, errors.Error)                                                              // Get information about all the users.
	PutUserInfo(u *User) errors.Error                                                                    // Set information for a specific user.
	GetRolesAll() ([]Role, errors.Error)                                                                 // Get all roles that exist in the system.

	AuditInfo() (*AuditInfo, errors.Error)
	ProcessAuditUpdateStream(callb func(uid string) error) errors.Error

	SetConnectionSecurityConfig(conSecConfig *ConnectionSecurityConfig) // Update TLS or node-to-node encryption settings.
}

type AuditInfo struct {
	AuditEnabled    bool
	EventDisabled   map[uint32]bool
	UserWhitelisted map[UserInfo]bool
	Uid             string
}

type UserInfo struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// Secondary information about this datastore. None of these methods
// should change anything about the store. They are informational only.
type Info interface {
	Version() string
	Topology() ([]string, []errors.Error)
	Services(node string) (map[string]interface{}, []errors.Error)
}

// Namespace represents a logical boundary that is within a datastore and above
// a keyspace. In the query language, a namespace is only used as a namespace
// to qualify keyspace names. No assumptions are made about namespaces and
// isolation, resource management, or any other concerns.
type Namespace interface {
	DatastoreId() string // Id of the datastore that contains this namespace
	Id() string          // Id of this namespace
	Name() string        // Name of this namespace

	// For keyspaces that appear directly under namespaces, such as system keyspaces.
	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this namespace
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this namespace
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this namespace using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this namespace using the keyspace's name
	MetadataVersion() uint64                             // Current version of the metadata

	// For keyspaces that are more deeply nested.
	// Namespaces contain Buckets contain Scopes contain Keyspaces, which are collections.
	BucketIds() ([]string, errors.Error)             // Ids of the buckets contained in this namespace
	BucketNames() ([]string, errors.Error)           // Names of the buckets contained in this namespace
	BucketById(name string) (Bucket, errors.Error)   // Find a bucket in this namespace using the bucket's id
	BucketByName(name string) (Bucket, errors.Error) // Find a bucket in this namespace using the bucket's name
}

type VirtualNamespace interface {
	Namespace
	VirtualKeyspaceByName(name string) (Keyspace, errors.Error)
}

type Bucket interface {
	Id() string
	Name() string

	NamespaceId() string
	Namespace() Namespace

	ScopeIds() ([]string, errors.Error)            // Ids of the scopes contained in this bucket
	ScopeNames() ([]string, errors.Error)          // Names of the scopes contained in this bucket
	ScopeById(name string) (Scope, errors.Error)   // Find a scope in this bucket using the scope's id
	ScopeByName(name string) (Scope, errors.Error) // Find a scope in this bucket using the scope's name
}

type Scope interface {
	Id() string
	Name() string

	BucketId() string
	Bucket() Bucket

	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this scope
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this scope
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this scope using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this scope using the keyspace's name
}

// Keyspace is a map of key-value entries (typically key-document, but
// also key-counter, key-blob, etc.). Keys are unique within a
// keyspace.
type Keyspace interface {
	Id() string   // Id of this keyspace
	Name() string // Name of this keyspace

	// A keyspace is found either directly under a namespace or under a scope.
	// If the keyspace is directly under a namespace, the ScopeId() returns "" and Scope() returns nil,
	// but NamespaceId() and Namespace() return normal values.
	// If the keyspace is under a scope, NamespaceId() returns "" and Namespace() returns nil,
	// but ScopeId() and Scope() return normally.
	NamespaceId() string  // Id of the namespace that contains this keyspace
	Namespace() Namespace // Backpointer to namespace
	ScopeId() string      // Id of the scope that contains this keyspace
	Scope() Scope         // Backpointer to scope

	Count(context QueryContext) (int64, errors.Error) // count of all documents
	Size(context QueryContext) (int64, errors.Error)  // size of all documents
	Indexer(name IndexType) (Indexer, errors.Error)   // Indexer provider by name, e.g. VIEW or GSI; "" returns default Indexer
	Indexers() ([]Indexer, errors.Error)              // List of index providers

	// Used by both SELECT and DML statements
	Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context QueryContext, subPath []string) []errors.Error // Bulk key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value insert into this keyspace
	Update(updates []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value updates into this keyspace
	Upsert(upserts []value.Pair) ([]value.Pair, errors.Error)               // Bulk key-value upserts into this keyspace
	Delete(deletes []string, context QueryContext) ([]string, errors.Error) // Bulk key-value deletes from this keyspace

	Release(close bool) // Release any resources held by this object
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

	if namespace == "" {
		namespace = "default"
	}

	ns, err := datastore.NamespaceByName(namespace)
	if err != nil {
		return nil, err
	}

	return ns.KeyspaceByName(keyspace)
}

// These structures are generic representations of users and their roles.
// Very similar structures exist in go-couchbase, but to keep open the
// possibility of connecting to other back ends, the query engine
// uses its own representation.
type User struct {
	Name   string
	Id     string
	Domain string
	Roles  []Role
}

type Role struct {
	Name   string
	Bucket string
}

var NO_STRINGS = make([]string, 0)
