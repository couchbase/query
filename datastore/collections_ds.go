package datastore

import (
	"github.com/couchbase/go_json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type CollectionsNamespace struct {
	id        string
	keyspaces map[string]*CollectionsKeyspace // keyspaces by id
	buckets   map[string]*CollectionsBucket   // buckets by id
}

func NewCollectionsNamespace(id string) *CollectionsNamespace {
	cns := &CollectionsNamespace{
		id:        id,
		keyspaces: make(map[string]*CollectionsKeyspace),
		buckets:   make(map[string]*CollectionsBucket),
	}

	bucket := NewCollectionsBucket("myBucket")
	cns.AddBucket(bucket)

	scope := NewCollectionsScope("myScope")
	bucket.AddScope(scope)

	ks := NewCollectionsKeyspace("myCollection")
	scope.AddKeyspace(ks)

	ks.AddDocument("v1", `{ "f1" : "string value A", "f2" : 10 }`)
	ks.AddDocument("v2", `{ "f1" : "string value B", "f2" : 11 }`)
	ks.AddDocument("v3", `{ "f1" : "string value C", "f2" : 12 }`)

	indexer, _ := ks.Indexer(GSI)
	_, err := indexer.CreatePrimaryIndex("", "#primary", nil)
	if err != nil {
		logging.Errorf(" Error creating Primary Index - %v ", err)
	}
	return cns
}

func (ns *CollectionsNamespace) AddBucket(bucket *CollectionsBucket) {
	ns.buckets[bucket.Id()] = bucket
	bucket.namespace = ns
}

func (ns *CollectionsNamespace) AddKeyspace(keyspace *CollectionsKeyspace) {
	ns.keyspaces[keyspace.Id()] = keyspace
	keyspace.namespace = ns
	keyspace.scope = nil
}

func (ns *CollectionsNamespace) DatastoreId() string {
	return "http://127.0.0.1:8094"
}

func (ns *CollectionsNamespace) Id() string {
	return ns.id
}

func (ns *CollectionsNamespace) Name() string {
	return ns.id
}

