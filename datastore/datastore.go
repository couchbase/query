//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

const SYSTEM_NAMESPACE = "#system"
const SYSTEM_NAMESPACE_NAME = "system"

// Datastore represents a cluster or single-node server.
type Datastore interface {
	Id() string                                                 // Id of this datastore
	URL() string                                                // URL to this datastore
	Info() Info                                                 // Secondary information about this datastore
	NamespaceIds() ([]string, errors.Error)                     // Ids of the namespaces contained in this datastore
	NamespaceNames() ([]string, errors.Error)                   // Names of the namespaces contained in this datastore
	NamespaceById(id string) (Namespace, errors.Error)          // Find a namespace in this datastore using the namespace's Id
	NamespaceByName(name string) (Namespace, errors.Error)      // Find a namespace in this datastore using the namespace's name
	Authorize(*auth.Privileges, *auth.Credentials) errors.Error // Perform authorization and return nil if successful
	AdminUser(string) (string, string, error)                   // Admin credentials for a specific node
	PreAuthorize(*auth.Privileges)                              // Transform privileges in the internal format
	CredsString(*auth.Credentials) string                       // Return name from credentials in http request
	GetUserUUID(*auth.Credentials) string                       // Returns user UUID for stats
	GetUserBuckets(*auth.Credentials) []string                  // Returns buckets user has access to for serverless accounting
	SetLogLevel(level logging.Level)                            // Set log level of in-process indexers
	Inferencer(name InferenceType) (Inferencer, errors.Error)   // Schema inference provider by name, e.g. INF_DEFAULT
	Inferencers() ([]Inferencer, errors.Error)                  // List of schema inference providers
	StatUpdater() (StatUpdater, errors.Error)                   // Statistics Updater
	UserInfo() (value.Value, errors.Error)                      // The users, and their roles. JSON data.
	GetUserInfoAll() ([]User, errors.Error)                     // Get information about all the users.
	PutUserInfo(u *User) errors.Error                           // Set information for a specific user.
	GetRolesAll() ([]Role, errors.Error)                        // Get all roles that exist in the system.

	AuditInfo() (*AuditInfo, errors.Error)
	ProcessAuditUpdateStream(callb func(uid string) error) errors.Error
	EnableStorageAudit(val bool)

	SetConnectionSecurityConfig(conSecConfig *ConnectionSecurityConfig) // Update TLS or node-to-node encryption settings.
	CreateSystemCBOStats(requestId string) errors.Error
	GetSystemCBOStats() (Keyspace, errors.Error)
	HasSystemCBOStats() (bool, errors.Error)
	GetSystemCollection(bucketName string) (Keyspace, errors.Error)
	CheckSystemCollection(bucketName, requestId string) errors.Error

	StartTransaction(stmtAtomicity bool, context QueryContext) (map[string]bool, errors.Error)
	CommitTransaction(stmtAtomicity bool, context QueryContext) errors.Error
	RollbackTransaction(stmtAtomicity bool, context QueryContext, sname string) errors.Error
	SetSavepoint(stmtAtomicity bool, context QueryContext, sname string) errors.Error
	TransactionDeltaKeyScan(keyspace string, conn *IndexConnection) // Keys of Delta keyspace
}

type Systemstore interface {
	Datastore
	PrivilegesFromPath(fullname string, keyspace string, privilege auth.Privilege, privs *auth.Privileges)
}

