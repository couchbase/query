//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package mock provides a fake, mock 100%-in-memory implementation of
the datastore package, which can be useful for testing.  Because it is
memory-oriented, performance testing of higher layers may be easier
with this mock datastore.
*/
package mock

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/virtual"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

const (
	DEFAULT_NUM_NAMESPACES = 1
	DEFAULT_NUM_KEYSPACES  = 1
	DEFAULT_NUM_ITEMS      = 100000
)

// store is the root for the mock-based Store.
type store struct {
	path           string
	namespaces     map[string]*namespace
	namespaceNames []string
	params         map[string]int
}

func (s *store) Id() string {
	return s.URL()
}

func (s *store) URL() string {
	return "mock:" + s.path
}

func (s *store) Info() datastore.Info {
	return nil
}

func (s *store) NamespaceIds() ([]string, errors.Error) {
	return s.NamespaceNames()
}

func (s *store) NamespaceNames() ([]string, errors.Error) {
	return s.namespaceNames, nil
}

func (s *store) NamespaceById(id string) (p datastore.Namespace, e errors.Error) {
	return s.NamespaceByName(id)
}

func (s *store) NamespaceByName(name string) (p datastore.Namespace, e errors.Error) {
	p, ok := s.namespaces[name]
	if !ok {
		p, e = nil, errors.NewOtherNamespaceNotFoundError(nil, name+" for Mock datastore")
	}

	return
}

func (s *store) Authorize(priv *auth.Privileges, creds *auth.Credentials) errors.Error {
	creds.AuthenticatedUsers = []string{"dummy"}
	return nil
}

func (s *store) AuthorizeInternal(priv *auth.Privileges, creds *auth.Credentials) errors.Error {
	creds.AuthenticatedUsers = []string{"dummy"}
	return nil
}

func (s *store) AdminUser(node string) (string, string, error) {
	return "", "", nil
}

func (s *store) GetUserUUID(*auth.Credentials) string {
	return ""
}

func (s *store) GetUserBuckets(*auth.Credentials) []string {
	return []string{}
}

func (s *store) GetImpersonateBuckets(string, string) []string {
	return []string{}
}

func (s *store) PreAuthorize(*auth.Privileges) {
}

func (s *store) CredsString(*auth.Credentials) (string, string) {
	return "", ""
}

func (s *store) SetLogLevel(level logging.Level) {
	// No-op. Uses query engine logger.
}

func (s *store) Inferencer(name datastore.InferenceType) (datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func (s *store) Inferencers() ([]datastore.Inferencer, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "INFER")
}

func (s *store) StatUpdater() (datastore.StatUpdater, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "UPDATE STATISTICS")
}

func (s *store) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing
}

func (s *store) AuditInfo() (*datastore.AuditInfo, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "AuditInfo")
}

func (s *store) ProcessAuditUpdateStream(callb func(uid string) error) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "ProcessAuditUpdateStream")
}

func (s *store) EnableStorageAudit(val bool) {
}

func (s *store) UserInfo() (value.Value, errors.Error) {
	// Stub implementation with fixed content.
	content := `[{"name":"Ivan Ivanov","id":"ivanivanov","domain":"local","roles":[{"role":"cluster_admin"},
                        {"role":"bucket_admin","bucket_name":"default"}]},
                        {"name":"Petr Petrov","id":"petrpetrov","domain":"local","roles":[{"role":"replication_admin"}]}]`
	jsonData := make([]interface{}, 3)
	err := json.Unmarshal([]byte(content), &jsonData)
	if err != nil {
		return nil, errors.NewServiceErrorInvalidJSON(err)
	}
	v := value.NewValue(jsonData)
	return v, nil
}

func (s *store) GetUserInfoAll() ([]datastore.User, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GetUserInfoAll")
}

func (s *store) PutUserInfo(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "PutUserInfo")
}

func (s *store) DeleteUser(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DeleteUser")
}

func (s *store) GetUserInfo(u *datastore.User) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "GetUserInfo")
}

