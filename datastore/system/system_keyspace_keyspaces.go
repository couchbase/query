//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type keyspaceKeyspace struct {
	keyspaceBase
	skipSystem bool
	info       bool
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

// Checks if the user has permissions to access system keyspaces
// isInternal: whether the authorization check is for an internal action.
func canAccessSystemTables(context datastore.QueryContext, isInternal bool) bool {
	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)

	var err errors.Error
	if isInternal {
		// avoid logging an audit on authorization failures for an internal authorization action
		err = datastore.GetDatastore().AuthorizeInternal(privs, context.Credentials())
	} else {
		err = datastore.GetDatastore().Authorize(privs, context.Credentials())
	}

	res := err == nil
	return res

}

func (b *keyspaceKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var bucket datastore.Bucket
	var err errors.Error

	count := int64(0)
	namespaceIds, excp := b.store.NamespaceIds()

	// since CountScan is only allowed when the user can access system keyspaces
	// do not consider the access check as an internal action
	canAccessAll := canAccessSystemTables(context, false)

	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.store.NamespaceById(namespaceId)
			if excp == nil {
				objects, excp := namespace.Objects(context.Credentials(), nil, true)
				if excp == nil {
					for _, object := range objects {
						includeDefaultKeyspace := canAccessAll || canRead(context, namespace.Datastore(), namespaceId, object.Id)

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

						if object.IsKeyspace {
							if includeDefaultKeyspace {
								count++
							} else {
								context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
							}
						}
						if object.IsBucket {
							scopeIds, _ := bucket.ScopeIds()
							for _, scopeId := range scopeIds {
								scope, _ := bucket.ScopeById(scopeId)
								if scope != nil {
									includeScope := includeDefaultKeyspace ||
										canRead(context, namespace.Datastore(), namespaceId, object.Id, scopeId)
									keyspaceIds, _ := scope.KeyspaceIds()
									for _, keyspaceId := range keyspaceIds {
										if b.skipSystem && keyspaceId[0] == '_' {
											continue
										}
										if includeScope || canRead(context, namespace.Datastore(), namespaceId, object.Id,
											scopeId, keyspaceId) {

											count++
										} else {
											context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
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
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	var e errors.Error
	var item value.AnnotatedValue

	for _, k := range keys {
		err, elems := splitId(k)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(elems) == 2 {
			item, e = b.fetchOne(elems[0], elems[1], context)
		} else {
			item, e = b.fetchOneCollection(elems[0], elems[1], elems[2], elems[3], context)
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

func (b *keyspaceKeyspace) fetchOne(ns string, ks string, context datastore.QueryContext) (value.AnnotatedValue, errors.Error) {
	namespace, err := b.store.NamespaceById(ns)
	if namespace != nil {
		keyspace, err := namespace.KeyspaceById(ks)
		if keyspace != nil {
			doc := value.NewAnnotatedValue(map[string]interface{}{
				"datastore_id": namespace.Datastore().Id(),
				"namespace_id": namespace.Id(),
				"namespace":    namespace.Name(),
				"id":           keyspace.Id(),
				"name":         keyspace.Name(),
				"path":         path(namespace.Name(), keyspace.Name()),
			})
			if b.info {
				var d datastore.Keyspace

				b, ok := keyspace.(datastore.Bucket)
				if ok {
					d, _ = b.DefaultKeyspace()
				}
				if d == nil {
					d = keyspace
				}
				res, err2 := d.Stats(context, []datastore.KeyspaceStats{datastore.KEYSPACE_COUNT, datastore.KEYSPACE_SIZE})
				if err2 == nil {
					doc.SetField("count", res[0])
					doc.SetField("size", res[1])
				}
			}
			return doc, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func (b *keyspaceKeyspace) fetchOneCollection(ns, bn, sn, ks string, context datastore.QueryContext) (
	value.AnnotatedValue, errors.Error) {

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
						"datastore_id": namespace.Datastore().Id(),
						"namespace_id": namespace.Id(),
						"namespace":    namespace.Name(),
						"id":           keyspace.Id(),
						"name":         keyspace.Name(),
						"bucket":       bucket.Name(),
						"scope":        scope.Name(),
						"path":         path(namespace.Name(), bucket.Name(), scope.Name(), keyspace.Name()),
					})
					if keyspace.MaxTTL() != 0 {
						doc.SetField("maxTTL", value.NewValue(keyspace.MaxTTL()))
					}
					if b.info {
						res, err2 := keyspace.Stats(context, []datastore.KeyspaceStats{datastore.KEYSPACE_COUNT,
							datastore.KEYSPACE_SIZE})
						if err2 == nil {
							doc.SetField("count", res[0])
							doc.SetField("size", res[1])
						}
					}
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

func newKeyspacesKeyspace(p *namespace, store datastore.Datastore, name string, skipSystem bool, info bool) (
	*keyspaceKeyspace, errors.Error) {

	b := new(keyspaceKeyspace)
	b.store = store
	b.skipSystem = skipSystem
	b.info = info
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &keyspaceIndex{name: "#primary", keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `bucket_id`
	expr, err := parser.Parse(`bucket_id`)

	if err == nil {
		key := expression.Expressions{expr}
		buckets := &keyspaceIndex{
			name:     "#buckets",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&buckets.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(buckets.name, buckets)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type keyspaceIndex struct {
	indexBase
	name     string
	keyspace *keyspaceKeyspace
	primary  bool
	idxKey   expression.Expressions
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
	return pi.idxKey
}

func (pi *keyspaceIndex) RangeKey2() datastore.IndexKeys {
	if !pi.primary {
		rangeKey := &datastore.IndexKey{
			Expr: pi.idxKey[0],
		}
		rangeKey.SetAttribute(datastore.IK_MISSING, true)
		return datastore.IndexKeys{rangeKey}
	}
	return nil
}

func (pi *keyspaceIndex) Condition() expression.Expression {
	return nil
}

func (pi *keyspaceIndex) IsPrimary() bool {
	return pi.primary
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

	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			conn.Sender().Close()
			return
		}
		pi.scan(requestId, spanEvaluator, limit, cons, vector, conn)
	}
}

func (pi *keyspaceIndex) Scan2(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection,
	ordered bool, projection *datastore.IndexProjection, offset, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	if spans == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {

		spanEvaluator, err := compileSpan2(spans)
		if err != nil {
			conn.Error(err)
			conn.Sender().Close()
			return
		}
		pi.scan(requestId, spanEvaluator, limit, cons, vector, conn)
	}
}

func (pi *keyspaceIndex) scan(requestId string, spanEvaluator compiledSpans, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var objects []datastore.Object
	var numProduced int64 = 0
	var filter func(string) bool

	defer conn.Sender().Close()
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {
		canAccessAll := canAccessSystemTables(conn.QueryContext(), true)
		if !pi.primary && len(spanEvaluator) > 0 && !spanEvaluator.acceptMissing() {
			filter = func(name string) bool {
				return spanEvaluator.evaluate(name)
			}
		}
	loop:
		for _, namespaceId := range namespaceIds {
			namespace, err = pi.keyspace.store.NamespaceById(namespaceId)
			if err == nil {
				objects, err = namespace.Objects(conn.QueryContext().Credentials(), filter, true)
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

						includeDefaultKeyspace := canAccessAll ||
							canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id)
						if object.IsKeyspace && includeDefaultKeyspace {
							var res bool

							id := makeId(namespaceId, object.Id)
							if pi.primary {
								res = spanEvaluator.evaluate(id)
							} else {
								res = spanEvaluator.acceptMissing()
							}
							if res {
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
							if !pi.primary && !spanEvaluator.evaluate(object.Id) {
								continue loop
							}
							scopeIds, _ := bucket.ScopeIds()
							for _, scopeId := range scopeIds {
								scope, _ := bucket.ScopeById(scopeId)
								if scope != nil {
									includeScope := includeDefaultKeyspace ||
										canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id, scopeId)
									keyspaceIds, _ := scope.KeyspaceIds()
									for _, keyspaceId := range keyspaceIds {
										if pi.keyspace.skipSystem && keyspaceId[0] == '_' {
											continue
										}
										id := makeId(namespaceId, object.Id, scopeId, keyspaceId)
										if !pi.primary || spanEvaluator.evaluate(id) {
											if !(includeScope ||
												canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id,
													scopeId, keyspaceId)) {
												conn.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
												continue
											}
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

func (pi *keyspaceIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var objects []datastore.Object

	defer conn.Sender().Close()

	var numProduced int64 = 0
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {
		canAccessAll := canAccessSystemTables(conn.QueryContext(), true)

	loop:
		for _, namespaceId := range namespaceIds {
			namespace, err = pi.keyspace.store.NamespaceById(namespaceId)
			if err == nil {
				objects, err = namespace.Objects(conn.QueryContext().Credentials(), nil, true)
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
						includeDefaultKeyspace := canAccessAll ||
							canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id)
						if object.IsKeyspace && includeDefaultKeyspace {
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
									includeScope := includeDefaultKeyspace ||
										canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id, scopeId)
									keyspaceIds, _ := scope.KeyspaceIds()
									for _, keyspaceId := range keyspaceIds {
										if pi.keyspace.skipSystem && keyspaceId[0] == '_' {
											continue
										}
										if !(includeScope ||
											canRead(conn.QueryContext(), namespace.Datastore(), namespaceId, object.Id, scopeId,
												keyspaceId)) {

											conn.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
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
