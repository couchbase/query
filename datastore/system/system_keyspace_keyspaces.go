//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type keyspaceKeyspace struct {
	keyspaceBase
	skipSystem bool
	store      datastore.Datastore
	indexer    datastore.Indexer
}

func (b *keyspaceKeyspace) Release(close bool) {
}

func (b *keyspaceKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *keyspaceKeyspace) Id() string {
	return b.Name()
}

func (b *keyspaceKeyspace) Name() string {
	return b.name
}

func canAccessSystemTables(context datastore.QueryContext) bool {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
	_, err := datastore.GetDatastore().Authorize(privs, context.Credentials())
	res := err == nil
	return res

}

func (b *keyspaceKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var bucket datastore.Bucket
	var err errors.Error

	count := int64(0)
	namespaceIds, excp := b.store.NamespaceIds()
	canAccessAll := canAccessSystemTables(context)
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.store.NamespaceById(namespaceId)
			if excp == nil {
				objects, excp := namespace.Objects()
				if excp == nil {
					for _, object := range objects {
						excludeResult := !canAccessAll && !canRead(context, namespaceId, object.Id)

						// The list of bucket ids can include memcached buckets.
						// We do not want to include them in the count of
						// of queryable buckets. Attempting to retrieve the bucket
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						if object.IsKeyspace {
							_, err = namespace.KeyspaceByName(object.Id)
							if err != nil {
								continue
							}
						}
						if object.IsBucket {
							bucket, err = namespace.BucketByName(object.Id)
							if err != nil {
								continue
							}
						}
						if excludeResult {
							context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
						} else {

							if object.IsKeyspace {
								count++
							}
							if object.IsBucket {
								scopeIds, _ := bucket.ScopeIds()
								for _, scopeId := range scopeIds {
									scope, _ := bucket.ScopeById(scopeId)
									if scope != nil {
										keyspaceIds, _ := scope.KeyspaceIds()
										for _, keyspaceId := range keyspaceIds {

											if !canAccessAll && !canRead(context, namespaceId, object.Id, scopeId, keyspaceId) {
												context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
											} else {
												count++
											}
										}
									}
								}
							}
						}
					}
				} else {
					return 0, errors.NewSystemDatastoreError(excp, "")
				}
			} else {
				return 0, errors.NewSystemDatastoreError(excp, "")
			}
		}
		return count, nil
	}
	return 0, errors.NewSystemDatastoreError(excp, "")
}

func (b *keyspaceKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *keyspaceKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *keyspaceKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *keyspaceKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
	var e errors.Error
	var item value.AnnotatedValue

	canAccessAll := canAccessSystemTables(context)
	for _, k := range keys {
		err, elems := splitId(k)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(elems) == 2 {
			if !canAccessAll && !canRead(context, elems[0], elems[1]) {
				context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
				continue
			}
			item, e = b.fetchOne(elems[0], elems[1])
		} else {
			if !canAccessAll && !canRead(context, elems[0], elems[1], elems[2], elems[3]) {
				context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
				continue
			}
			item, e = b.fetchOneCollection(elems[0], elems[1], elems[2], elems[3])
		}

		if e != nil {
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.NewMeta()["keyspace"] = b.fullName
			item.SetId(k)
		}

		keysMap[k] = item
	}

	return
}

// we should take it from algebra, but we need to avoid circular references
func path(elems ...string) string {
	out := elems[0] + ":" + elems[1]
	for i := 2; i < len(elems); i++ {
		out = out + "." + elems[i]
	}
	return out
}