func (ns *CollectionsNamespace) KeyspaceIds() ([]string, errors.Error) {
	ret := make([]string, len(ns.keyspaces))
	i := 0
	for _, v := range ns.keyspaces {
		ret[i] = v.Id()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) KeyspaceNames() ([]string, errors.Error) {
	ret := make([]string, len(ns.keyspaces))
	i := 0
	for _, v := range ns.keyspaces {
		ret[i] = v.Name()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) Objects(preload bool) ([]Object, errors.Error) {
	ret := make([]Object, len(ns.keyspaces)+len(ns.buckets))
	i := 0
	for _, v := range ns.keyspaces {
		ret[i] = Object{Id: v.Name(), Name: v.Name(), IsKeyspace: true}
		i++
	}
	for _, v := range ns.buckets {
		ret[i] = Object{Id: v.Name(), Name: v.Name(), IsBucket: true}
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) KeyspaceById(id string) (Keyspace, errors.Error) {
	ks := ns.keyspaces[id]
	if ks == nil {
		return nil, errors.NewCbKeyspaceNotFoundError(nil, id)
	}
	return ks, nil
}

func (ns *CollectionsNamespace) KeyspaceByName(name string) (Keyspace, errors.Error) {
	for _, v := range ns.keyspaces {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbKeyspaceNotFoundError(nil, name)
}

func (ns *CollectionsNamespace) MetadataVersion() uint64 {
	return 1
}

func (ns *CollectionsNamespace) BucketIds() ([]string, errors.Error) {
	ret := make([]string, len(ns.buckets))
	i := 0
	for _, v := range ns.buckets {
		ret[i] = v.Id()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) BucketNames() ([]string, errors.Error) {
	ret := make([]string, len(ns.buckets))
	i := 0
	for _, v := range ns.buckets {
		ret[i] = v.Name()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) BucketById(id string) (Bucket, errors.Error) {
	bucket := ns.buckets[id]
	if bucket == nil {
		return nil, errors.NewCbBucketNotFoundError(nil, id)
	}
	return bucket, nil
}

func (ns *CollectionsNamespace) BucketByName(name string) (Bucket, errors.Error) {
	for _, v := range ns.buckets {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbBucketNotFoundError(nil, name)
}

type CollectionsBucket struct {
	id        string
	namespace *CollectionsNamespace

	scopes map[string]*CollectionsScope // scopes by id
}

func NewCollectionsBucket(id string) *CollectionsBucket {
	return &CollectionsBucket{
		id:     id,
		scopes: make(map[string]*CollectionsScope),
	}
}

func (cb *CollectionsBucket) AddScope(scope *CollectionsScope) {
	cb.scopes[scope.Id()] = scope
	scope.bucket = cb
}

func (cb *CollectionsBucket) Id() string {
	return cb.id
}

func (cb *CollectionsBucket) Name() string {
	return cb.id
}

func (cb *CollectionsBucket) Uid() string {
	return cb.id
}

func (cb *CollectionsBucket) AuthKey() string {
	return cb.id
}

func (cb *CollectionsBucket) NamespaceId() string {
	return cb.namespace.Id()
}

func (cb *CollectionsBucket) Namespace() Namespace {
	return cb.namespace
}

func (cb *CollectionsBucket) DefaultKeyspace() (Keyspace, errors.Error) {
	return nil, nil
}

func (cb *CollectionsBucket) ScopeIds() ([]string, errors.Error) {
	ids := make([]string, len(cb.scopes))
	ix := 0
	for k := range cb.scopes {
		ids[ix] = k
		ix++
	}
	return ids, nil
}

func (cb *CollectionsBucket) ScopeNames() ([]string, errors.Error) {
	ids := make([]string, len(cb.scopes))
	ix := 0
	for _, v := range cb.scopes {
		ids[ix] = v.Name()
		ix++
	}
	return ids, nil
}

func (cb *CollectionsBucket) ScopeById(id string) (Scope, errors.Error) {
	scope := cb.scopes[id]
	if scope == nil {
		return nil, errors.NewCbScopeNotFoundError(nil, id)
	}
	return scope, nil
}

func (cb *CollectionsBucket) ScopeByName(name string) (Scope, errors.Error) {
	for _, v := range cb.scopes {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbScopeNotFoundError(nil, name)
}

func (cb *CollectionsBucket) CreateScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(cb.Name())
}

func (cb *CollectionsBucket) DropScope(name string) errors.Error {
	return errors.NewScopesNotSupportedError(cb.Name())
}

type CollectionsScope struct {
	id     string
	bucket *CollectionsBucket

	keyspaces map[string]*CollectionsKeyspace // keyspaces by id
}

func NewCollectionsScope(id string) *CollectionsScope {
	return &CollectionsScope{
		id:        id,
		keyspaces: make(map[string]*CollectionsKeyspace),
	}
}

func (cs *CollectionsScope) AddKeyspace(ks *CollectionsKeyspace) {
	cs.keyspaces[ks.Id()] = ks
	ks.namespace = nil
	ks.scope = cs
}

func (cs *CollectionsScope) Id() string {
	return cs.id
}

func (cs *CollectionsScope) Name() string {
	return cs.id
}

func (cs *CollectionsScope) AuthKey() string {
	return cs.id
}

func (cs *CollectionsScope) BucketId() string {
	return cs.bucket.Id()
}

func (cs *CollectionsScope) Bucket() Bucket {
	return cs.bucket
}

func (cs *CollectionsScope) KeyspaceIds() ([]string, errors.Error) {
	ids := make([]string, len(cs.keyspaces))
	ix := 0
	for k := range cs.keyspaces {
		ids[ix] = k
		ix++
	}
	return ids, nil
}

func (cs *CollectionsScope) KeyspaceNames() ([]string, errors.Error) {
	ids := make([]string, len(cs.keyspaces))
	ix := 0
	for _, v := range cs.keyspaces {
		ids[ix] = v.Name()
		ix++
	}
	return ids, nil
}

func (cs *CollectionsScope) KeyspaceById(id string) (Keyspace, errors.Error) {
	ks := cs.keyspaces[id]
	if ks == nil {
		return nil, errors.NewCbKeyspaceNotFoundError(nil, id)
	}
	return ks, nil
}

func (cs *CollectionsScope) KeyspaceByName(name string) (Keyspace, errors.Error) {
	for _, v := range cs.keyspaces {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbKeyspaceNotFoundError(nil, name)
}

func (cs *CollectionsScope) CreateCollection(name string) errors.Error {
	return errors.NewScopesNotSupportedError(cs.Name())
}

func (cs *CollectionsScope) DropCollection(name string) errors.Error {
	return errors.NewScopesNotSupportedError(cs.Name())
}

type CollectionsKeyspace struct {
	id        string
	namespace *CollectionsNamespace
	scope     *CollectionsScope
	docs      map[string]value.AnnotatedValue
	indexer   *CollectionsIndexer // keyspace owns indexer; GSI only
}

func NewCollectionsKeyspace(name string) *CollectionsKeyspace {
	ks := &CollectionsKeyspace{
		id:   name,
		docs: make(map[string]value.AnnotatedValue),
	}
	indexer := &CollectionsIndexer{
		keyspace:       ks,
		primaryIndexes: make([]*CollectionsPrimaryIndex, 0),
	}
	ks.indexer = indexer
	return ks
}

func (ks *CollectionsKeyspace) AddDocument(id string, jsonDoc string) {
	var jsonVal map[string]interface{}
	err := json.Unmarshal([]byte(jsonDoc), &jsonVal)
	if err != nil {
		logging.Errorf("Unable to parse document value %s: %v", id, err)
	}
	doc := value.NewAnnotatedValue(jsonVal)
	doc.SetId(id)
	ks.docs[id] = doc
}

func (ks *CollectionsKeyspace) Id() string {
	return ks.id
}

func (ks *CollectionsKeyspace) Name() string {
	return ks.id
}

func (ks *CollectionsKeyspace) Uid() string {
	return ks.id
}

// not really used in tests, so can be left as unqualified
func (ks *CollectionsKeyspace) QualifiedName() string {
	return ks.id
}

func (ks *CollectionsKeyspace) AuthKey() string {
	return ks.id
}

func (ks *CollectionsKeyspace) NamespaceId() string {
	if ks.namespace == nil {
		return ""
	}
	return ks.namespace.Id()
}

func (ks *CollectionsKeyspace) Namespace() Namespace {
	return ks.namespace
}

func (ks *CollectionsKeyspace) ScopeId() string {
	if ks.scope == nil {
		return ""
	}
	return ks.scope.Id()
}

func (ks *CollectionsKeyspace) Scope() Scope {
	return ks.scope
}

func (ks *CollectionsKeyspace) Stats(context QueryContext, which []KeyspaceStats) ([]int64, errors.Error) {
	var err errors.Error

	res := make([]int64, len(which))
	for i, f := range which {
		var val int64

		switch f {
		case KEYSPACE_COUNT:
			val, err = ks.Count(context)
		case KEYSPACE_SIZE:
			val, err = ks.Size(context)
		}
		if err != nil {
			return nil, err
		}
		res[i] = val
	}
	return res, err
}

func (ks *CollectionsKeyspace) Count(context QueryContext) (int64, errors.Error) {
	return int64(len(ks.docs)), nil
}

func (ks *CollectionsKeyspace) Size(context QueryContext) (int64, errors.Error) {
	return int64(len(ks.docs)) * 25, nil // assume 25-bytes per document (similar to mock)
}

func (ks *CollectionsKeyspace) Indexer(name IndexType) (Indexer, errors.Error) {
	if name == GSI {
		return ks.indexer, nil
	}
	return nil, nil
}

func (ks *CollectionsKeyspace) Indexers() ([]Indexer, errors.Error) {
	if ks.indexer == nil {
		return make([]Indexer, 0), nil
	}
	return []Indexer{ks.indexer}, nil
}

func (ks *CollectionsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue, context QueryContext, subPath []string) []errors.Error {
	for _, v := range keys {
		doc := ks.docs[v]
		if doc != nil {
			keysMap[v] = doc
		}
	}

	return nil
}

// Used by DML statements
// For insert and upsert, nil input keys are replaced with auto-generated keys
func (ks *CollectionsKeyspace) Insert(inserts []value.Pair, context QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Insert()")
}

func (ks *CollectionsKeyspace) Update(updates []value.Pair, context QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Update()")
}

func (ks *CollectionsKeyspace) Upsert(upserts []value.Pair, context QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Upsert()")
}

func (ks *CollectionsKeyspace) Delete(deletes []value.Pair, context QueryContext) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Delete()")
}

func (ks *CollectionsKeyspace) Release(close bool) {
	// do nothing
}

func (ks *CollectionsKeyspace) Flush() errors.Error {
	return errors.NewNoFlushError(ks.Name())
}

func (ks *CollectionsKeyspace) IsBucket() bool {
	return false
}

type CollectionsIndexer struct {
	keyspace       *CollectionsKeyspace       // keyspace owns indexer
	primaryIndexes []*CollectionsPrimaryIndex // indexer owns indexes
}

func (indexer *CollectionsIndexer) BucketId() string {
	return ""
}

func (indexer *CollectionsIndexer) ScopeId() string {
	return ""
}

func (indexer *CollectionsIndexer) KeyspaceId() string {
	return indexer.keyspace.Id()
}

func (indexer *CollectionsIndexer) Name() IndexType {
	return GSI
}

func (indexer *CollectionsIndexer) IndexIds() ([]string, errors.Error) {
	ret := make([]string, len(indexer.primaryIndexes))
	for i, v := range indexer.primaryIndexes {
		ret[i] = v.Id()
	}
	return ret, nil
}

func (indexer *CollectionsIndexer) IndexNames() ([]string, errors.Error) {
	ret := make([]string, len(indexer.primaryIndexes))
	for i, v := range indexer.primaryIndexes {
		ret[i] = v.Name()
	}
	return ret, nil
}

func (indexer *CollectionsIndexer) IndexById(id string) (Index, errors.Error) {
	for _, v := range indexer.primaryIndexes {
		if v.Id() == id {
			return v, nil
		}
	}
	return nil, nil
}

func (indexer *CollectionsIndexer) IndexByName(name string) (Index, errors.Error) {
	for _, v := range indexer.primaryIndexes {
		if v.Name() == name {
			return v, nil
		}
	}
	return nil, nil
}

func (indexer *CollectionsIndexer) PrimaryIndexes() ([]PrimaryIndex, errors.Error) {
	ret := make([]PrimaryIndex, len(indexer.primaryIndexes))
	for i, v := range indexer.primaryIndexes {
		ret[i] = v
	}
	return ret, nil
}

func (indexer *CollectionsIndexer) Indexes() ([]Index, errors.Error) {
	ret := make([]Index, len(indexer.primaryIndexes))
	for i, v := range indexer.primaryIndexes {
		ret[i] = v
	}
	return ret, nil
}

func (indexer *CollectionsIndexer) CreatePrimaryIndex(requestId, name string, with value.Value) (PrimaryIndex, errors.Error) {
	index := &CollectionsPrimaryIndex{
		keyspace: indexer.keyspace,
		id:       name,
		indexer:  indexer,
	}
	indexer.primaryIndexes = append(indexer.primaryIndexes, index)
	return index, nil
}

func (indexer *CollectionsIndexer) CreateIndex(requestId, name string, seekKey, rangeKey expression.Expressions,
	where expression.Expression, with value.Value) (Index, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsIndexer.CreateIndex()")
}

func (indexer *CollectionsIndexer) BuildIndexes(requestId string, name ...string) errors.Error {
	return errors.NewNotImplemented("CollectionsIndexer.BuildIndexes()")
}

func (indexer *CollectionsIndexer) Refresh() errors.Error {
	return nil
}

func (indexer *CollectionsIndexer) MetadataVersion() uint64 {
	return 1
}

func (indexer *CollectionsIndexer) SetLogLevel(level logging.Level) {
	// Do nothing.
}

func (indexer *CollectionsIndexer) SetConnectionSecurityConfig(conSecConfig *ConnectionSecurityConfig) {
	// Do nothing for now.
}

type CollectionsPrimaryIndex struct {
	keyspace *CollectionsKeyspace
	id       string
	indexer  *CollectionsIndexer
}

func (index *CollectionsPrimaryIndex) BucketId() string {
	return ""
}

func (index *CollectionsPrimaryIndex) ScopeId() string {
	return ""
}

func (index *CollectionsPrimaryIndex) KeyspaceId() string {
	return index.keyspace.Id()
}

func (index *CollectionsPrimaryIndex) Id() string {
	return index.id
}

func (index *CollectionsPrimaryIndex) Name() string {
	return index.id
}

func (index *CollectionsPrimaryIndex) Type() IndexType {
	return GSI
}

func (index *CollectionsPrimaryIndex) Indexer() Indexer {
	return index.indexer
}

func (index *CollectionsPrimaryIndex) SeekKey() expression.Expressions {
	return nil
}

func (index *CollectionsPrimaryIndex) RangeKey() expression.Expressions {
	return nil
}

func (index *CollectionsPrimaryIndex) Condition() expression.Expression {
	return nil
}

func (index *CollectionsPrimaryIndex) IsPrimary() bool {
	return true
}

func (index *CollectionsPrimaryIndex) State() (state IndexState, msg string, err errors.Error) {
	return ONLINE, "", nil
}

func (index *CollectionsPrimaryIndex) Statistics(requestId string, span *Span) (Statistics, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsPrimaryIndex.Statistics()")
}

func (index *CollectionsPrimaryIndex) Drop(requestId string) errors.Error {
	return errors.NewNotImplemented("CollectionsPrimaryIndex.Statistics()")
}

func (index *CollectionsPrimaryIndex) Scan(requestId string, span *Span, distinct bool, limit int64, cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection) {
	defer conn.Sender().Close()

	for k := range index.keyspace.docs {
		entry := IndexEntry{PrimaryKey: k}
		conn.Sender().SendEntry(&entry)
	}
}

func (index *CollectionsPrimaryIndex) ScanEntries(requestId string, limit int64, cons ScanConsistency, vector timestamp.Vector, conn *IndexConnection) {
	defer conn.Sender().Close()
	for key := range index.keyspace.docs {
		entry := IndexEntry{PrimaryKey: key}
		conn.Sender().SendEntry(&entry)
	}
}

func (index *CollectionsPrimaryIndex) Count(span *Span, cons ScanConsistency, vector timestamp.Vector) (int64, errors.Error) {
	return index.keyspace.Count(nil)
}