type AuditInfo struct {
	AuditEnabled    bool
	EventDisabled   map[uint32]bool
	UserAllowlisted map[UserInfo]bool
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
	Datastore() Datastore // The datastore that contains this namespace
	Id() string           // Id of this namespace
	Name() string         // Name of this namespace

	// For keyspaces that appear directly under namespaces, such as system keyspaces.
	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this namespace
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this namespace
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this namespace using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this namespace using the keyspace's name

	// For keyspaces that are more deeply nested.
	// Namespaces contain Buckets contain Scopes contain Keyspaces, which are collections.
	BucketIds() ([]string, errors.Error)             // Ids of the buckets contained in this namespace
	BucketNames() ([]string, errors.Error)           // Names of the buckets contained in this namespace
	BucketById(name string) (Bucket, errors.Error)   // Find a bucket in this namespace using the bucket's id
	BucketByName(name string) (Bucket, errors.Error) // Find a bucket in this namespace using the bucket's name

	// All keyspaces and buckets visible to the user
	Objects(credentials *auth.Credentials, preload bool) ([]Object, errors.Error) // All first level namespace objects
}

type Object struct {
	Id         string
	Name       string
	IsKeyspace bool
	IsBucket   bool
}

type VirtualNamespace interface {
	Namespace
	VirtualKeyspaceByName(path []string) (Keyspace, errors.Error)
}

type Bucket interface {
	Id() string
	Name() string
	AuthKey() string // Key of the object to be used for authorization purposes
	Uid() string     // unique key for the purpose of detecting object change

	NamespaceId() string
	Namespace() Namespace

	DefaultKeyspace() (Keyspace, errors.Error)     // Non nil if the bucket allows direct access
	ScopeIds() ([]string, errors.Error)            // Ids of the scopes contained in this bucket
	ScopeNames() ([]string, errors.Error)          // Names of the scopes contained in this bucket
	ScopeById(name string) (Scope, errors.Error)   // Find a scope in this bucket using the scope's id
	ScopeByName(name string) (Scope, errors.Error) // Find a scope in this bucket using the scope's name

	CreateScope(name string) errors.Error // Create a new scope
	DropScope(name string) errors.Error   // Drop a scope
}

type Scope interface {
	Id() string
	Name() string
	AuthKey() string // Key of the object to be used for authorization purposes

	BucketId() string
	Bucket() Bucket

	KeyspaceIds() ([]string, errors.Error)               // Ids of the keyspaces contained in this scope
	KeyspaceNames() ([]string, errors.Error)             // Names of the keyspaces contained in this scope
	KeyspaceById(name string) (Keyspace, errors.Error)   // Find a keyspace in this scope using the keyspace's id
	KeyspaceByName(name string) (Keyspace, errors.Error) // Find a keyspace in this scope using the keyspace's name

	CreateCollection(name string, with value.Value) errors.Error // Create a new collection
	DropCollection(name string) errors.Error                     // Drop a collection
}

