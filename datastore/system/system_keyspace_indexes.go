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
	"encoding/json"
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type indexKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *indexKeyspace) Release() {
}

func (b *indexKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *indexKeyspace) Id() string {
	return b.Name()
}

func (b *indexKeyspace) Name() string {
	return b.name
}

func handleKeyspace(keyspace datastore.Keyspace, warnF func(err errors.Error), excludeResults bool, handleF func(id string)) errors.Error {
	indexers, excp := keyspace.Indexers()
	if excp == nil {
		for _, indexer := range indexers {
			err := indexer.Refresh()
			if err == nil {

				indexIds, err := indexer.IndexIds()
				if err == nil {
					if excludeResults {
						if len(indexIds) > 0 {
							warnF(errors.NewSystemFilteredRowsWarning("system:indexes"))
						}
					} else {
						for _, indexId := range indexIds {
							handleF(indexId)
						}
					}
				}
			} else {
				warnF(errors.NewSystemDatastoreError(err, ""))
			}
		}
	}
	return excp
}

func (b *indexKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var scope datastore.Scope
	var keyspace datastore.Keyspace
	var objects []datastore.Object

	count := int64(0)
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {

	loop:
		for _, namespaceId := range namespaceIds {
			namespace, excp = b.namespace.store.actualStore.NamespaceById(namespaceId)
			if excp != nil {
				break loop
			}
			objects, excp = namespace.Objects()
			if excp != nil {
				break loop
			}
			for _, object := range objects {
				excludeResults := !canRead(context, namespaceId, object.Id) &&
					!canListIndexes(context, namespaceId, object.Id)

				if object.IsKeyspace {
					keyspace, excp = namespace.KeyspaceById(object.Id)
					if excp == nil {
						excp = handleKeyspace(keyspace, func(err errors.Error) {
							context.Warning(err)
						}, excludeResults, func(id string) {
							count++
						})
					}
					if excp != nil {
						break loop
					}
				}
				if object.IsBucket {
					bucket, excp = namespace.BucketById(object.Id)
					if excp == nil {
						break loop
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, excp = bucket.ScopeById(scopeId)
						if scope != nil {
							keyspaceIds, _ := scope.KeyspaceIds()
							for _, keyspaceId := range keyspaceIds {
								keyspace, excp = scope.KeyspaceById(keyspaceId)
								if excp == nil {

									// TODO
									// excludeResults := !canRead(context, namespaceId, keyspaceId) &&
									//		     !canListIndexes(context, namespaceId, keyspaceId)
									excp = handleKeyspace(keyspace, func(err errors.Error) {
										context.Warning(err)
									}, false /* excludeResults */, func(id string) {
										count++
									})
								}
								if excp != nil {
									break loop
								}
							}
						}
					}
				}
			}
		}
	}
	if excp == nil {
		return count, nil
	}
	return 0, errors.NewSystemDatastoreError(excp, "")
}

func (b *indexKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *indexKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *indexKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func splitIndexId(id string) (errors.Error, []string) {
	ids := strings.SplitN(id, "/", 5)
	if len(ids) != 3 && len(ids) != 5 {
		return errors.NewSystemMalformedKeyError(id, "system:indexes"), nil
	}
	return nil, ids
}

func (b *indexKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	for _, key := range keys {
		err, elems := splitIndexId(key)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(elems) == 3 {
			if !canRead(context, elems[0], elems[1]) &&
				!canListIndexes(context, elems[0], elems[1]) {
				context.Warning(errors.NewSystemFilteredRowsWarning("system:indexes"))
				continue
			}
			err = b.fetchOne(key, keysMap, elems[0], elems[1], elems[2])
		} else {
			//                      if !canAccessAll && !canRead(context, elems[0], elems[1]) {
			//                              context.Warning(errors.NewSystemFilteredRowsWarning("system:keyspaces"))
			//                              continue
			//                      }
			err = b.fetchOneCollection(key, keysMap, elems[0], elems[1], elems[2], elems[3], elems[4])
		}

		if err != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, err)
			continue
		}
	}

	return
}

