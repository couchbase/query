package datastore

import (
	"github.com/couchbase/go_json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
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

func (cb *CollectionsBucket) NamespaceId() string {
	return cb.namespace.Id()
}

func (cb *CollectionsBucket) Namespace() Namespace {
	return cb.namespace
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

type CollectionsKeyspace struct {
	id        string
	namespace *CollectionsNamespace
	scope     *CollectionsScope
	docs      map[string]value.AnnotatedValue
}

func NewCollectionsKeyspace(name string) *CollectionsKeyspace {
	ks := &CollectionsKeyspace{
		id:   name,
		docs: make(map[string]value.AnnotatedValue),
	}
	return ks
}

func (ks *CollectionsKeyspace) AddDocument(id string, jsonDoc string) {
	var jsonVal map[string]interface{}
	err := json.Unmarshal([]byte(jsonDoc), &jsonVal)
	if err != nil {
		logging.Errorf("Unable to parse document value %s: %v", id, err)
	}
	doc := value.NewAnnotatedValue(jsonVal)
	doc.SetAttachment("meta", map[string]interface{}{"id": id})
	doc.SetId(id)
	ks.docs[id] = doc
}

func (ks *CollectionsKeyspace) Id() string {
	return ks.id
}

func (ks *CollectionsKeyspace) Name() string {
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

func (ks *CollectionsKeyspace) Count(context QueryContext) (int64, errors.Error) {
	return int64(len(ks.docs)), nil
}

func (ks *CollectionsKeyspace) Indexer(name IndexType) (Indexer, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Indexer()")
}

func (ks *CollectionsKeyspace) Indexers() ([]Indexer, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Indexers()")
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
func (ks *CollectionsKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Insert()")
}

func (ks *CollectionsKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Update()")
}

func (ks *CollectionsKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Upsert()")
}

func (ks *CollectionsKeyspace) Delete(deletes []string, context QueryContext) ([]string, errors.Error) {
	return nil, errors.NewNotImplemented("CollectionsKeyspace.Delete()")
}

func (ks *CollectionsKeyspace) Release() {
	// do nothing
}
