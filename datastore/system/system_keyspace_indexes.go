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
	"fmt"
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
	namespace *namespace
	name      string
	indexer   datastore.Indexer
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

func (b *indexKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	count := int64(0)
	namespaceIds, excp := b.namespace.store.actualStore.NamespaceIds()
	if excp == nil {
		for _, namespaceId := range namespaceIds {
			namespace, excp := b.namespace.store.actualStore.NamespaceById(namespaceId)
			if excp == nil {
				keyspaceIds, excp := namespace.KeyspaceIds()
				if excp == nil {
					for _, keyspaceId := range keyspaceIds {
						if !canRead(context, namespaceId, keyspaceId) &&
							!canListIndexes(context, namespaceId, keyspaceId) {
							continue
						}
						keyspace, excp := namespace.KeyspaceById(keyspaceId)
						if excp == nil {
							indexers, excp := keyspace.Indexers()
							if excp == nil {
								for _, indexer := range indexers {
									excp = indexer.Refresh()
									if excp != nil {
										return 0, errors.NewSystemDatastoreError(excp, "")
									}

									indexIds, excp := indexer.IndexIds()
									if excp == nil {
										count += int64(len(indexIds))
									} else {
										return 0, errors.NewSystemDatastoreError(excp, "")
									}
								}
							} else {
								return 0, errors.NewSystemDatastoreError(excp, "")
							}
						} else {
							return 0, errors.NewSystemDatastoreError(excp, "")
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

func (b *indexKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *indexKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *indexKeyspace) Fetch(keys []string, context datastore.QueryContext) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys)*2)

	for _, key := range keys {
		ids := strings.SplitN(key, "/", 3)
		namespaceId := ids[0]
		keyspaceId := ids[1]
		indexId := ids[2]
		if !canRead(context, namespaceId, keyspaceId) &&
			!canListIndexes(context, namespaceId, keyspaceId) {
			continue
		}
		pairs, err := b.fetchOne(key, namespaceId, keyspaceId, indexId)
		if err != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, err)
			continue
		}

		rv = append(rv, pairs...)
	}

	return rv, errs
}

func (b *indexKeyspace) fetchOne(key string, namespaceId string, keyspaceId string, indexId string) ([]value.AnnotatedPair, errors.Error) {
	rv := make([]value.AnnotatedPair, 0, 2)

	actualStore := b.namespace.store.actualStore
	namespace, err := actualStore.NamespaceById(namespaceId)
	if err != nil {
		return nil, err
	}

	keyspace, err := namespace.KeyspaceById(keyspaceId)
	if err != nil {
		return nil, err
	}

	indexers, err := keyspace.Indexers()
	if err != nil {
		logging.Infof("Indexer returned error %v", err)
		return nil, err
	}

	for _, indexer := range indexers {
		index, err := indexer.IndexById(indexId)
		if err != nil {
			continue
		}

		state, msg, err := index.State()
		if err != nil {
			return nil, err
		}
		doc := value.NewAnnotatedValue(map[string]interface{}{
			"id":           index.Id(),
			"name":         index.Name(),
			"keyspace_id":  keyspace.Id(),
			"namespace_id": namespace.Id(),
			"datastore_id": actualStore.URL(),
			"index_key":    datastoreObjectToJSONSafe(indexKeyToIndexKeyStringArray(index.RangeKey())),
			"using":        datastoreObjectToJSONSafe(index.Type()),
			"state":        string(state),
		})

		doc.SetAttachment("meta", map[string]interface{}{
			"id": key,
		})

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

		rv = append(rv, value.AnnotatedPair{key, doc})
	}

	return rv, nil
}

func indexKeyToIndexKeyStringArray(key expression.Expressions) []string {
	rv := make([]string, len(key))
	for i, kp := range key {
		rv[i] = expression.NewStringer().Visit(kp)
	}
	return rv
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
	b.namespace = p
	b.name = KEYSPACE_NAME_INDEXES

	primary := &indexIndex{name: "#primary", keyspace: b}
	b.indexer = newSystemIndexer(b, primary)

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
func canRead(context datastore.QueryContext, namespace string, keyspace string) bool {
	privs := auth.NewPrivileges()
	privs.Add(namespace+":"+keyspace, auth.PRIV_QUERY_SELECT)
	_, err := datastore.GetDatastore().Authorize(privs, context.Credentials(), context.OriginalHttpRequest())
	res := err == nil
	return res
}

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
	defer close(conn.EntryChannel())

	// eliminate duplicate keys
	keys := make(map[string]string, 64)

	actualStore := pi.keyspace.namespace.store.actualStore
	namespaceIds, err := actualStore.NamespaceIds()
	if err == nil {
		for _, namespaceId := range namespaceIds {
			namespace, err := actualStore.NamespaceById(namespaceId)
			if err == nil {
				keyspaceIds, err := namespace.KeyspaceIds()
				if err == nil {
					for _, keyspaceId := range keyspaceIds {
						keyspace, err := namespace.KeyspaceById(keyspaceId)
						if err == nil {
							indexers, err := keyspace.Indexers()
							if err == nil {
								for _, indexer := range indexers {
									err = indexer.Refresh()
									if err != nil {
										logging.Errorf("Refreshing indexes failed %v", err)

										// MB-23555, don't throw errors, or the scan will be terminated
										conn.Warning(errors.NewSystemDatastoreError(err, ""))
										// don't return here but continue processing, because other keyspaces may still be responsive. MB-15834
										continue
									}

									indexIds, err := indexer.IndexIds()
									if err == nil {
										for _, indexId := range indexIds {
											key := fmt.Sprintf("%s/%s/%s", namespaceId, keyspaceId, indexId)
											keys[key] = key
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

	for k, _ := range keys {
		entry := datastore.IndexEntry{PrimaryKey: k}
		conn.EntryChannel() <- &entry
	}
}
