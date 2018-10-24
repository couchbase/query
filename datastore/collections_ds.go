package datastore

import (
	"github.com/couchbase/go_json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type CollectionsNamespace struct {
	Keyspaces map[string]*CollectionsKeyspace // keyspaces by Id
}

func NewCollectionsNamespace() *CollectionsNamespace {
	// No keyspaces, for now.
	cns := &CollectionsNamespace{Keyspaces: make(map[string]*CollectionsKeyspace)}

	ksName := "test"
	ks := NewCollectionsKeyspace(ksName, cns)
	cns.Keyspaces[ksName] = ks
	ks.addDocument("v1", `{ "f1" : "string value A", "f2" : 10 }`)
	ks.addDocument("v2", `{ "f1" : "string value B", "f2" : 11 }`)
	ks.addDocument("v3", `{ "f1" : "string value C", "f2" : 12 }`)

	return cns
}

func (ns *CollectionsNamespace) DatastoreId() string {
	return "http://127.0.0.1:8094"
}

func (ns *CollectionsNamespace) Id() string {
	return "collections"
}

func (ns *CollectionsNamespace) Name() string {
	return "collections"
}

func (ns *CollectionsNamespace) KeyspaceIds() ([]string, errors.Error) {
	ret := make([]string, len(ns.Keyspaces))
	i := 0
	for _, v := range ns.Keyspaces {
		ret[i] = v.Id()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) KeyspaceNames() ([]string, errors.Error) {
	ret := make([]string, len(ns.Keyspaces))
	i := 0
	for _, v := range ns.Keyspaces {
		ret[i] = v.Name()
		i++
	}
	return ret, nil
}

func (ns *CollectionsNamespace) KeyspaceById(id string) (Keyspace, errors.Error) {
	ks := ns.Keyspaces[id]
	if ks == nil {
		return nil, errors.NewCbKeyspaceNotFoundError(nil, id)
	}
	return ks, nil
}

func (ns *CollectionsNamespace) KeyspaceByName(name string) (Keyspace, errors.Error) {
	for _, v := range ns.Keyspaces {
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
	return NO_STRINGS, nil
}

func (ns *CollectionsNamespace) BucketNames() ([]string, errors.Error) {
	return NO_STRINGS, nil
}

func (ns *CollectionsNamespace) BucketById(name string) (Bucket, errors.Error) {
	return nil, errors.NewNotImplemented("collections BucketById")
}

func (ns *CollectionsNamespace) BucketByName(name string) (Bucket, errors.Error) {
	return nil, errors.NewNotImplemented("collection BucketByName")
}

type CollectionsKeyspace struct {
	id   string
	ns   *CollectionsNamespace
	docs map[string]value.AnnotatedValue
}

func NewCollectionsKeyspace(name string, namespace *CollectionsNamespace) *CollectionsKeyspace {
	ks := &CollectionsKeyspace{
		id:   name,
		ns:   namespace,
		docs: make(map[string]value.AnnotatedValue),
	}
	return ks
}

func (ks *CollectionsKeyspace) addDocument(id string, jsonDoc string) {
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
	return ks.ns.Name()
}

func (ks *CollectionsKeyspace) Namespace() Namespace {
	return ks.ns
}

func (ks *CollectionsKeyspace) ScopeId() string {
	return ""
}

func (ks *CollectionsKeyspace) Scope() Scope {
	return nil
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