// Keyspace is a map of key-value entries (typically key-document, but
// also key-counter, key-blob, etc.). Keys are unique within a
// keyspace.
type Keyspace interface {
	Id() string // Id of this keyspace

	// A word on why we need three names:
	// Name is just a name unique among the object tracked by the namespace or scope under which this object sits. Easy enough.

	// QualifiedName returns the unique path of the storage object supporting this keyspace. This is not necessarily the full
	// path of this object.
	// For instance, for couchbase buckets, the QualifiedName is the full path of the default collection for that bucket.
	// This is needed for things like the planner, or index advisor, or the dictionary cache - by using a consistent unique
	// full path, both SELECT * FROM bucket and SELECT * FROM default:bucket._default._default are optimized in the same way,
	// using statistics from the same underlying objects, etc.
	// ADVISE provides the same indexes whether used agains the bucket and the default collection
	// Delta table names generated are the same whether we use a bucket or its defeault collection.

	// AuthKey is used for a similar reason: we want to use the same RBAC role for a default collection and a bucket.
	// But while the KV uses default collections for storage, and default:bucket internally uses default:bucket._default._default,
	// cbauth uses buckets to authorize default collections, so to access default:bucket._default._default, you need query_select
	// on bucket, not bucket:_default:_default.
	// Also, cbauth does not support namespaces, so AuthKeys only have bucket, scope and collection.
	Name() string          // Name of this keyspace
	QualifiedName() string // Full path of the storage object supporting keyspace, including default or system names if implied
	AuthKey() string       // Key of the object to be used for authorization purposes

	Uid() string // unique key for the purpose of detecting object change

	// A keyspace is found either directly under a namespace or under a scope.
	// If the keyspace is directly under a namespace, the ScopeId() returns "" and Scope() returns nil,
	// but NamespaceId() and Namespace() return normal values.
	// If the keyspace is under a scope, NamespaceId() returns "" and Namespace() returns nil,
	// but ScopeId() and Scope() return normally.
	NamespaceId() string  // Id of the namespace that contains this keyspace
	Namespace() Namespace // Backpointer to namespace
	ScopeId() string      // Id of the scope that contains this keyspace
	Scope() Scope         // Backpointer to scope

	Stats(context QueryContext, which []KeyspaceStats) ([]int64, errors.Error) // Collect multiple stats at once (eg Count, Size)
	Count(context QueryContext) (int64, errors.Error)                          // count of all documents
	Size(context QueryContext) (int64, errors.Error)                           // size of all documents
	Indexer(name IndexType) (Indexer, errors.Error)                            // Indexer provider by name, e.g. VIEW or GSI; "" returns default Indexer
	Indexers() ([]Indexer, errors.Error)                                       // List of index providers

	// Used by both SELECT and DML statements
	Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
		context QueryContext, subPath []string) errors.Errors // Bulk key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	Insert(inserts value.Pairs, context QueryContext) (value.Pairs, errors.Errors) // Bulk key-value insert into this keyspace
	Update(updates value.Pairs, context QueryContext) (value.Pairs, errors.Errors) // Bulk key-value updates into this keyspace
	Upsert(upserts value.Pairs, context QueryContext) (value.Pairs, errors.Errors) // Bulk key-value upserts into this keyspace
	Delete(deletes value.Pairs, context QueryContext) (value.Pairs, errors.Errors) // Bulk key-value deletes from this keyspace

	Flush() errors.Error // For flush collection
	IsBucket() bool
	Release(close bool) // Release any resources held by this object
}

// sequential scan

type SeqScanRange struct {
	Start        []byte
	ExcludeStart bool
	End          []byte
	ExcludeEnd   bool
}

func (this *SeqScanRange) Equals(other *SeqScanRange) bool {
	return this.ExcludeStart == other.ExcludeStart && this.ExcludeEnd == other.ExcludeEnd &&
		bytes.Compare(this.Start, other.Start) == 0 && bytes.Compare(this.End, other.End) == 0
}

func (this *SeqScanRange) OverlapsWith(other *SeqScanRange) bool {
	return (bytes.Compare(this.Start, other.Start) <= 0 && bytes.Compare(this.End, other.Start) >= 0) ||
		(bytes.Compare(this.Start, other.End) <= 0 && bytes.Compare(this.End, other.End) >= 0) ||
		(bytes.Compare(other.Start, this.Start) <= 0 && bytes.Compare(other.End, this.Start) >= 0) ||
		(bytes.Compare(other.Start, this.End) <= 0 && bytes.Compare(other.End, this.End) >= 0)
}

func (this *SeqScanRange) MergeWith(other *SeqScanRange) bool {
	c := 0
	if bytes.Compare(other.Start, this.Start) < 0 {
		this.Start, this.ExcludeStart = other.Start, other.ExcludeStart
		c++
	}
	if bytes.Compare(other.End, this.End) > 0 {
		this.End, this.ExcludeEnd = other.End, other.ExcludeEnd
		c++
	}
	return c == 2
}

