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
	"sync"
	"time"
	"unicode"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const SYSTEM_NAMESPACE = "#system"
const SYSTEM_NAMESPACE_NAME = "system"
const DEPLOYMENT_MODEL_DEFAULT = "default"
const DEPLOYMENT_MODEL_SERVERLESS = "serverless"
const DEPLOYMENT_MODEL_PROVISIONED = "provisioned"

const (
	BACKUP_NOT_POSSIBLE    = -1
	CURRENT_BACKUP_VERSION = 0
	BACKUP_VERSION_1       = 1
	BACKUP_VERSION_2       = 2
)

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

	// Perform authorization and return nil if successful. But does not audit the check.
	AuthorizeInternal(*auth.Privileges, *auth.Credentials) errors.Error

	AdminUser(string) (string, string, error)                 // Admin credentials for a specific node
	PreAuthorize(*auth.Privileges)                            // Transform privileges in the internal format
	CredsString(*auth.Credentials) (string, string)           // Return name, domain from credentials in http request
	GetUserUUID(*auth.Credentials) string                     // Returns user UUID for stats
	GetUserBuckets(*auth.Credentials) []string                // Returns buckets user has access to for serverless accounting
	GetImpersonateBuckets(string, string) []string            // Returns  buckets impersonated user has access to for serverless
	SetLogLevel(level logging.Level)                          // Set log level of in-process indexers
	Inferencer(name InferenceType) (Inferencer, errors.Error) // Schema inference provider by name, e.g. INF_DEFAULT
	Inferencers() ([]Inferencer, errors.Error)                // List of schema inference providers
	StatUpdater() (StatUpdater, errors.Error)                 // Statistics Updater
	UserInfo() (value.Value, errors.Error)                    // The users, and their roles. JSON data.
	GetUserInfoAll() ([]User, errors.Error)                   // Get information about all the users.
	PutUserInfo(u *User) errors.Error                         // Set information for a specific user.
	GetRolesAll() ([]Role, errors.Error)                      // Get all roles that exist in the system.
	DeleteUser(u *User) errors.Error                          // Delete a user
	GetUserInfo(u *User) errors.Error                         // Get a single user's info
	GetGroupInfo(g *Group) errors.Error
	PutGroupInfo(g *Group) errors.Error
	DeleteGroup(g *Group) errors.Error
	GroupInfo() (value.Value, errors.Error)
	GetGroupInfoAll() ([]Group, errors.Error)

	CreateBucket(string, value.Value) errors.Error
	AlterBucket(string, value.Value) errors.Error
	DropBucket(string) errors.Error
	BucketInfo() (value.Value, errors.Error)

	AuditInfo() (*AuditInfo, errors.Error)
	ProcessAuditUpdateStream(callb func(uid string) error) errors.Error
	EnableStorageAudit(val bool)

	SetConnectionSecurityConfig(conSecConfig *ConnectionSecurityConfig) // Update TLS or node-to-node encryption settings.
	CreateSystemCBOStats(requestId string) errors.Error
	DropSystemCBOStats() errors.Error
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