func (s *store) GetRolesAll() ([]datastore.Role, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GetRolesAll")
}

func (s *store) GetGroupInfo(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "GetGroupInfo")
}

func (s *store) PutGroupInfo(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "PutGroupInfo")
}

func (s *store) DeleteGroup(*datastore.Group) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DeleteGroup")
}

func (s *store) GroupInfo() (value.Value, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GroupInfo")
}

func (s *store) GetGroupInfoAll() ([]datastore.Group, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "GetGroupInfoAll")
}

func (s *store) CreateBucket(string, value.Value) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "CreateBucket")
}

func (s *store) AlterBucket(string, value.Value) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "AlterBucket")
}

func (s *store) DropBucket(string) errors.Error {
	return errors.NewOtherNotImplementedError(nil, "DropBucket")
}

func (s *store) BucketInfo() (value.Value, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "BucketInfo")
}

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	return nil
}

func (s *store) DropSystemCBOStats() errors.Error {
	return nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	return false, nil
}

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) CheckSystemCollection(bucketName, requestId string) errors.Error {
	return nil
}

func (s *store) StartTransaction(stmtAtomicity bool, context datastore.QueryContext) (map[string]bool, errors.Error) {
	return nil, errors.NewTranDatastoreNotSupportedError("mock")
}

func (s *store) CommitTransaction(stmtAtomicity bool, context datastore.QueryContext) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("mock")
}

func (s *store) RollbackTransaction(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("mock")
}

func (s *store) SetSavepoint(stmtAtomicity bool, context datastore.QueryContext, sname string) errors.Error {
	return errors.NewTranDatastoreNotSupportedError("mock")
}

func (s *store) TransactionDeltaKeyScan(keyspace string, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()
}

// namespace represents a mock-based Namespace.
type namespace struct {
	store         *store
	name          string
	keyspaces     map[string]*keyspace
	keyspaceNames []string
}

func (p *namespace) Datastore() datastore.Datastore {
	return p.store
}

func (p *namespace) Id() string {
	return p.Name()
}

func (p *namespace) Name() string {
	return p.name
}

func (p *namespace) KeyspaceIds() ([]string, errors.Error) {
	return p.KeyspaceNames()
}

func (p *namespace) KeyspaceNames() ([]string, errors.Error) {
	return p.keyspaceNames, nil
}

func (p *namespace) Objects(credentials *auth.Credentials, filter func(string) bool, preload bool) (
	[]datastore.Object, errors.Error) {

	rv := make([]datastore.Object, len(p.keyspaceNames))
	i := 0
	for _, k := range p.keyspaceNames {
		rv[i] = datastore.Object{Id: k, Name: k, IsKeyspace: true}
		i++
	}
	return rv, nil
}

func (p *namespace) KeyspaceById(id string) (b datastore.Keyspace, e errors.Error) {
	return p.KeyspaceByName(id)
}

func (p *namespace) KeyspaceByName(name string) (b datastore.Keyspace, e errors.Error) {
	b, ok := p.keyspaces[name]
	if !ok {
		b, e = nil, errors.NewOtherKeyspaceNotFoundError(nil, name+" for Mock datastore")
	}

	return
}

func (p *namespace) VirtualKeyspaceByName(path []string) (datastore.Keyspace, errors.Error) {
	return virtual.NewVirtualKeyspace(p, path)
}

func (p *namespace) MetadataVersion() uint64 {
	return 0
}

func (p *namespace) MetadataId() string {
	return p.name
}

func (p *namespace) BucketIds() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketNames() ([]string, errors.Error) {
	return datastore.NO_STRINGS, nil
}

func (p *namespace) BucketById(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewOtherNoBuckets("mock")
}

func (p *namespace) BucketByName(name string) (datastore.Bucket, errors.Error) {
	return nil, errors.NewOtherNoBuckets("mock")
}

// keyspace is a mock-based keyspace.
type keyspace struct {
	namespace *namespace
	name      string
	nitems    int
	mi        datastore.Indexer
}

