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
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/value"
)

type transactionsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *transactionsKeyspace) Release(close bool) {
}

func (b *transactionsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *transactionsKeyspace) Id() string {
	return b.Name()
}

func (b *transactionsKeyspace) Name() string {
	return b.name
}

func (b *transactionsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "transactions", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	return int64(transactions.CountTransContext() + count), nil
}

func (b *transactionsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *transactionsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *transactionsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *transactionsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"transactions", "POST",
				func(doc map[string]interface{}) {
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.NewMeta()["keyspace"] = b.fullName
					remoteValue.SetField("node", node)
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					context.Warning(warn)
				}, distributed.NO_CREDS, "")
		} else {

			// local entry
			transactions.TransactionEntryDo(localKey, func(d interface{}) {
				itemMap := map[string]interface{}{}
				entry := d.(*transactions.TranContext)
				entry.Content(itemMap)
				if node != "" {
					itemMap["node"] = node
				}
				item := value.NewAnnotatedValue(itemMap)
				item.NewMeta()["keyspace"] = b.fullName
				item.SetId(key)
				keysMap[key] = item
			})
		}
	}
	return
}

func (b *transactionsKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *transactionsKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *transactionsKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *transactionsKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	var err errors.Error

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"transactions", "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				distributed.NO_CREDS, "")

			// local entry
		} else {
			err = transactions.DeleteTransContext(localKey, true)
		}
		if err != nil {
			deleted := make([]value.Pair, i)
			if i > 0 {
				copy(deleted, deletes[0:i-1])
			}
			return deleted, err
		}
	}
	return deletes, nil
}

func newTransactionsKeyspace(p *namespace) (*transactionsKeyspace, errors.Error) {
	b := new(transactionsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_TRANSACTIONS)

	primary := &transactionsIndex{
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
		nodes := &transactionsIndex{
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

type transactionsIndex struct {
	indexBase
	name     string
	keyspace *transactionsKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *transactionsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *transactionsIndex) Id() string {
	return pi.Name()
}

func (pi *transactionsIndex) Name() string {
	return pi.name
}

func (pi *transactionsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *transactionsIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *transactionsIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *transactionsIndex) Condition() expression.Expression {
	return nil
}

func (pi *transactionsIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *transactionsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *transactionsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *transactionsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *transactionsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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
				transactions.TransactionEntriesForeach(func(name string, d interface{}) bool {
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
				distributed.RemoteAccess().GetRemoteKeys(nodes, "transactions", func(id string) bool {
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
						transactions.TransactionEntriesForeach(func(name string, d interface{}) bool {
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
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "transactions", func(id string) bool {
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

func (pi *transactionsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	transactions.TransactionEntriesForeach(func(name string, d interface{}) bool {
		entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
		return true
	}, func() bool {
		return sendSystemKey(conn, entry)
	})
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "transactions", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
