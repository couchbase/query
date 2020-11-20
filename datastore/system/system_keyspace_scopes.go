//  Copyright (c) 2020 Couchbase, Inc.
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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type scopeKeyspace struct {
	keyspaceBase
	skipSystem bool
	store      datastore.Datastore
	indexer    datastore.Indexer
}

func (b *scopeKeyspace) Release(close bool) {
}

func (b *scopeKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *scopeKeyspace) Id() string {
	return b.Name()
}

func (b *scopeKeyspace) Name() string {
	return b.name
}

func (b *scopeKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var bucket datastore.Bucket
	var err errors.Error

	count := int64(0)
	namespaceIds, excp := b.store.NamespaceIds()
	canAccessAll := canAccessSystemTables(context)
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.store.NamespaceById(namespaceId)
			if excp == nil {
				objects, excp := namespace.Objects(true)
				if excp == nil {
					for _, object := range objects {
						excludeResult := !canAccessAll && !canRead(context, namespaceId, object.Id)

						// The list of bucket ids can include memcached buckets.
						// We do not want to include them in the count of
						// of queryable buckets. Attempting to retrieve the bucket
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						if object.IsBucket {
							bucket, err = namespace.BucketByName(object.Id)
							if err != nil {
								continue
							}
						} else {
							continue
						}
						if excludeResult {
							context.Warning(errors.NewSystemFilteredRowsWarning("system:scopes"))
						} else {

							scopeIds, _ := bucket.ScopeIds()
							for _, scopeId := range scopeIds {
								scope, _ := bucket.ScopeById(scopeId)
								if scope != nil {

									if !canAccessAll && !canRead(context, namespaceId, object.Id, scopeId) {
										context.Warning(errors.NewSystemFilteredRowsWarning("system:scopes"))
									} else {
										count++
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

func (b *scopeKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *scopeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *scopeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *scopeKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {
	var e errors.Error
	var item value.AnnotatedValue

	canAccessAll := canAccessSystemTables(context)
	for _, k := range keys {
		err, elems := splitScopeId(k)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !canAccessAll && !canRead(context, elems[0], elems[1], elems[2]) {
			context.Warning(errors.NewSystemFilteredRowsWarning("system:scopes"))
			continue
		}
		item, e = b.fetchOne(elems[0], elems[1], elems[2])

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

func (b *scopeKeyspace) fetchOne(ns, bn, sn string) (value.AnnotatedValue, errors.Error) {
	var err errors.Error
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var scope datastore.Scope

	// this should never happen, but if it does, we skip silently system collections
	// (not an error, they are just not part of the result set)
	if b.skipSystem && sn[0] == '_' {
		return nil, nil
	}
	namespace, err = b.store.NamespaceById(ns)
	if namespace != nil {
		bucket, err = namespace.BucketById(bn)
		if bucket != nil {
			scope, err = bucket.ScopeById(sn)
			if scope != nil {
				doc := value.NewAnnotatedValue(map[string]interface{}{
					"datastore_id": namespace.DatastoreId(),
					"namespace_id": namespace.Id(),
					"namespace":    namespace.Name(),
					"bucket":       bucket.Name(),
					"name":         scope.Name(),
					"path":         path(namespace.Name(), bucket.Name(), scope.Name()),
				})
				return doc, nil
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func (b *scopeKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *scopeKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *scopeKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *scopeKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func newScopesKeyspace(p *namespace, store datastore.Datastore, name string, skipSystem bool) (*scopeKeyspace, errors.Error) {
	b := new(scopeKeyspace)
	b.store = store
	b.skipSystem = skipSystem
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &scopeIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

type scopeIndex struct {
	indexBase
	name     string
	keyspace *scopeKeyspace
}

func (pi *scopeIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *scopeIndex) Id() string {
	return pi.Name()
}

func (pi *scopeIndex) Name() string {
	return pi.name
}

func (pi *scopeIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *scopeIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *scopeIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *scopeIndex) Condition() expression.Expression {
	return nil
}

func (pi *scopeIndex) IsPrimary() bool {
	return true
}

func (pi *scopeIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *scopeIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *scopeIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func splitScopeId(id string) (errors.Error, []string) {
	ids := strings.SplitN(id, "/", 3)
	if len(ids) != 3 {
		return errors.NewSystemMalformedKeyError(id, "system:scopes"), nil
	}
	return nil, ids
}

func (pi *scopeIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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
					objects, err = namespace.Objects(true)
					if err == nil {
						for _, object := range objects {

							// The list of bucket ids can include memcached buckets.
							// We do not want to include them in the list
							// of queryable buckets. Attempting to retrieve the bucket
							// record of a memcached bucket returns an error,
							// which allows us to distinguish these buckets, and exclude them.
							// See MB-19364 for more info.
							if object.IsBucket {
								bucket, err = namespace.BucketByName(object.Id)
								if err != nil {
									continue
								}
							} else {
								continue
							}

							scopeIds, _ := bucket.ScopeIds()
							for _, scopeId := range scopeIds {
								scope, _ := bucket.ScopeById(scopeId)
								if scope != nil {
									id := makeId(namespaceId, object.Id, scopeId)
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

func (pi *scopeIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
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
				objects, err = namespace.Objects(true)
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
						} else {
							continue
						}

						scopeIds, _ := bucket.ScopeIds()
						for _, scopeId := range scopeIds {
							scope, _ := bucket.ScopeById(scopeId)
							if scope != nil {
								id := makeId(namespaceId, object.Id, scopeId)
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
