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
	"github.com/couchbase/query/datastore"
	dictionary "github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type dictionaryCacheKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *dictionaryCacheKeyspace) Release(close bool) {
}

func (b *dictionaryCacheKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *dictionaryCacheKeyspace) Id() string {
	return b.Name()
}

func (b *dictionaryCacheKeyspace) Name() string {
	return b.name
}

func (b *dictionaryCacheKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, b.name, func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	return int64(dictionary.CountDictCacheEntries() + count), nil
}

func (b *dictionaryCacheKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *dictionaryCacheKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *dictionaryCacheKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *dictionaryCacheKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				b.name, "POST",
				func(doc map[string]interface{}) {

					distributions := doc["distributions"]
					delete(doc, "distributions")
					if distributions != nil {
						dists := distributions.(map[string]interface{})
						if len(dists) > 0 {
							distKeys := make([]interface{}, 0, len(dists))
							for n, _ := range dists {
								distKeys = append(distKeys, n)
							}
							doc["distributionKeys"] = distKeys
						}
					}
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					meta := remoteValue.NewMeta()
					meta["keyspace"] = b.fullName
					meta["distributions"] = distributions
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					context.Warning(warn)
				}, distributed.NO_CREDS, "")
		} else {

			// local entry
			dictionary.DictCacheEntryDo(localKey, func(d interface{}) {
				itemMap := map[string]interface{}{}
				entry := d.(dictionary.DictCacheEntry)
				entry.Target(itemMap)
				entry.Dictionary(itemMap)
				distributions := itemMap["distributions"]
				delete(itemMap, "distributions")
				if distributions != nil {
					dists := distributions.(map[string]interface{})
					if len(dists) > 0 {
						distKeys := make([]interface{}, 0, len(dists))
						for n, _ := range dists {
							distKeys = append(distKeys, n)
						}
						itemMap["distributionKeys"] = distKeys
					}
				}
				item := value.NewAnnotatedValue(itemMap)
				meta := item.NewMeta()
				meta["keyspace"] = b.fullName
				meta["distributions"] = distributions
				item.SetId(key)
				keysMap[key] = item
			})
		}
	}
	return
}

func (b *dictionaryCacheKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryCacheKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryCacheKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *dictionaryCacheKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				b.name, "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				distributed.NO_CREDS, "")

		} else {
			// local entry
			dictionary.DropDictCacheEntry(localKey, false)
		}
	}
	return deletes, nil
}

func newDictionaryCacheKeyspace(p *namespace, name string) (*dictionaryCacheKeyspace, errors.Error) {
	b := new(dictionaryCacheKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &dictionaryCacheIndex{
		name:     "#primary",
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `node`
	expr, err := parser.Parse(`node`)

	if err == nil {
		key := expression.Expressions{expr}
		nodes := &dictionaryCacheIndex{
			name:     "#nodes",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&nodes.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type dictionaryCacheIndex struct {
	indexBase
	name     string
	keyspace *dictionaryCacheKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *dictionaryCacheIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *dictionaryCacheIndex) Id() string {
	return pi.Name()
}

func (pi *dictionaryCacheIndex) Name() string {
	return pi.name
}

func (pi *dictionaryCacheIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *dictionaryCacheIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *dictionaryCacheIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *dictionaryCacheIndex) Condition() expression.Expression {
	return nil
}

func (pi *dictionaryCacheIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *dictionaryCacheIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *dictionaryCacheIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *dictionaryCacheIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *dictionaryCacheIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		var entry *datastore.IndexEntry
		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		if spanEvaluator.isEquals() {

			// now that the node name can change in flight, use a consistent one across the scan
			whoAmI := distributed.RemoteAccess().WhoAmI()
			if spanEvaluator.key() == whoAmI {
				dictionary.DictCacheEntriesForeach(func(name string, d interface{}) bool {
					entry = &datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
						EntryKey:   value.Values{value.NewValue(whoAmI)},
					}
					return true
				}, func() bool {
					return sendSystemKey(conn, entry)
				})
			} else {
				nodes := []string{spanEvaluator.key()}
				distributed.RemoteAccess().GetRemoteKeys(nodes, pi.keyspace.name, func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		} else {

			// now that the node name can change in flight, use a consistent one across the scan
			whoAmI := distributed.RemoteAccess().WhoAmI()
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				if spanEvaluator.evaluate(node) {
					if node == whoAmI {

						dictionary.DictCacheEntriesForeach(func(name string, d interface{}) bool {
							entry = &datastore.IndexEntry{
								PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
								EntryKey:   value.Values{value.NewValue(whoAmI)},
							}
							return true
						}, func() bool {
							return sendSystemKey(conn, entry)
						})
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, pi.keyspace.name, func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		}
	}
}

func (pi *dictionaryCacheIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	dictionary.DictCacheEntriesForeach(func(name string, d interface{}) bool {
		entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
		return true
	}, func() bool {
		return sendSystemKey(conn, entry)
	})
	distributed.RemoteAccess().GetRemoteKeys([]string{}, pi.keyspace.name, func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