func (b *keyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspace) Namespace() datastore.Namespace {
	return b.namespace
}

func (b *keyspace) ScopeId() string {
	return ""
}

func (b *keyspace) Scope() datastore.Scope {
	return nil
}

func (b *keyspace) MaxTTL() int64 {
	return 0
}

func (b *keyspace) Id() string {
	return b.Name()
}

func (b *keyspace) Name() string {
	return b.name
}

func (b *keyspace) Uid() string {
	return b.name
}

func (b *keyspace) QualifiedName() string {
	return b.namespace.name + ":" + b.name
}

func (b *keyspace) AuthKey() string {
	return b.name
}

func (b *keyspace) MetadataVersion() uint64 {
	return 0
}

func (this *keyspace) Stats(context datastore.QueryContext, which []datastore.KeyspaceStats) ([]int64, errors.Error) {
	var err errors.Error

	res := make([]int64, len(which))
	for i, f := range which {
		var val int64

		switch f {
		case datastore.KEYSPACE_COUNT:
			val, err = this.Count(context)
		case datastore.KEYSPACE_SIZE:
			val, err = this.Size(context)
		case datastore.KEYSPACE_MEM_SIZE:
			val = -1
		}
		if err != nil {
			return nil, err
		}
		res[i] = val
	}
	return res, err
}

func (b *keyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	return int64(b.nitems), nil
}

func (b *keyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return int64(b.nitems) * 25, nil // assumes each document is 25 bytes, see genItem()
}

func (b *keyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.mi, nil
}

func (b *keyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.mi}, nil
}

func (b *keyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	for _, k := range keys {
		item, e := b.fetchOne(k)
		if e != nil {
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetId(k)
		}

		keysMap[k] = item
	}
	return errs
}

func (b *keyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	i, e := strconv.Atoi(key)
	if e != nil {
		return nil, errors.NewOtherKeyNotFoundError(e, fmt.Sprintf("no mock item: %v", key))
	} else {
		return genItem(i, b.nitems)
	}
}

// generate a mock document - used by fetchOne to mock a document in the keyspace
func genItem(i int, nitems int) (value.AnnotatedValue, errors.Error) {
	if i < 0 || i >= nitems {
		return nil, errors.NewOtherDatastoreError(nil,
			fmt.Sprintf("item out of mock range: %v [0,%v)", i, nitems))
	}
	id := strconv.Itoa(i)
	doc := value.NewAnnotatedValue(map[string]interface{}{"id": id, "i": float64(i)})
	doc.SetId(id)
	return doc, nil
}

func (b *keyspace) Insert(inserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	// FIXME
	return 0, nil, errors.Errors{errors.NewOtherNotImplementedError(nil, "for Mock datastore")}
}

func (b *keyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	// FIXME
	return 0, nil, errors.Errors{errors.NewOtherNotImplementedError(nil, "for Mock datastore")}
}

func (b *keyspace) Upsert(upserts value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	// FIXME
	return 0, nil, errors.Errors{errors.NewOtherNotImplementedError(nil, "for Mock datastore")}
}

func (b *keyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	// FIXME
	return 0, nil, errors.Errors{errors.NewOtherNotImplementedError(nil, "for Mock datastore")}
}

func (b *keyspace) SetSubDoc(string, value.Pairs, datastore.QueryContext) (value.Pairs, errors.Error) {
	return nil, errors.NewOtherNotImplementedError(nil, "for mock datastore.")
}

func (b *keyspace) Release(close bool) {
}

func (b *keyspace) CreateScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(b.name)
}

func (b *keyspace) DropScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(b.name)
}

func (b *keyspace) Flush() errors.Error {
	return errors.NewNoFlushError(b.name)
}

func (b *keyspace) IsBucket() bool {
	return true
}

type mockIndexer struct {
	keyspace *keyspace
	indexes  map[string]datastore.Index
	primary  datastore.PrimaryIndex
}

