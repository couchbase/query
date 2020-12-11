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
	"fmt"
	"io"
	"strconv"
	"sync"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached/client" // package name is memcached
	gsi "github.com/couchbase/indexing/secondary/queryport/n1ql"
	ftsclient "github.com/couchbase/n1fty"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	functions "github.com/couchbase/query/functions/metakv"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const _DEFAULT_SCOPE_COLLECTION_NAME = "._default._default"

type scope struct {
	id      string
	authKey string
	bucket  *keyspace

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

func (sc *scope) AuthKey() string {
	return sc.authKey
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
	sync.Mutex
	id         string
	name       string
	uid        uint32
	uidString  string
	namespace  *namespace
	scope      *scope
	bucket     *keyspace
	fullName   string
	authKey    string
	checked    bool
	gsiIndexer datastore.Indexer
	ftsIndexer datastore.Indexer
	chkIndex   chkIndexDict
	isDefault  bool
	isBucket   bool
}

func (coll *collection) Id() string {
	return coll.id
}

func (coll *collection) Name() string {
	return coll.name
}

func (coll *collection) QualifiedName() string {
	if coll.isBucket {
		return coll.fullName + _DEFAULT_SCOPE_COLLECTION_NAME
	}
	return coll.fullName
}

func (coll *collection) AuthKey() string {
	return coll.authKey
}

func (coll *collection) Uid() string {
	return coll.uidString
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

func (coll *collection) Stats(context datastore.QueryContext, which []datastore.KeyspaceStats) ([]int64, errors.Error) {
	return coll.bucket.stats(context, which, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Count(context datastore.QueryContext) (int64, errors.Error) {
	return coll.bucket.count(context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Size(context datastore.QueryContext) (int64, errors.Error) {
	return coll.bucket.size(context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {

	// default collection
	if coll.isDefault {
		k := datastore.Keyspace(coll.bucket)
		return k.Indexer(name)
	}

	coll.loadIndexes()
	switch name {
	case datastore.GSI, datastore.DEFAULT:
		if coll.gsiIndexer != nil {
			return coll.gsiIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("GSI may not be enabled"))
	case datastore.FTS:
		if coll.ftsIndexer != nil {
			return coll.ftsIndexer, nil
		}
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("FTS may not be enabled"))
	default:
		return nil, errors.NewCbIndexerNotImplementedError(nil, fmt.Sprintf("Type %s", name))
	}
}

func (coll *collection) Indexers() ([]datastore.Indexer, errors.Error) {
	var err errors.Error

	// default collection
	if coll.isDefault {
		k := datastore.Keyspace(coll.bucket)
		return k.Indexers()
	}

	coll.loadIndexes()
	indexers := make([]datastore.Indexer, 0, 2)

	if coll.gsiIndexer != nil {
		indexers = append(indexers, coll.gsiIndexer)
		err = checkIndexCache(coll.id, coll.gsiIndexer, &coll.chkIndex)
	}
	if coll.ftsIndexer != nil {
		indexers = append(indexers, coll.ftsIndexer)
	}
	return indexers, err
}

func (coll *collection) loadIndexes() {
	var qerr errors.Error

	if coll.checked {
		return
	}
	coll.Lock()
	defer coll.Unlock()

	// somebody may have already done it while we were waiting
	if coll.checked {
		return
	}

	namespace := coll.namespace
	store := namespace.store
	connSecConfig := store.connSecConfig
	coll.gsiIndexer, qerr = gsi.NewGSIIndexer2(store.URL(), namespace.name, coll.bucket.name, coll.scope.id, coll.id, connSecConfig)
	if qerr != nil {
		logging.Warnf("Error loading GSI indexes for keyspace %s. Error %v", coll.id, qerr)
	} else {
		coll.gsiIndexer.SetConnectionSecurityConfig(connSecConfig)
	}

	// FTS indexer
	coll.ftsIndexer, qerr = ftsclient.NewFTSIndexer2(store.URL(), namespace.name, coll.bucket.name, coll.scope.id, coll.id)
	if qerr != nil {
		logging.Warnf("Error loading FTS indexes for keyspace %s. Error %v", coll.id, qerr)
	} else {
		coll.ftsIndexer.SetConnectionSecurityConfig(connSecConfig)
	}
	coll.checked = true
}

func (coll *collection) GetRandomEntry() (string, value.Value, errors.Error) {
	return coll.bucket.getRandomEntry(coll.scope.id, coll.id, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Fetch(keys []string, fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string) []errors.Error {

	return coll.bucket.fetch(coll.fullName, coll.QualifiedName(), coll.scope.id, coll.id,
		keys, fetchMap, context, subPaths, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(MOP_INSERT, coll.QualifiedName(), coll.scope.id, coll.id,
		inserts, context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(MOP_UPDATE, coll.QualifiedName(), coll.scope.id, coll.id,
		updates, context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(MOP_UPSERT, coll.QualifiedName(), coll.scope.id, coll.id,
		upserts, context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	return coll.bucket.performOp(MOP_DELETE, coll.QualifiedName(), coll.scope.id, coll.id,
		deletes, context, &memcached.ClientContext{CollId: coll.uid})
}

func (coll *collection) Release(bclose bool) {
	// close an ftsIndexer that belongs to this keyspace
	if ftsIndexerCloser, ok := coll.ftsIndexer.(io.Closer); ok {
		// FTSIndexer implements a Close() method
		ftsIndexerCloser.Close()
	}
}

func (coll *collection) Flush() errors.Error {
	err := coll.bucket.cbbucket.FlushCollection(coll.scope.id, coll.id)
	if err != nil {
		return errors.NewCbBucketFlushCollectionError(coll.scope.objectFullName(coll.id), err)
	}
	return nil
}

func (coll *collection) IsBucket() bool {
	return coll.isBucket
}

func buildScopesAndCollections(mani *cb.Manifest, bucket *keyspace) (map[string]*scope, datastore.Keyspace) {
	scopes := make(map[string]*scope, len(mani.Scopes))
	var defaultCollection *collection

	for _, s := range mani.Scopes {
		scope := &scope{
			id:        s.Name,
			bucket:    bucket,
			keyspaces: make(map[string]*collection, len(s.Collections)),
			authKey:   bucket.name + ":" + s.Name,
		}
		for _, c := range s.Collections {
			coll := &collection{
				id:        c.Name,
				name:      c.Name,
				namespace: bucket.namespace,
				fullName:  bucket.namespace.name + ":" + bucket.name + "." + s.Name + "." + c.Name,
				uid:       uint32(c.Uid),
				uidString: strconv.FormatUint(c.Uid, 16),
				scope:     scope,
			}
			scope.keyspaces[c.Name] = coll
			coll.bucket = bucket
			if s.Uid == 0 && c.Uid == 0 {
				coll.isDefault = true

				// the authorization key for the default collection is the bucket
				coll.authKey = bucket.name

				// the default collection has the bucket name to represent itself as the bucket
				// this is to differentiate from the default collection being addressed explicitly
				defaultCollection = &collection{
					id:        c.Name,
					name:      bucket.name,
					namespace: bucket.namespace,
					fullName:  bucket.namespace.name + ":" + bucket.name,
					authKey:   bucket.name,
					uid:       uint32(c.Uid),
					uidString: strconv.FormatUint(c.Uid, 16),
					scope:     scope,
					bucket:    bucket,
					isDefault: true,
					isBucket:  true,
				}
			} else {
				coll.authKey = bucket.name + ":" + scope.id + ":" + coll.name
			}
		}
		scopes[s.Name] = scope
	}
	return scopes, defaultCollection
}

func refreshScopesAndCollections(mani *cb.Manifest, bucket *keyspace) (map[string]*scope, datastore.Keyspace) {
	oldScopes := bucket.scopes

	// this shouldn't happen on a refresh, but if there aren't old scopes, just go
	if oldScopes == nil {
		return nil, nil
	}

	scopes := make(map[string]*scope, len(mani.Scopes))
	var defaultCollection *collection

	// check the new scopes
	for _, s := range mani.Scopes {
		scope := &scope{
			id:        s.Name,
			bucket:    bucket,
			keyspaces: make(map[string]*collection, len(s.Collections)),
			authKey:   bucket.name + ":" + s.Name,
		}

		oldScope := oldScopes[s.Name]
		for _, c := range s.Collections {
			coll := &collection{
				id:        c.Name,
				name:      c.Name,
				namespace: bucket.namespace,
				fullName:  bucket.namespace.name + ":" + bucket.name + "." + s.Name + "." + c.Name,
				uid:       uint32(c.Uid),
				uidString: strconv.FormatUint(c.Uid, 16),
				scope:     scope,
			}
			scope.keyspaces[c.Name] = coll
			coll.bucket = bucket

			// copy the indexers
			if oldScope != nil {
				oldColl := oldScope.keyspaces[c.Name]
				if oldColl != nil {
					oldColl.Lock()
					coll.gsiIndexer = oldColl.gsiIndexer
					coll.ftsIndexer = oldColl.ftsIndexer
					coll.checked = oldColl.checked
					oldColl.Unlock()
				}
			}
			if s.Uid == 0 && c.Uid == 0 {
				coll.isDefault = true
				coll.authKey = bucket.name

				// the default collection has the bucket name to represent itself as the bucket
				// this is to differentiate from the default collection being addressed explicitly
				defaultCollection = &collection{
					id:        c.Name,
					name:      bucket.name,
					namespace: bucket.namespace,
					fullName:  bucket.namespace.name + ":" + bucket.name,
					authKey:   bucket.name,
					uid:       uint32(c.Uid),
					uidString: strconv.FormatUint(c.Uid, 16),
					scope:     scope,
					bucket:    bucket,
					isDefault: true,
					isBucket:  true,
				}

				// copy the indexers
				if bucket.defaultCollection != nil {
					switch old := bucket.defaultCollection.(type) {
					case *collection:
						old.Lock()
						coll.gsiIndexer = old.gsiIndexer
						coll.ftsIndexer = old.ftsIndexer
						coll.checked = old.checked
						old.Unlock()
					case *keyspace:
						old.Lock()
						coll.gsiIndexer = old.gsiIndexer
						coll.ftsIndexer = old.ftsIndexer
						coll.checked = old.indexersLoaded
						old.Unlock()
					}
				}
			} else {
				coll.authKey = bucket.name + ":" + scope.id + ":" + coll.name
			}
		}
		scopes[s.Name] = scope

		// clear collections that have disappeared
		if oldScope != nil {
			for n, val := range oldScope.keyspaces {
				if scope.keyspaces[n] == nil {
					DropDictionaryEntry(oldScope.keyspaces[n].QualifiedName())
				}
				if val != nil {
					// invoke Release(..) on collection for any cleanup
					val.Release(false)
				}
			}
		}
	}

	// Clear scopes that have disappeared
	for n, _ := range oldScopes {

		// not here anymore
		if scopes[n] == nil {
			clearOldScope(bucket, oldScopes[n])
		}
	}

	return scopes, defaultCollection
}

func dropDictCacheEntries(bucket *keyspace) {
	for n, s := range bucket.scopes {
		bucket.scopes[n] = nil
		clearOldScope(bucket, s)
	}
}

func clearOldScope(bucket *keyspace, s *scope) {
	for n, val := range s.keyspaces {
		if val != nil {
			s.keyspaces[n] = nil
			DropDictionaryEntry(val.QualifiedName())
			// invoke Release(..) on collection for any cleanup
			val.Release(false)
		}
	}

	functions.DropScope(bucket.namespace.name, bucket.name, s.Name())
}
