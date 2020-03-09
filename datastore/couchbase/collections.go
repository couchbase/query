//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import (
	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached/client" // package name is memcached

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

// TODO remove
var _COLLECTIONS_SUPPORTED bool = true

var _COLLECTIONS_NOT_SUPPORTED string = "Collections are not yet supported."

type scope struct {
	id     string
	bucket *keyspace

	keyspaces map[string]*collection // keyspaces by id
}

func NewScope(id string) *scope {
	return &scope{
		id:        id,
		keyspaces: make(map[string]*collection),
	}
}

func (sc *scope) AddKeyspace(coll *collection) {
	sc.keyspaces[coll.Id()] = coll
	coll.namespace = nil
	coll.scope = sc
}

func (sc *scope) Id() string {
	return sc.id
}

func (sc *scope) Name() string {
	return sc.id
}

func (sc *scope) BucketId() string {
	return sc.bucket.Id()
}

func (sc *scope) Bucket() datastore.Bucket {
	return sc.bucket
}

func (sc *scope) KeyspaceIds() ([]string, errors.Error) {
	ids := make([]string, len(sc.keyspaces))
	ix := 0
	for k := range sc.keyspaces {
		ids[ix] = k
		ix++
	}
	return ids, nil
}

func (sc *scope) KeyspaceNames() ([]string, errors.Error) {
	ids := make([]string, len(sc.keyspaces))
	ix := 0
	for _, v := range sc.keyspaces {
		ids[ix] = v.Name()
		ix++
	}
	return ids, nil
}

func (sc *scope) objectFullName(id string) string {
	return fullName(sc.bucket.namespace.name, sc.bucket.name, sc.id, id)
}

func (sc *scope) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	ks := sc.keyspaces[id]
	if ks == nil {
		return nil, errors.NewCbKeyspaceNotFoundError(nil, sc.objectFullName(id))
	}
	return ks, nil
}

func (sc *scope) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	for _, v := range sc.keyspaces {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbKeyspaceNotFoundError(nil, sc.objectFullName(name))
}

func (sc *scope) CreateCollection(name string) errors.Error {
	err := sc.bucket.cbbucket.CreateCollection(sc.id, name)
	if err != nil {
		return errors.NewCbBucketCreateCollectionError(sc.objectFullName(name), err)
	}
	sc.bucket.setNeedsManifest()
	return nil
}

func (sc *scope) DropCollection(name string) errors.Error {
	err := sc.bucket.cbbucket.DropCollection(sc.id, name)
	if err != nil {
		return errors.NewCbBucketDropCollectionError(sc.objectFullName(name), err)
	}
	sc.bucket.setNeedsManifest()
	return nil
}

type collection struct {
	id        string
	uid       uint32
	namespace *namespace
	scope     *scope
	bucket    *keyspace
	isDefault bool
}

func NewCollection(name string) *collection {
	coll := &collection{
		id: name,
	}
	return coll
}

func (coll *collection) Id() string {
	return coll.id
}

func (coll *collection) Name() string {
	return coll.id
}

func (coll *collection) NamespaceId() string {
	if coll.namespace == nil {
		return ""
	}
	return coll.namespace.Id()
}

func (coll *collection) Namespace() datastore.Namespace {
	return coll.namespace
}

func (coll *collection) ScopeId() string {
	if coll.scope == nil {
		return ""
	}
	return coll.scope.Id()
}

func (coll *collection) Scope() datastore.Scope {
	return coll.scope
}

func (coll *collection) Count(context datastore.QueryContext) (int64, errors.Error) {

	// default collection
	if coll.isDefault {
		return coll.bucket.count(context, &memcached.ClientContext{CollId: coll.uid})
	}
	return -1, errors.NewNotImplemented("collection.Count()")
}

func (coll *collection) Size(context datastore.QueryContext) (int64, errors.Error) {

	// default collection
	if coll.isDefault {
		return coll.bucket.size(context, &memcached.ClientContext{CollId: coll.uid})
	}
	return -1, errors.NewNotImplemented("collection.Size()")
}

func (coll *collection) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {

	// default collection
	if coll.isDefault {
		k := datastore.Keyspace(coll.bucket)
		return k.Indexer(name)
	}
	return nil, errors.NewNotImplemented("collection.Indexer()")
}

func (coll *collection) Indexers() ([]datastore.Indexer, errors.Error) {

	// default collection
	if coll.isDefault {
		k := datastore.Keyspace(coll.bucket)
		return k.Indexers()
	}
	return nil, errors.NewNotImplemented("collection.Indexers()")
}

func (coll *collection) GetRandomEntry() (string, value.Value, errors.Error) {

	// default collection
	if coll.isDefault {
		return coll.bucket.getRandomEntry(&memcached.ClientContext{CollId: coll.uid})
	}
	return "", nil, errors.NewNotImplemented("collection.GetRandomEntry()")
}

func (coll *collection) Fetch(keys []string, fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string) []errors.Error {
	return coll.bucket.fetch(keys, fetchMap, context, subPaths, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(INSERT, inserts, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(UPDATE, updates, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(UPSERT, upserts, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	return coll.bucket.delete(deletes, context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Release() {
	// do nothing
}

func (coll *collection) Flush() errors.Error {
	err := coll.bucket.cbbucket.FlushCollection(coll.scope.id, coll.id)
	if err != nil {
		return errors.NewCbBucketFlushCollectionError(coll.scope.objectFullName(coll.id), err)
	}
	return nil
}

func buildScopesAndCollections(mani *cb.Manifest, bucket *keyspace) (map[string]*scope, datastore.Keyspace) {
	scopes := make(map[string]*scope, len(mani.Scopes))
	var defaultCollection *collection

	for _, s := range mani.Scopes {
		scope := &scope{
			id:        s.Name,
			bucket:    bucket,
			keyspaces: make(map[string]*collection, len(s.Collections)),
		}
		for _, c := range s.Collections {
			coll := &collection{
				id:        c.Name,
				namespace: bucket.namespace,
				uid:       uint32(c.Uid),
				scope:     scope,
			}
			scope.keyspaces[c.Name] = coll
			coll.bucket = bucket
			if s.Uid == 0 && c.Uid == 0 {
				coll.isDefault = true

				// the default collection has the bucket name to represent itself as the bucket
				// this is to differentiate from the default collection being addressed explicitly
				defaultCollection = &collection{
					id:        bucket.name,
					namespace: bucket.namespace,
					uid:       uint32(c.Uid),
					scope:     scope,
					bucket:    bucket,
					isDefault: true,
				}
			}
		}
		scopes[s.Name] = scope
	}
	return scopes, defaultCollection
}