func (this *SeqScanRange) String() string {
	var b strings.Builder
	b.WriteRune('[')
	if this.ExcludeStart {
		b.WriteRune('-')
	} else {
		b.WriteRune('+')
	}
	for _, c := range this.Start {
		if c != 0xff && unicode.IsPrint(rune(c)) {
			b.WriteRune(rune(c))
		} else {
			b.WriteString(fmt.Sprintf("<%02x>", byte(c)))
		}
	}
	b.WriteRune(':')
	if this.ExcludeEnd {
		b.WriteRune('-')
	} else {
		b.WriteRune('+')
	}
	for _, c := range this.End {
		if c != 0xff && unicode.IsPrint(rune(c)) {
			b.WriteRune(rune(c))
		} else {
			b.WriteString(fmt.Sprintf("<%02x>", byte(c)))
		}
	}
	b.WriteRune(']')
	return b.String()
}

type SeqScanner interface {
	StartKeyScan(ranges []*SeqScanRange, offset int64, limit int64, ordered bool, timeout time.Duration, pipelineSize int,
		kvTimeout time.Duration, serverless bool) (interface{}, errors.Error)
	StopKeyScan(interface{}) (uint64, errors.Error)
	FetchKeys(interface{}, time.Duration) ([]string, errors.Error, bool)
}

type KeyspaceStats int

const (
	KEYSPACE_COUNT = KeyspaceStats(iota)
	KEYSPACE_SIZE
	KEYSPACE_MEM_SIZE
)

type KeyspaceMetadata interface {
	MetadataVersion() uint64 // A counter that shows the current version of the list of objects contained within
	MetadataId() string      // A unique identifier across all of the stores. We choose the path of the object
}

// Globally accessible Datastore instance
var _DATASTORE Datastore
var _SYSTEMSTORE Systemstore

func SetDatastore(datastore Datastore) {
	_DATASTORE = datastore
}

func GetDatastore() Datastore {
	return _DATASTORE
}

func SetSystemstore(systemstore Systemstore) {
	_SYSTEMSTORE = systemstore
}

func GetSystemstore() Systemstore {
	return _SYSTEMSTORE
}

func getNamespace(parts ...string) (Namespace, errors.Error) {
	var datastore Datastore

	l := len(parts)
	if l == 0 {
		return nil, errors.NewDatastoreInvalidPathError("empty path")
	}
	namespace := parts[0]
	if namespace == SYSTEM_NAMESPACE {
		datastore = _SYSTEMSTORE
	} else {
		datastore = _DATASTORE
	}

	if datastore == nil {
		return nil, errors.NewError(nil, "Datastore not set.")
	}

	// FIXME once SetDefaultNamespace is resolved, this should go
	if namespace == "" {
		namespace = "default"
	}

	return datastore.NamespaceById(namespace)
}

func GetKeyspace(parts ...string) (Keyspace, errors.Error) {
	ns, err := getNamespace(parts...)
	if err != nil {
		return nil, err
	}
	switch len(parts) {
	case 2:
		ks, err := ns.KeyspaceByName(parts[1])
		if err != nil {
			return nil, err
		} else {

			// check if the bucket supports collections
			bucket, ok := ks.(Bucket)
			if !ok {
				return ks, err
			}

			// and if it has a default collection
			dks, err := bucket.DefaultKeyspace()
			if err != nil {
				return nil, err
			} else if dks == nil {
				return ks, nil
			} else {
				return dks, nil
			}
		}
	case 4:
		bucket, err := ns.BucketByName(parts[1])
		if err != nil {
			return nil, err
		}
		scope, err := bucket.ScopeByName(parts[2])
		if err != nil {
			return nil, err
		}
		return scope.KeyspaceByName(parts[3])
	default:
		return nil, errors.NewDatastoreInvalidKeyspacePartsError(parts...)
	}

}

func GetScope(parts ...string) (Scope, errors.Error) {
	ns, err := getNamespace(parts...)
	if err != nil {
		return nil, err
	}
	switch len(parts) {
	case 3:
		b, err := ns.BucketByName(parts[1])
		if err != nil {
			return nil, err
		}
		return b.ScopeByName(parts[2])
	default:
		return nil, errors.NewDatastoreInvalidScopePartsError(parts...)
	}
}