type Datastore2 interface {
	Datastore
	ForeachBucket(func(ExtendedBucket))  // goes through currently loaded buckets
	LoadAllBuckets(func(ExtendedBucket)) // loads all buckets
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
	// All first level namespace objects
	Objects(credentials *auth.Credentials, filter func(string) bool, preload bool) ([]Object, errors.Error)
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
type ExtendedBucket interface {
	Bucket
	GetIOStats(bool, bool, bool, bool, bool) map[string]interface{} // get an object containing IO stats for the bucket
	HasCapability(Migration) bool
	DurabilityPossible() bool
	MarshalJSON() ([]byte, error)
}

type Scope interface {
	Id() string
	Name() string
	AuthKey() string // Key of the object to be used for authorization purposes
	Uid() string

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
	MaxTTL() int64        // The maxTTL setting

	Stats(context QueryContext, which []KeyspaceStats) ([]int64, errors.Error) // Collect multiple stats at once (eg Count, Size)

	Count(context QueryContext) (int64, errors.Error) // count of all documents
	Size(context QueryContext) (int64, errors.Error)  // size of all documents
	Indexer(name IndexType) (Indexer, errors.Error)   // Indexer provider by name, e.g. VIEW or GSI; "" returns default Indexer
	Indexers() ([]Indexer, errors.Error)              // List of index providers

	// Used by both SELECT and DML statements
	Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context QueryContext, subPath []string,
		projection []string, useSubDoc bool) errors.Errors // Bulk key-value fetch from this keyspace

	// Used by DML statements
	// For insert and upsert, nil input keys are replaced with auto-generated keys
	//
	//	@params
	//	preserveMutations : whether the method should preserve & return the mutated keys to the caller
	//
	//	Returns:
	//	1. Number of successfully mutated keys
	//	2. Slice of successfully mutated keys (if preserveMutations = true)
	//	3. List of errors
	//
	Insert(inserts value.Pairs, context QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) // Bulk key-value
	Update(updates value.Pairs, context QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) // Bulk key-value
	Upsert(upserts value.Pairs, context QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) // Bulk key-value
	Delete(deletes value.Pairs, context QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) // Bulk key-value

	SetSubDoc(key string, elems value.Pairs, context QueryContext) (value.Pairs, errors.Error)

	Flush() errors.Error // For flush collection
	IsBucket() bool
	Release(close bool) // Release any resources held by this object

	IsSystemCollection() bool
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
	StartKeyScan(context QueryContext, ranges []*SeqScanRange, offset int64, limit int64, ordered bool, timeout time.Duration,
		pipelineSize int, serverless bool, skipKey func(string) bool) (interface{}, errors.Error)
	StopScan(interface{}) (uint64, errors.Error)
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

// migration infrastructure

type Migration int

const (
	HAS_SYSTEM_COLLECTION = Migration(iota)
)

type Migrator func(string)
type AbortFn func() (string, errors.Error)

var migrationLock sync.RWMutex

var migrations map[Migration]*migrationType

type migrationType struct {
	mtype Migration
	done  bool
	elems map[string]*migrationElem
}

type migrationState int

const (
	_MIGRATION_UNKNOWN = migrationState(iota)
	_MIGRATION_SUCCESS
	_MIGRATION_ABORT
)

type migrationElem struct {
	sync.RWMutex
	name     string
	state    migrationState
	migrator Migrator
	abort    AbortFn
}

func getMigrationElem(what string, t Migration, alloc bool) *migrationElem {
	migrationLock.Lock()
	if migrations == nil {
		migrations = make(map[Migration]*migrationType)
	}
	mt := migrations[t]
	if mt == nil && alloc {
		mt = &migrationType{
			mtype: t,
			elems: make(map[string]*migrationElem, 4),
		}
		migrations[t] = mt
	}
	var me *migrationElem
	if mt != nil {
		me = mt.elems[what]
		if me == nil && alloc {
			me = &migrationElem{
				name:  what,
				state: _MIGRATION_UNKNOWN,
			}
			mt.elems[what] = me
		}
	}
	migrationLock.Unlock()
	return me
}

func RegisterMigrationAbort(abort AbortFn, what string, t Migration) {
	// getMigrationElem() will always return a valid element when alloc is true
	me := getMigrationElem(what, t, true)
	me.Lock()
	me.abort = abort
	me.Unlock()
}

func RegisterMigrator(f Migrator, what string, t Migration) {
	// getMigrationElem() will always return a valid element when alloc is true
	me := getMigrationElem(what, t, true)
	me.Lock()
	me.migrator = f
	me.Unlock()
}

func ExecuteMigrators(b string, t Migration) {
	migrationLock.RLock()
	mt := migrations[t]
	if mt != nil {
		for _, me := range mt.elems {
			if me != nil {
				me.RLock()
				if me.migrator != nil {
					go me.migrator(b)
				}
				me.RUnlock()
			}
		}
	}
	migrationLock.RUnlock()
}

func AbortMigration() (string, errors.Error) {
	var b strings.Builder
	var res string
	var err errors.Error
	found := false
	migrationLock.RLock()
mig:
	for _, mt := range migrations {
		if mt == nil {
			continue
		}
		for _, me := range mt.elems {
			if me != nil {
				me.RLock()
				if me.abort != nil {
					found = true
					res, err = me.abort()
					if err == nil && res != "" {
						b.WriteString(res)
					}
				}
				me.RUnlock()
				if err != nil {
					break mig
				}
			}
		}
	}
	migrationLock.RUnlock()
	if found {
		return b.String(), err
	}
	return "Migration abort: Nothing to abort.", nil
}

func MarkMigrationComplete(success bool, what string, t Migration) {
	me := getMigrationElem(what, t, false)
	if me != nil {
		changed := false
		me.Lock()
		if !(me.state == _MIGRATION_SUCCESS || me.state == _MIGRATION_ABORT) {
			if success {
				me.state = _MIGRATION_SUCCESS
			} else {
				me.state = _MIGRATION_ABORT
			}
			changed = true
		}
		me.Unlock()
		if changed {
			checkMigrationDone(t)
		}
	}
}

func checkMigrationDone(t Migration) {
	migrationLock.Lock()
	mt := migrations[t]
	if mt != nil && !mt.done {
		done := true
		for _, me := range mt.elems {
			if me == nil {
				continue
			}
			me.RLock()
			if !(me.state == _MIGRATION_SUCCESS || me.state == _MIGRATION_ABORT) {
				done = false
			}
			me.RUnlock()
			if !done {
				break
			}
		}
		if done {
			mt.done = true
		}
	}
	migrationLock.Unlock()
}

func IsMigrationComplete(t Migration) bool {
	// if migration is not needed (e.g. alredy done previously), nothing will be allocated
	// for migrations map; in this case assume migration is already complete
	complete := true
	migrationLock.RLock()
	mt := migrations[t]
	if mt != nil {
		complete = mt.done
	}
	migrationLock.RUnlock()
	return complete
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
		return nil, errors.NewDatastoreInvalidPathError("empty path", nil)
	}
	namespace := parts[0]
	if namespace == SYSTEM_NAMESPACE {
		datastore = _SYSTEMSTORE
	} else {
		datastore = _DATASTORE
	}

	if datastore == nil {
		return nil, errors.NewDatastoreNotSetError()
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

func GetScopeUid(parts ...string) (string, errors.Error) {
	scope, err := GetScope(parts...)
	if err != nil {
		return "00000000", err
	}
	return scope.Uid(), nil
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
	Name     string
	Id       string
	Domain   string
	Roles    []Role
	Password string
	Groups   []string
}

type Role struct {
	Name    string
	Target  string
	IsScope bool
}

type Group struct {
	Id    string
	Desc  string
	Roles []Role
}

var NO_STRINGS = make([]string, 0)

// Generates a generic Authorization failure error with code E_ACCESS_DENIED in serverless environments for non-admin users
// Checks if the user has access to the bucket in the path
// Returns E_ACCESS_DENIED if the user does not have access to the bucket
// If the target path is a system keyspace - do not return the generic error. As the system keyspaces are documented.
func CheckBucketAccess(credentials *auth.Credentials, e errors.Error, path []string) errors.Error {

	if tenant.IsServerless() && !IsAdmin(credentials) {

		if len(path) == 0 {
			return nil
		}

		if e != nil && e.Code() == errors.E_DATASTORE_INVALID_BUCKET_PARTS {
			return nil
		}

		namespace := path[0]
		if namespace == SYSTEM_NAMESPACE || namespace == SYSTEM_NAMESPACE_NAME {
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

func getSystemCollection(bucketName string) (Keyspace, errors.Error) {
	store := GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}
	return store.GetSystemCollection(bucketName)
}

func getSystemCollectonIndexConnection(systemCollection Keyspace) (*IndexConnection, Index3, errors.Error) {
	indexerGSI, err := systemCollection.Indexer(GSI)
	if err == nil {
		index3, err := getPrimaryIndexFromIndexer(indexerGSI)
		if err != nil {
			return nil, nil, err
		}
		if index3 != nil {
			return NewIndexConnection(NewSystemContext()), index3, nil
		}
	} else if err.Code() != errors.E_CB_INDEXER_NOT_IMPLEMENTED {
		return nil, nil, err
	}

	// checks the status of or creates (if needed) the primary index
	primaryFunc := func(bucketName string) errors.Error {
		var err errors.Error
		store := GetDatastore()
		if store != nil {
			requestId, _ := util.UUIDV4()
			err = store.CheckSystemCollection(bucketName, requestId)
		}
		return err
	}

	indexerSEQ, err := systemCollection.Indexer(SEQ_SCAN)
	if err == nil {
		index3, err := getPrimaryIndexFromIndexer(indexerSEQ)
		if err != nil {
			return nil, nil, err
		}
		if index3 != nil {
			go primaryFunc(systemCollection.Scope().BucketId())
			return NewIndexConnection(NewSystemContext()), index3, nil
		}
	} else if err.Code() != errors.E_CB_INDEXER_NOT_IMPLEMENTED {
		return nil, nil, err
	}

	// if not successful so far, e.g. if bucket doesn't have SEQ_SCAN support, or if SEQ_SCAN
	// is disabled, try to wait for the creation of primary index (issued above) synchronously
	logging.Debugf("Creating primary index on system collection for bucket %s synchronously", systemCollection.Scope().BucketId())
	err = primaryFunc(systemCollection.Scope().BucketId())
	if err != nil {
		return nil, nil, err
	}
	indexer, err := systemCollection.Indexer(GSI)
	if err != nil {
		return nil, nil, err
	}
	index3, err := getPrimaryIndexFromIndexer(indexer)
	if err != nil {
		return nil, nil, err
	}
	if index3 != nil {
		return NewIndexConnection(NewSystemContext()), index3, nil
	}
	return nil, nil, errors.NewInvalidGSIIndexerError("Primary scan is not available")
}

func getPrimaryIndexFromIndexer(indexer Indexer) (PrimaryIndex3, errors.Error) {
	indexer3, ok := indexer.(Indexer3)
	if !ok {
		return nil, errors.NewInvalidGSIIndexerError("Cannot load from system collection")
	}
	primaries, err := indexer3.PrimaryIndexes()
	if err != nil {
		return nil, err
	}
	for i := range primaries {
		index3, ok := primaries[i].(PrimaryIndex3)
		if ok {
			state, _, err := index3.State()
			if err != nil {
				return nil, err
			} else if state == ONLINE {
				return index3, nil
			}
		}
	}

	return nil, nil
}

func ScanSystemCollection(bucketName string, prefix string, preScan func(Keyspace) errors.Error,
	handler func(string, Keyspace) errors.Error, postScan func(Keyspace) errors.Error) errors.Error {

	systemCollection, err := getSystemCollection(bucketName)
	if err != nil {
		return err
	}
	if systemCollection == nil {
		logging.Debugf("No system collection for '%s'.", bucketName)
		return errors.NewSystemCollectionError(bucketName, nil)
	}

	conn, index3, err := getSystemCollectonIndexConnection(systemCollection)
	if err != nil || conn == nil || index3 == nil {
		return err
	}
	defer func() {
		conn.Dispose()
		conn.SendStop()
	}()

	requestId, err1 := util.UUIDV4()
	if err1 != nil {
		return errors.NewSystemCollectionError("error generating requestId", err1)
	}

	var spans Spans2

	// generate span, which is equivalent to meta().id LIKE _PREFIX+"%"
	spans = make(Spans2, 1)
	ranges := make(Ranges2, 1)

	low := prefix
	high := []byte(low)
	if len(high) > 0 {
		high[len(high)-1]++
	} else {
		high = []byte{0xff}
	}

	ranges[0] = &Range2{
		Low:       value.NewValue(low),
		High:      value.NewValue(string(high)),
		Inclusion: LOW,
	}
	spans[0] = &Span2{
		Ranges: ranges,
	}

	if preScan != nil {
		err := preScan(systemCollection)
		if err != nil {
			return err
		}
	}

	go index3.Scan3(requestId, spans, false, false, nil, 0, 0, nil, nil, UNBOUNDED, nil, conn)

	var item *IndexEntry
	ok := true
	for ok {
		// logic from execution/base.getItemEntry()
		item, ok = conn.Sender().GetEntry()
		if ok {
			if item != nil {
				err = handler(item.PrimaryKey, systemCollection)
				if err != nil {
					return err
				}
			} else {
				ok = false
			}
		}
	}

	errs := conn.GetErrors()
	if len(errs) > 0 {
		return errs[0]
	}

	if postScan != nil {
		err := postScan(systemCollection)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetFlattenKeyAttributes(fks *expression.FlattenKeys, pos int) (attr IkAttributes) {
	attr = IK_NONE
	if fks.HasDesc(pos) {
		attr |= IK_DESC
	}
	if fks.HasMissing(pos) {
		attr |= IK_MISSING
	}
	return
}

func GetIndexKeys(index Index) (indexKeys IndexKeys) {
	if index2, ok := index.(Index2); ok {
		indexKeys = index2.RangeKey2()
	} else {
		for _, e := range index.RangeKey() {
			indexKeys = append(indexKeys, &IndexKey{Expr: e, Attributes: IK_NONE})
		}
	}

	flattenIndexKeys := make(IndexKeys, 0, len(indexKeys))
	for _, ik := range indexKeys {
		if all, ok := ik.Expr.(*expression.All); ok && all.Flatten() {
			fkeys := all.FlattenKeys()
			for pos, fk := range fkeys.Operands() {
				fkey := all.Copy().(*expression.All)
				fkey.SetFlattenValueMapping(fk.Copy())
				attr := GetFlattenKeyAttributes(fkeys, pos)
				flattenIndexKeys = append(flattenIndexKeys, &IndexKey{fkey, attr})
			}
		} else {
			flattenIndexKeys = append(flattenIndexKeys, ik)
		}
	}

	return flattenIndexKeys
}

func GetIndexIncludes(index Index) expression.Expressions {
	if index6, ok := index.(Index6); ok {
		includes := index6.Include()
		if len(includes) > 0 {
			return includes.Copy()
		}
	}
	return nil
}

func CompatibleMetric(distanceType IndexDistanceType, metric expression.VectorMetric) bool {
	switch metric {
	case expression.EUCLIDEAN, expression.EUCLIDEAN_SQUARED, expression.L2, expression.L2_SQUARED:
		return distanceType == IX_DIST_EUCLIDEAN_SQUARED || distanceType == IX_DIST_L2_SQUARED
	case expression.COSINE:
		return distanceType == IX_DIST_COSINE
	case expression.DOT:
		return distanceType == IX_DIST_DOT
	}
	return false
}

func GetVectorDistanceType(metric expression.VectorMetric) IndexDistanceType {
	switch metric {
	case expression.EUCLIDEAN, expression.EUCLIDEAN_SQUARED:
		return IX_DIST_EUCLIDEAN_SQUARED
	case expression.L2, expression.L2_SQUARED:
		return IX_DIST_L2_SQUARED
	case expression.COSINE:
		return IX_DIST_COSINE
	case expression.DOT:
		return IX_DIST_DOT
	}
	return ""
}
