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

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/gomemcached"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

var _COLLECTIONS_SUPPORTED bool = false

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

func (sc *scope) KeyspaceById(id string) (datastore.Keyspace, errors.Error) {
	ks := sc.keyspaces[id]
	if ks == nil {
		return nil, errors.NewCbKeyspaceNotFoundError(nil, id)
	}
	return ks, nil
}

func (sc *scope) KeyspaceByName(name string) (datastore.Keyspace, errors.Error) {
	for _, v := range sc.keyspaces {
		if name == v.Name() {
			return v, nil
		}
	}
	return nil, errors.NewCbKeyspaceNotFoundError(nil, name)
}

type collection struct {
	id        string
	uid       uint32
	namespace *namespace
	scope     *scope
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
	return 0, errors.NewNotImplemented("collection.Count()")
}

func (coll *collection) Size(context datastore.QueryContext) (int64, errors.Error) {
	return 0, errors.NewNotImplemented("collection.Size()")
}

func (coll *collection) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Indexer()")
}

func (coll *collection) Indexers() ([]datastore.Indexer, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Indexers()")
}

func (coll *collection) Fetch(keys []string, fetchMap map[string]value.AnnotatedValue, context datastore.QueryContext, subPaths []string) []errors.Error {
	var noVirtualDocAttr bool
	var bulkResponse map[string]*gomemcached.MCResponse
	var mcr *gomemcached.MCResponse
	var err error

	l := len(keys)
	if l == 0 {
		return nil
	}

	cbbucket := coll.scope.bucket.cbbucket
	ls := len(subPaths)
	fast := l == 1 && ls == 0
	if fast {
		mcr, err = cbbucket.GetsMCFromCollection(coll.uid, keys[0], context.GetReqDeadline())
	} else {
		if ls > 0 && subPaths[0] != "$document" {
			subPaths = append([]string{"$document"}, subPaths...)
			noVirtualDocAttr = true
		}

		if l == 1 {
			// TODO: fetch from collection using subdoc.
			//mcr, err = cbbucket.GetsSubDoc(keys[0], context.GetReqDeadline(), subPaths)
			mcr = nil
			err = fmt.Errorf("Subdoc fetch not supported for collections")
		} else {
			// TODO: fetch from collection using GetBulk
			//bulkResponse, err = cbbucket.GetBulk(keys, context.GetReqDeadline(), subPaths)
			//defer cbbucket.ReleaseGetBulkPools(bulkResponse)
			bulkResponse = nil
			err = fmt.Errorf("Bulk GET not supported for collections")
		}
	}

	if err != nil {
		coll.scope.bucket.checkRefresh(err)

		// Ignore "Not found" keys
		if !isNotFoundError(err) {
			if cb.IsReadTimeOutError(err) {
				logging.Errorf(err.Error())
			}
			return []errors.Error{errors.NewCbBulkGetError(err, "")}
		}
	}

	if fast {
		if mcr != nil && err == nil {
			fetchMap[keys[0]] = doFetch(keys[0], mcr)
		}

	} else if l == 1 {
		if mcr != nil && err == nil {
			fetchMap[keys[0]] = getSubDocFetchResults(keys[0], mcr, subPaths, noVirtualDocAttr)
		}
	} else {
		i := 0
		if ls > 0 {
			for k, v := range bulkResponse {
				fetchMap[k] = getSubDocFetchResults(k, v, subPaths, noVirtualDocAttr)
				i++
			}
		} else {
			for k, v := range bulkResponse {
				fetchMap[k] = doFetch(k, v)
				i++
			}
			logging.Debugf("Requested keys %d Fetched %d keys ", l, i)
		}
	}

	return nil
}

// Used by DML statements
// For insert and upsert, nil input keys are replaced with auto-generated keys
func (coll *collection) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Insert()")
}

func (coll *collection) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Update()")
}

func (coll *collection) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Upsert()")
}

func (coll *collection) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	return nil, errors.NewNotImplemented("collection.Delete()")
}

func (coll *collection) Release(close bool) {
	// do nothing
}

func buildScopesAndCollections(mani *cb.Manifest, bucket *keyspace) map[string]*scope {
	scopes := make(map[string]*scope, len(mani.Scopes))
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
		}
		scopes[s.Name] = scope
	}
	return scopes
}