func GetBucket(parts ...string) (Bucket, errors.Error) {
	ns, err := getNamespace(parts...)
	if err != nil {
		return nil, err
	}
	switch len(parts) {
	case 2:
		return ns.BucketByName(parts[1])
	default:
		return nil, errors.NewDatastoreInvalidBucketPartsError(parts...)
	}
}

func GetPath(keyspace Keyspace) []string {
	namespace := keyspace.NamespaceId()
	scope := keyspace.Scope()
	if scope == nil {
		return []string{namespace, keyspace.Name()}
	}

	// for the default collection we want the actual name of the collection
	return []string{namespace, scope.BucketId(), scope.Name(), keyspace.Id()}
}

func IndexQualifiedKeyspacePath(index Index) string {

	// there is an outside chance that a virtual index might not have an indexer associated
	// this is code is never called, but just in case
	if index.Indexer() == nil {
		collIdx, ok := index.(CollectionIndex)
		if ok {
			bucketId := collIdx.BucketId()
			scopeId := collIdx.ScopeId()
			if bucketId != "" && scopeId != "" {
				return "default:" + bucketId + "." + scopeId + "." + index.KeyspaceId()
			}
		}
		return "default:" + index.KeyspaceId()
	}

	// The code below could have been duplicated here, but this makes maintenance easier
	return IndexerQualifiedKeyspacePath(index.Indexer())
}

func IndexerQualifiedKeyspacePath(indexer Indexer) string {
	if indexer.Name() == SYSTEM {
		return string(SYSTEM) + ":" + indexer.KeyspaceId()
	}

	// FIXME currently indexers and indexes only support a type and not a namespace, hence we hardwire it
	namespace := "default"

	bucket := indexer.BucketId()
	scope := indexer.ScopeId()

	// we have a fully qualified path
	if bucket != "" && scope != "" {
		return namespace + ":" + bucket + "." + scope + "." + indexer.KeyspaceId()
	}

	// It must be a bucket, get the fully qualified name
	keyspace, err := GetKeyspace(namespace, indexer.KeyspaceId())

	// we couldn't find it, return a token name
	if err != nil {
		return namespace + ":" + indexer.KeyspaceId()
	}

	return keyspace.QualifiedName()
}

// These structures are generic representations of users and their roles.
// Very similar structures exist in primitives/couchbase, but to keep open the
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
	Target string
}

var NO_STRINGS = make([]string, 0)

// In serverless mode ,check if the bucket passed to the function is accessible to the user
// If the bucket is not accessible, return generic "Access Denied" error
// If the user is an Admin, then skip this check since Admin has access to all buckets
// If the namespace is "#system" the generic error message is not to be returned - since system namespace is documented, the existing error messages are allowed for tenant users
func CheckBucketAccess(credentials *auth.Credentials, e errors.Error, path []string, privs *auth.Privileges) errors.Error {

	if tenant.IsServerless() && !IsAdmin(credentials) {

		if len(path) == 0 {
			return nil
		}

		// if the query is to create a global inline/ external function, the generic error message isnt to be returned
		if privs != nil {
			if len(privs.List) == 1 {
				priv := privs.List[0].Priv

				if priv == auth.PRIV_QUERY_MANAGE_FUNCTIONS || priv == auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL {
					return nil
				}
			}
		}

		code := e.Code()
		namespace := path[0]
		if code == errors.E_DATASTORE_INVALID_BUCKET_PARTS || namespace == "#system" {
			return nil
		}

		// the buckets the user has access to
		userBuckets := GetUserBuckets(credentials)
		bucket := path[1]
		if len(userBuckets) == 0 {
			return errors.NewCbAccessDeniedError(bucket)
		}

		for _, b := range userBuckets {
			if bucket == b {
				return nil
			}
		}

		return errors.NewCbAccessDeniedError(bucket)
	}

	return nil
}
