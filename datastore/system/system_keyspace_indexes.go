//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"encoding/json"
	"strings"

	"github.com/couchbase/query/algebra"
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
	skipSystem bool
	store      datastore.Datastore
	indexer    datastore.Indexer
}

func (b *indexKeyspace) Release(close bool) {
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

func handleKeyspace(keyspace datastore.Keyspace, warnF func(err errors.Error), includeResults bool, handleF func(id string), includeSeqScan bool) errors.Error {
	indexers, excp := keyspace.Indexers()
	if excp == nil {
		for _, indexer := range indexers {
			if !includeSeqScan && indexer.Name() == datastore.SEQ_SCAN {
				continue
			}
			err := indexer.Refresh()
			if err == nil {

				indexIds, err := indexer.IndexIds()
				if err == nil {
					if includeResults {
						for _, indexId := range indexIds {
							handleF(indexId)
						}
					} else {
						if len(indexIds) > 0 {
							warnF(errors.NewSystemFilteredRowsWarning("system:indexes"))
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
	namespaceIds, excp := b.store.NamespaceIds()
	if excp == nil {
		canAccessAll := canAccessSystemTables(context)

		includeSeqScan := b.Name() == KEYSPACE_NAME_ALL_INDEXES
	loop:
		for _, namespaceId := range namespaceIds {
			namespace, excp = b.store.NamespaceById(namespaceId)
			if excp != nil {
				break loop
			}
			objects, excp = namespace.Objects(true)
			if excp != nil {
				break loop
			}
			for _, object := range objects {
				includeDefaultKeyspace := canAccessAll || (canRead(context, namespaceId, object.Id) &&
					canListIndexes(context, namespaceId, object.Id))

				if object.IsKeyspace {
					keyspace, excp = namespace.KeyspaceById(object.Id)
					if excp == nil {
						excp = handleKeyspace(keyspace, func(err errors.Error) {
							context.Warning(err)
						}, includeDefaultKeyspace, func(id string) {
							count++
						}, includeSeqScan)
					}
					if excp != nil {
						break loop
					}
				}
				if object.IsBucket {
					bucket, excp = namespace.BucketById(object.Id)
					if excp != nil {
						break loop
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, excp = bucket.ScopeById(scopeId)
						if scope != nil {
							includeScope := includeDefaultKeyspace || (canRead(context, namespaceId, object.Id, scopeId) &&
								canListIndexes(context, namespaceId, object.Id, scopeId))
							keyspaceIds, _ := scope.KeyspaceIds()
							for _, keyspaceId := range keyspaceIds {
								keyspace, excp = scope.KeyspaceById(keyspaceId)
								if excp == nil {

									includeResults := includeScope || (canRead(context, namespaceId, object.Id, scopeId, keyspaceId) &&
										canListIndexes(context, namespaceId, object.Id, scopeId, keyspaceId))
									excp = handleKeyspace(keyspace, func(err errors.Error) {
										context.Warning(err)
									}, includeResults, func(id string) {
										count++
									}, includeSeqScan)
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
	context datastore.QueryContext, subPaths []string) (errs errors.Errors) {

	for _, key := range keys {
		err, elems := splitIndexId(key)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(elems) == 3 {
			err = b.fetchOne(key, keysMap, elems[0], elems[1], elems[2])
		} else {
			err = b.fetchOneCollection(key, keysMap, elems[0], elems[1], elems[2], elems[3], elems[4])
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return
}

func (b *indexKeyspace) fetchOne(key string, keysMap map[string]value.AnnotatedValue,
	namespaceId string, keyspaceId string, indexId string) errors.Error {

	namespace, err := b.store.NamespaceById(namespaceId)
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
			"datastore_id": b.store.URL(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index)),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.NewMeta()["keyspace"] = b.fullName
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

		if ixm, ok := index.(interface{ IndexMetadata() map[string]interface{} }); ok {
			doc.SetField("metadata", datastoreObjectToJSONSafe(ixm.IndexMetadata()))
		}

		keysMap[key] = doc
	}

	return nil
}

func (b *indexKeyspace) fetchOneCollection(key string, keysMap map[string]value.AnnotatedValue,
	namespaceId string, bucketId string, scopeId string, keyspaceId string, indexId string) errors.Error {

	// this should never happen, but if it does, we skip silently system collections
	// (not an error, they are just not part of the result set)
	if b.skipSystem && keyspaceId[0] == '_' {
		return nil
	}

	namespace, err := b.store.NamespaceById(namespaceId)
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
			"datastore_id": b.store.URL(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index)),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.NewMeta()["keyspace"] = b.fullName
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

func newIndexesKeyspace(p *namespace, store datastore.Datastore, name string, skipSystem bool) (*indexKeyspace, errors.Error) {
	b := new(indexKeyspace)
	b.store = store
	b.skipSystem = skipSystem
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &indexIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	return b, nil
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

// Do the presented credentials authorize the user to read the namespace/keyspace bucket?
func canRead(context datastore.QueryContext, elems ...string) bool {
	privs := auth.NewPrivileges()
	privs.Add(algebra.NewPathFromElements(elems).FullName(), auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)
	err := datastore.GetDatastore().Authorize(privs, context.Credentials())
	res := err == nil
	return res
}

// Do the presented credentials authorize the user to list indexes of the namespace/keyspace bucket?
func canListIndexes(context datastore.QueryContext, elems ...string) bool {
	privs := auth.NewPrivileges()
	privs.Add(algebra.NewPathFromElements(elems).FullName(), auth.PRIV_QUERY_LIST_INDEX, auth.PRIV_PROPS_NONE)
	err := datastore.GetDatastore().Authorize(privs, context.Credentials())
	res := err == nil
	return res
}

func (pi *indexIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	includeSeqScan := pi.keyspace.Name() == KEYSPACE_NAME_ALL_INDEXES
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {
		canAccessAll := canAccessSystemTables(conn.QueryContext())
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.store.NamespaceById(namespaceId)
			if err != nil {
				continue
			}
			objects, err := namespace.Objects(true)
			if err != nil {
				continue
			}
		loop:
			for _, object := range objects {
				includeDefaultKeyspace := canAccessAll || (canRead(conn.QueryContext(), namespaceId, object.Id) &&
					canListIndexes(conn.QueryContext(), namespaceId, object.Id))
				if object.IsKeyspace {
					keyspace, excp := namespace.KeyspaceById(object.Id)
					if excp == nil {
						keys := make(map[string]bool, 64)
						excp = handleKeyspace(keyspace, func(err errors.Error) {
							conn.Warning(err)
						}, includeDefaultKeyspace, func(id string) {
							key := makeId(namespaceId, object.Id, id)

							// avoid duplicates
							if !keys[key] {
								entry := datastore.IndexEntry{PrimaryKey: key}
								if !sendSystemKey(conn, &entry) {
									return
								}
								keys[key] = true
							}
						}, includeSeqScan)
						keys = nil
					}
					if excp != nil {
						continue loop
					}
				}
				if object.IsBucket {
					bucket, excp := namespace.BucketById(object.Id)
					if excp != nil {
						continue loop
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, _ := bucket.ScopeById(scopeId)
						if scope != nil {
							includeScope := includeDefaultKeyspace || (canRead(conn.QueryContext(), namespaceId, object.Id, scopeId) &&
								canListIndexes(conn.QueryContext(), namespaceId, object.Id, scopeId))
							keyspaceIds, _ := scope.KeyspaceIds()
							for _, keyspaceId := range keyspaceIds {
								if pi.keyspace.skipSystem && keyspaceId[0] == '_' {
									continue
								}

								keyspace, excp := scope.KeyspaceById(keyspaceId)
								if keyspace != nil {
									keys := make(map[string]bool, 64)
									includeResults := includeScope || (canRead(conn.QueryContext(), namespaceId, object.Id, scopeId, keyspaceId) &&
										canListIndexes(conn.QueryContext(), namespaceId, object.Id, scopeId, keyspaceId))
									excp = handleKeyspace(keyspace, func(err errors.Error) {
										conn.Warning(err)
									}, includeResults, func(id string) {
										key := makeId(namespaceId, object.Id, scopeId, keyspaceId, id)

										// avoid duplicates
										if !keys[key] {
											entry := datastore.IndexEntry{PrimaryKey: key}
											if !sendSystemKey(conn, &entry) {
												return
											}
											keys[key] = true
										}
									}, includeSeqScan)
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