func (b *indexKeyspace) fetchOne(key string, keysMap map[string]value.AnnotatedValue,
	namespaceId string, keyspaceId string, indexId string) errors.Error {

	actualStore := b.namespace.store.actualStore
	namespace, err := actualStore.NamespaceById(namespaceId)
	if err != nil {
		return err
	}

	keyspace, err := namespace.KeyspaceById(keyspaceId)
	if err != nil {
		return err
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		logging.Infof("Indexer returned error %v", err)
		return err
	}

	for _, indexer := range indexers {
		index, err := indexer.IndexById(indexId)
		if err != nil {
			continue
		}

		state, msg, err := index.State()
		if err != nil {
			return err
		}
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":           index.Id(),
			"name":         index.Name(),
			"keyspace_id":  keyspace.Id(),
			"namespace_id": namespace.Id(),
			"datastore_id": actualStore.URL(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index)),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.SetAttachment("meta", map[string]interface{}{
			"id":       key,
			"keyspace": b.fullName,
		})
		doc.SetId(key)

		partition := indexPartitionToString(index)
		if partition != "" {
			doc.SetField("partition", partition)
		}

		if msg != "" {
			doc.SetField("message", msg)
		}

		cond := index.Condition()
		if cond != nil {
			doc.SetField("condition", cond.String())
		}

		if index.IsPrimary() {
			doc.SetField("is_primary", true)
		}

		keysMap[key] = doc
	}

	return nil
}

func (b *indexKeyspace) fetchOneCollection(key string, keysMap map[string]value.AnnotatedValue,
	namespaceId string, bucketId string, scopeId string, keyspaceId string, indexId string) errors.Error {

	actualStore := b.namespace.store.actualStore
	namespace, err := actualStore.NamespaceById(namespaceId)
	if err != nil {
		return err
	}
	bucket, err := namespace.BucketById(bucketId)
	if err != nil {
		return err
	}
	scope, err := bucket.ScopeById(scopeId)
	if err != nil {
		return err
	}
	keyspace, err := scope.KeyspaceById(keyspaceId)
	if err != nil {
		return err
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		logging.Infof("Indexer returned error %v", err)
		return err
	}

	for _, indexer := range indexers {
		index, err := indexer.IndexById(indexId)
		if err != nil {
			continue
		}

		state, msg, err := index.State()
		if err != nil {
			return err
		}
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":           index.Id(),
			"name":         index.Name(),
			"keyspace_id":  keyspace.Id(),
			"scope_id":     scope.Id(),
			"bucket_id":    bucket.Id(),
			"namespace_id": namespace.Id(),
			"datastore_id": actualStore.URL(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index)),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.SetAttachment("meta", map[string]interface{}{
			"id":       key,
			"keyspace": b.fullName,
		})
		doc.SetId(key)

		partition := indexPartitionToString(index)
		if partition != "" {
			doc.SetField("partition", partition)
		}

		if msg != "" {
			doc.SetField("message", msg)
		}

		cond := index.Condition()
		if cond != nil {
			doc.SetField("condition", cond.String())
		}

		if index.IsPrimary() {
			doc.SetField("is_primary", true)
		}

		keysMap[key] = doc
	}

	return nil
}

func indexKeyToIndexKeyStringArray(index datastore.Index) (rv []string) {
	if index2, ok2 := index.(datastore.Index2); ok2 {
		keys := index2.RangeKey2()
		rv = make([]string, len(keys))
		for i, kp := range keys {
			s := expression.NewStringer().Visit(kp.Expr)
			if i == 0 && kp.HasAttribute(datastore.IK_MISSING) {
				s += " MISSING"
			}
			if kp.HasAttribute(datastore.IK_DESC) {
				s += " DESC"
			}
			rv[i] = s
		}

	} else {
		rv = make([]string, len(index.RangeKey()))
		for i, kp := range index.RangeKey() {
			rv[i] = expression.NewStringer().Visit(kp)
		}
	}
	return
}

func indexPartitionToString(index datastore.Index) (rv string) {
	index3, ok3 := index.(datastore.Index3)
	if !ok3 {
		return
	}
	partition, _ := index3.PartitionKeys()
	if partition == nil || partition.Strategy == datastore.NO_PARTITION {
		return
	}

	rv = string(partition.Strategy) + "("
	for i, expr := range partition.Exprs {
		if i > 0 {
			rv += ","
		}
		rv += expression.NewStringer().Visit(expr)
	}
	rv += ")"
	return
}