func (b *keyspaceKeyspace) fetchOne(ns string, ks string) (value.AnnotatedValue, errors.Error) {
	namespace, err := b.store.NamespaceById(ns)
	if namespace != nil {
		keyspace, err := namespace.KeyspaceById(ks)
		if keyspace != nil {
			doc := value.NewAnnotatedValue(map[string]interface{}{
				"datastore_id": namespace.DatastoreId(),
				"namespace_id": namespace.Id(),
				"namespace":    namespace.Name(),
				"id":           keyspace.Id(),
				"name":         keyspace.Name(),
				"path":         path(namespace.Name(), keyspace.Name()),
			})
			return doc, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func (b *keyspaceKeyspace) fetchOneCollection(ns, bn, sn, ks string) (value.AnnotatedValue, errors.Error) {
	var err errors.Error
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var scope datastore.Scope
	var keyspace datastore.Keyspace

	// this should never happen, but if it does, we skip silently system collections
	// (not an error, they are just not part of the result set)
	if b.skipSystem && ks[0] == '_' {
		return nil, nil
	}
	namespace, err = b.store.NamespaceById(ns)
	if namespace != nil {
		bucket, err = namespace.BucketById(bn)
		if bucket != nil {
			scope, err = bucket.ScopeById(sn)
			if scope != nil {
				keyspace, err = scope.KeyspaceById(ks)
				if keyspace != nil {
					doc := value.NewAnnotatedValue(map[string]interface{}{
						"datastore_id": namespace.DatastoreId(),
						"namespace_id": namespace.Id(),
						"namespace":    namespace.Name(),
						"id":           keyspace.Id(),
						"name":         keyspace.Name(),
						"bucket":       bucket.Name(),
						"scope":        scope.Name(),
						"path":         path(namespace.Name(), bucket.Name(), scope.Name(), keyspace.Name()),
					})
					return doc, nil
				}
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func (b *keyspaceKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *keyspaceKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newKeyspacesKeyspace(p *namespace, store datastore.Datastore, name string, skipSystem bool) (*keyspaceKeyspace, errors.Error) {
	b := new(keyspaceKeyspace)
	b.store = store
	b.skipSystem = skipSystem
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &keyspaceIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type keyspaceIndex struct {
	indexBase
	name     string
	keyspace *keyspaceKeyspace
}

func (pi *keyspaceIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *keyspaceIndex) Id() string {
	return pi.Name()
}

func (pi *keyspaceIndex) Name() string {
	return pi.name
}

func (pi *keyspaceIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *keyspaceIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *keyspaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *keyspaceIndex) IsPrimary() bool {
	return true
}

func (pi *keyspaceIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *keyspaceIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *keyspaceIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func makeId(elems ...string) string {
	return strings.Join(elems, "/")
}

func splitId(id string) (errors.Error, []string) {
	ids := strings.SplitN(id, "/", 4)
	if len(ids) != 2 && len(ids) != 4 {
		return errors.NewSystemMalformedKeyError(id, "system:keyspaces"), nil
	}
	return nil, ids
}

func (pi *keyspaceIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var objects []datastore.Object

	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		var numProduced int64 = 0
		namespaceIds, err := pi.keyspace.store.NamespaceIds()
		if err == nil {

		loop:
			for _, namespaceId := range namespaceIds {
				namespace, err = pi.keyspace.store.NamespaceById(namespaceId)
				if err == nil {
					objects, err = namespace.Objects()
					if err == nil {
						for _, object := range objects {

							// The list of bucket ids can include memcached buckets.
							// We do not want to include them in the list
							// of queryable buckets. Attempting to retrieve the bucket
							// record of a memcached bucket returns an error,
							// which allows us to distinguish these buckets, and exclude them.
							// See MB-19364 for more info.
							if object.IsKeyspace {
								_, err = namespace.KeyspaceByName(object.Id)
								if err != nil {
									continue
								}
							}
							if object.IsBucket {
								bucket, err = namespace.BucketByName(object.Id)
								if err != nil {
									continue
								}
							}

							if object.IsKeyspace {
								id := makeId(namespaceId, object.Id)
								if spanEvaluator.evaluate(id) {
									entry := datastore.IndexEntry{PrimaryKey: id}
									if !sendSystemKey(conn, &entry) {
										return
									}
									numProduced++
									if limit > 0 && numProduced >= limit {
										break loop
									}
								}
							}
							if object.IsBucket {
								scopeIds, _ := bucket.ScopeIds()
								for _, scopeId := range scopeIds {
									scope, _ := bucket.ScopeById(scopeId)
									if scope != nil {
										keyspaceIds, _ := scope.KeyspaceIds()
										for _, keyspaceId := range keyspaceIds {
											if pi.keyspace.skipSystem && keyspaceId[0] == '_' {
												continue
											}
											id := makeId(namespaceId, object.Id, scopeId, keyspaceId)
											if spanEvaluator.evaluate(id) {
												entry := datastore.IndexEntry{PrimaryKey: id}
												if !sendSystemKey(conn, &entry) {
													return
												}
												numProduced++
												if limit > 0 && numProduced >= limit {
													break loop
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func (pi *keyspaceIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var objects []datastore.Object

	defer conn.Sender().Close()

	var numProduced int64 = 0
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {

	loop:
		for _, namespaceId := range namespaceIds {
			namespace, err = pi.keyspace.store.NamespaceById(namespaceId)
			if err == nil {
				objects, err = namespace.Objects()
				if err == nil {
					for _, object := range objects {

						// The list of buckets ids can include memcached buckets.
						// We do not want to include them in the list
						// of queryable buckets. Attempting to retrieve the bucket
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						if object.IsKeyspace {
							_, err = namespace.KeyspaceByName(object.Id)
							if err != nil {
								continue
							}
						}
						if object.IsBucket {
							bucket, err = namespace.BucketByName(object.Id)
							if err != nil {
								continue
							}
						}

						if object.IsKeyspace {
							id := makeId(namespaceId, object.Id)
							entry := datastore.IndexEntry{PrimaryKey: id}
							if !sendSystemKey(conn, &entry) {
								return
							}
							numProduced++
							if limit > 0 && numProduced >= limit {
								break loop
							}
						}
						if object.IsBucket {
							scopeIds, _ := bucket.ScopeIds()
							for _, scopeId := range scopeIds {
								scope, _ := bucket.ScopeById(scopeId)
								if scope != nil {
									keyspaceIds, _ := scope.KeyspaceIds()
									for _, keyspaceId := range keyspaceIds {
										if pi.keyspace.skipSystem && keyspaceId[0] == '_' {
											continue
										}
										id := makeId(namespaceId, object.Id, scopeId, keyspaceId)
										entry := datastore.IndexEntry{PrimaryKey: id}
										if !sendSystemKey(conn, &entry) {
											return
										}
										numProduced++
										if limit > 0 && numProduced >= limit {
											break loop
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}
