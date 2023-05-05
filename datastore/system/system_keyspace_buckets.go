//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type bucketKeyspace struct {
	keyspaceBase
	store   datastore.Datastore
	indexer datastore.Indexer
}

func (b *bucketKeyspace) Release(close bool) {
}

func (b *bucketKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *bucketKeyspace) Id() string {
	return b.Name()
}

func (b *bucketKeyspace) Name() string {
	return b.name
}

func (b *bucketKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var err errors.Error

	count := int64(0)
	namespaceIds, excp := b.store.NamespaceIds()
	canAccessAll := canAccessSystemTables(context)
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.store.NamespaceById(namespaceId)
			if excp == nil {
				objects, excp := namespace.Objects(context.Credentials(), nil, true)
				if excp == nil {
					for _, object := range objects {
						excludeResult := !canAccessAll && !canRead(context, nil, namespaceId, object.Id)

						// The list of bucket ids can include memcached buckets.
						// We do not want to include them in the count of
						// of queryable buckets. Attempting to retrieve the bucket
						// record of a memcached bucket returns an error,
						// which allows us to distinguish these buckets, and exclude them.
						// See MB-19364 for more info.
						if object.IsBucket {
							_, err = namespace.BucketByName(object.Id)
							if err != nil {
								continue
							}
						}
						if excludeResult {
							context.Warning(errors.NewSystemFilteredRowsWarning("system:buckets"))
						} else {
							count++
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

func (b *bucketKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *bucketKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *bucketKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *bucketKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs errors.Errors) {
	var e errors.Error
	var item value.AnnotatedValue

	for _, k := range keys {
		err, elems := splitBucketId(k)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		item, e = b.fetchOne(elems[0], elems[1])

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

func (b *bucketKeyspace) fetchOne(ns string, bn string) (value.AnnotatedValue, errors.Error) {
	namespace, err := b.store.NamespaceById(ns)
	if namespace != nil {
		bucket, err := namespace.BucketById(bn)
		if bucket != nil {
			doc := value.NewAnnotatedValue(map[string]interface{}{
				"datastore_id": namespace.Datastore().Id(),
				"namespace_id": namespace.Id(),
				"namespace":    namespace.Name(),
				"name":         bucket.Name(),
				"path":         path(namespace.Name(), bucket.Name()),
			})
			return doc, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return nil, err
}

func newBucketsKeyspace(p *namespace, store datastore.Datastore, name string) (*bucketKeyspace, errors.Error) {
	b := new(bucketKeyspace)
	b.store = store
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &bucketIndex{name: "#primary", keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)
	// add a secondary index on name
	expr, err := parser.Parse(`name`)

	if err == nil {
		key := expression.Expressions{expr}
		buckets := &bucketIndex{
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

type bucketIndex struct {
	indexBase
	name     string
	keyspace *bucketKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *bucketIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *bucketIndex) Id() string {
	return pi.Name()
}

func (pi *bucketIndex) Name() string {
	return pi.name
}

func (pi *bucketIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *bucketIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *bucketIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *bucketIndex) Condition() expression.Expression {
	return nil
}

func (pi *bucketIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *bucketIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *bucketIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *bucketIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func splitBucketId(id string) (errors.Error, []string) {
	ids := strings.SplitN(id, "/", 2)
	if len(ids) != 2 {
		return errors.NewSystemMalformedKeyError(id, "system:buckets"), nil
	}
	return nil, ids
}

func (pi *bucketIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var objects []datastore.Object

	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		var filter func(string) bool

		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		if !pi.primary {
			filter = func(name string) bool {
				return spanEvaluator.evaluate(name)
			}
		}

		var numProduced int64 = 0
		namespaceIds, err := pi.keyspace.store.NamespaceIds()
		if err == nil {

		loop:
			for _, namespaceId := range namespaceIds {
				namespace, err = pi.keyspace.store.NamespaceById(namespaceId)
				if err == nil {
					objects, err = namespace.Objects(conn.QueryContext().Credentials(), filter, true)
					if err == nil {
						for _, object := range objects {
							if !pi.primary && !spanEvaluator.evaluate(object.Id) {
								continue loop
							}

							// The list of bucket ids can include memcached buckets.
							// We do not want to include them in the list
							// of queryable buckets. Attempting to retrieve the bucket
							// record of a memcached bucket returns an error,
							// which allows us to distinguish these buckets, and exclude them.
							// See MB-19364 for more info.
							if object.IsBucket {
								var res bool

								_, err = namespace.BucketByName(object.Id)
								if err != nil {
									continue
								}
								id := makeId(namespaceId, object.Id)
								if pi.primary {
									res = spanEvaluator.evaluate(id)
								} else {
									res = spanEvaluator.evaluate(object.Id)
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
						}
					}
				}
			}
		}
	}
}

func (pi *bucketIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var namespace datastore.Namespace
	var objects []datastore.Object

	defer conn.Sender().Close()

	var numProduced int64 = 0
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {

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
						if object.IsBucket {
							_, err = namespace.BucketByName(object.Id)
							if err != nil {
								continue
							}
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
					}
				}
			}
		}
	}
}