func datastoreObjectToJSONSafe(catobj interface{}) interface{} {
	var rv interface{}
	bytes, err := json.Marshal(catobj)
	if err == nil {
		json.Unmarshal(bytes, &rv)
	}
	return rv
}

func newIndexesKeyspace(p *namespace) (*indexKeyspace, errors.Error) {
	b := new(indexKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_INDEXES)

	primary := &indexIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
}

func (b *indexKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *indexKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
}

func (b *indexKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "Not yet implemented.")
}

type indexIndex struct {
	indexBase
	name     string
	keyspace *indexKeyspace
}

func (pi *indexIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *indexIndex) Id() string {
	return pi.Name()
}

func (pi *indexIndex) Name() string {
	return pi.name
}

func (pi *indexIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *indexIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *indexIndex) Condition() expression.Expression {
	return nil
}

func (pi *indexIndex) IsPrimary() bool {
	return true
}

func (pi *indexIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *indexIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *indexIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *indexIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.ScanEntries(requestId, limit, cons, vector, conn)
}

// TODO
// Do the presented credentials authorize the user to read the namespace/keyspace bucket?
func canRead(context datastore.QueryContext, namespace string, keyspace string) bool {
	privs := auth.NewPrivileges()
	privs.Add(namespace+":"+keyspace, auth.PRIV_QUERY_SELECT)
	_, err := datastore.GetDatastore().Authorize(privs, context.Credentials(), context.OriginalHttpRequest())
	res := err == nil
	return res
}

// TODO
// Do the presented credentials authorize the user to list indexes of the namespace/keyspace bucket?
func canListIndexes(context datastore.QueryContext, namespace string, keyspace string) bool {
	privs := auth.NewPrivileges()
	privs.Add(namespace+":"+keyspace, auth.PRIV_QUERY_LIST_INDEX)
	_, err := datastore.GetDatastore().Authorize(privs, context.Credentials(), context.OriginalHttpRequest())
	res := err == nil
	return res
}

func (pi *indexIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	actualStore := pi.keyspace.namespace.store.actualStore
	namespaceIds, err := actualStore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := actualStore.NamespaceById(namespaceId)
			if err != nil {
				continue
			}
			objects, err := namespace.Objects()
			if err != nil {
				continue
			}
		loop:
			for _, object := range objects {
				if object.IsKeyspace {
					keyspace, excp := namespace.KeyspaceById(object.Id)
					if excp == nil {
						keys := make(map[string]bool, 64)
						excp = handleKeyspace(keyspace, func(err errors.Error) {
							conn.Warning(err)
						}, false, func(id string) {
							key := makeId(namespaceId, object.Id, id)

							// avoid duplicates
							if !keys[key] {
								entry := datastore.IndexEntry{PrimaryKey: key}
								if !sendSystemKey(conn, &entry) {
									return
								}
								keys[key] = true
							}
						})
						keys = nil
					}
					if excp != nil {
						continue loop
					}
				}
				if object.IsBucket {
					bucket, excp := namespace.BucketById(object.Id)
					if excp == nil {
						continue loop
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, _ := bucket.ScopeById(scopeId)
						if scope != nil {
							keyspaceIds, _ := scope.KeyspaceIds()
							for _, keyspaceId := range keyspaceIds {
								keyspace, excp := scope.KeyspaceById(keyspaceId)
								if excp == nil {
									keys := make(map[string]bool, 64)
									excp = handleKeyspace(keyspace, func(error errors.Error) {
										conn.Warning(err)
									}, false, func(id string) {
										key := makeId(namespaceId, object.Id, scopeId, id)

										// avoid duplicates
										if !keys[key] {
											entry := datastore.IndexEntry{PrimaryKey: key}
											if !sendSystemKey(conn, &entry) {
												return
											}
											keys[key] = true
										}
									})
									keys = nil
								}
								if excp != nil {
									continue loop
								}
							}
						}
					}
				}
			}
		}
	}
}