func newMockIndexer(keyspace *keyspace) datastore.Indexer {

	return &mockIndexer{
		keyspace: keyspace,
		indexes:  make(map[string]datastore.Index),
	}
}

func (mi *mockIndexer) BucketId() string {
	return ""
}

func (mi *mockIndexer) ScopeId() string {
	return ""
}

func (mi *mockIndexer) KeyspaceId() string {
	return mi.keyspace.Id()
}

func (mi *mockIndexer) Name() datastore.IndexType {
	return datastore.DEFAULT
}

func (mi *mockIndexer) IndexIds() ([]string, errors.Error) {
	rv := make([]string, 0, len(mi.indexes))
	for name, _ := range mi.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (mi *mockIndexer) IndexNames() ([]string, errors.Error) {
	rv := make([]string, 0, len(mi.indexes))
	for name, _ := range mi.indexes {
		rv = append(rv, name)
	}
	return rv, nil
}

func (mi *mockIndexer) IndexById(id string) (datastore.Index, errors.Error) {
	return mi.IndexByName(id)
}

func (mi *mockIndexer) IndexByName(name string) (datastore.Index, errors.Error) {
	index, ok := mi.indexes[name]
	if !ok {
		return nil, errors.NewOtherIdxNotFoundError(nil, name+"for Mock datastore")
	}
	return index, nil
}

func (mi *mockIndexer) PrimaryIndexes() ([]datastore.PrimaryIndex, errors.Error) {
	return []datastore.PrimaryIndex{mi.primary}, nil
}

func (mi *mockIndexer) Indexes() ([]datastore.Index, errors.Error) {
	return []datastore.Index{mi.primary}, nil
}

func (mi *mockIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (datastore.PrimaryIndex, errors.Error) {
	if mi.primary == nil {
		pi := new(primaryIndex)
		mi.primary = pi
		pi.keyspace = mi.keyspace
		pi.name = name
		pi.indexer = mi
		mi.indexes[pi.name] = pi
	}

	return mi.primary, nil
}

func (mi *mockIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewOtherNotSupportedError(nil, "CREATE INDEX is not supported for mock datastore.")
}

func (mi *mockIndexer) BuildIndexes(requestId string, names ...string) errors.Error {
	return errors.NewOtherNotSupportedError(nil, "BUILD INDEXES is not supported for mock datastore.")
}

func (mi *mockIndexer) Refresh() errors.Error {
	return nil
}

func (mi *mockIndexer) MetadataVersion() uint64 {
	return 0
}

func (mi *mockIndexer) SetLogLevel(level logging.Level) {
	// No-op, uses query engine logger
}

func (mi *mockIndexer) SetConnectionSecurityConfig(conSecConfig *datastore.ConnectionSecurityConfig) {
	// Do nothing.
}

// NewDatastore creates a new mock store for the given "path".  The
// path has prefix "mock:", with the rest of the path treated as a
// comma-separated key=value params.  For example:
// mock:namespaces=2,keyspaces=5,items=50000 The above means 2
// namespaces.  And, each namespace has 5 keyspaces.  And, each
// keyspace with 50000 items.  By default, you get...
// mock:namespaces=1,keyspaces=1,items=100000 Which is what you'd get
// by specifying a path of just...  mock:
func NewDatastore(path string) (datastore.Datastore, errors.Error) {
	if strings.HasPrefix(path, "mock:") {
		path = path[5:]
	}
	params := map[string]int{}
	for _, kv := range strings.Split(path, ",") {
		if kv == "" {
			continue
		}
		pair := strings.Split(kv, "=")
		v, e := strconv.Atoi(pair[1])
		if e != nil {
			return nil, errors.NewOtherDatastoreError(e,
				fmt.Sprintf("could not parse mock param key: %s, val: %s",
					pair[0], pair[1]))
		}
		params[pair[0]] = v
	}
	nnamespaces := paramVal(params, "namespaces", DEFAULT_NUM_NAMESPACES)
	nkeyspaces := paramVal(params, "keyspaces", DEFAULT_NUM_KEYSPACES)
	nitems := paramVal(params, "items", DEFAULT_NUM_ITEMS)
	s := &store{path: path, params: params, namespaces: map[string]*namespace{}, namespaceNames: []string{}}
	for i := 0; i < nnamespaces; i++ {
		p := &namespace{store: s, name: "p" + strconv.Itoa(i), keyspaces: map[string]*keyspace{}, keyspaceNames: []string{}}
		for j := 0; j < nkeyspaces; j++ {
			b := &keyspace{namespace: p, name: "b" + strconv.Itoa(j), nitems: nitems}

			b.mi = newMockIndexer(b)
			b.mi.CreatePrimaryIndex("", "#primary", nil)
			p.keyspaces[b.name] = b
			p.keyspaceNames = append(p.keyspaceNames, b.name)
		}
		s.namespaces[p.name] = p
		s.namespaceNames = append(s.namespaceNames, p.name)
	}
	return s, nil
}

func paramVal(params map[string]int, key string, defaultVal int) int {
	v, ok := params[key]
	if ok {
		return v
	}
	return defaultVal
}

// primaryIndex performs full keyspace scans.
type primaryIndex struct {
	name     string
	keyspace *keyspace
	indexer  *mockIndexer
}

func (pi *primaryIndex) BucketId() string {
	return ""
}

func (pi *primaryIndex) ScopeId() string {
	return ""
}

func (pi *primaryIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *primaryIndex) Id() string {
	return pi.Name()
}

func (pi *primaryIndex) Name() string {
	return pi.name
}

func (pi *primaryIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *primaryIndex) Indexer() datastore.Indexer {
	return pi.indexer
}

func (pi *primaryIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *primaryIndex) Condition() expression.Expression {
	return nil
}

func (pi *primaryIndex) IsPrimary() bool {
	return true
}

func (pi *primaryIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *primaryIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *primaryIndex) Drop(requestId string) errors.Error {
	return errors.NewOtherIdxNoDrop(nil, "This primary index cannot be dropped for Mock datastore.")
}

func (pi *primaryIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	// For primary indexes, bounds must always be strings, so we
	// can just enforce that directly
	low, high := "", ""

	// Ensure that lower bound is a string, if any
	if len(span.Range.Low) > 0 {
		a := span.Range.Low[0].Actual()
		switch a := a.(type) {
		case string:
			low = a
		default:
			conn.Error(errors.NewOtherDatastoreError(nil, fmt.Sprintf("Invalid lower bound %v of type %T.", a, a)))
			return
		}
	}

	// Ensure that upper bound is a string, if any
	if len(span.Range.High) > 0 {
		a := span.Range.High[0].Actual()
		switch a := a.(type) {
		case string:
			high = a
		default:
			conn.Error(errors.NewOtherDatastoreError(nil, fmt.Sprintf("Invalid upper bound %v of type %T.", a, a)))
			return
		}
	}

	if limit == 0 {
		limit = int64(pi.keyspace.nitems)
	}

	for i := 0; i < pi.keyspace.nitems && int64(i) < limit; i++ {
		id := strconv.Itoa(i)

		if low != "" &&
			(id < low ||
				(id == low && (span.Range.Inclusion&datastore.LOW == 0))) {
			continue
		}

		low = ""

		if high != "" &&
			(id > high ||
				(id == high && (span.Range.Inclusion&datastore.HIGH == 0))) {
			break
		}

		entry := datastore.IndexEntry{PrimaryKey: id}
		conn.Sender().SendEntry(&entry)
	}
}

func (pi *primaryIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	if limit == 0 {
		limit = int64(pi.keyspace.nitems)
	}

	for i := 0; i < pi.keyspace.nitems && int64(i) < limit; i++ {
		entry := datastore.IndexEntry{PrimaryKey: strconv.Itoa(i)}
		conn.Sender().SendEntry(&entry)
	}
}
