//  Copyright (c) 2019 Couchbase, Inc.
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
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type tasksCacheKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *tasksCacheKeyspace) Release(close bool) {
}

func (b *tasksCacheKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *tasksCacheKeyspace) Id() string {
	return b.Name()
}

func (b *tasksCacheKeyspace) Name() string {
	return b.name
}

func (b *tasksCacheKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "tasks_cache", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	return int64(scheduler.CountTasks() + count), nil
}

func (b *tasksCacheKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *tasksCacheKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *tasksCacheKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *tasksCacheKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs []errors.Error) {

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"tasks_cache", "POST",
				func(doc map[string]interface{}) {

					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					remoteValue.NewMeta()["keyspace"] = b.fullName
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					context.Warning(warn)
				}, distributed.NO_CREDS, "")
		} else {

			// local entry
			scheduler.TaskDo(localKey, func(entry *scheduler.TaskEntry) {
				itemMap := map[string]interface{}{
					"class":      entry.Class,
					"subClass":   entry.SubClass,
					"name":       entry.Name,
					"id":         entry.Id,
					"state":      entry.State,
					"submitTime": entry.PostTime.String(),
					"delay":      entry.Delay.String(),
				}
				if entry.Results != nil {
					itemMap["results"] = entry.Results
				}
				if entry.Errors != nil {
					itemMap["errors"] = entry.Errors
				}
				if !entry.StartTime.IsZero() {
					itemMap["startTime"] = entry.StartTime.String()
				}
				if !entry.EndTime.IsZero() {
					itemMap["stopTime"] = entry.EndTime.String()
				}
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

func (b *tasksCacheKeyspace) Insert(inserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *tasksCacheKeyspace) Update(updates []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *tasksCacheKeyspace) Upsert(upserts []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *tasksCacheKeyspace) Delete(deletes []value.Pair, context datastore.QueryContext) ([]value.Pair, errors.Error) {

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"tasks_cache", "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				distributed.NO_CREDS, "")

		} else {
			// local entry
			scheduler.DeleteTask(localKey)
		}
	}
	return deletes, nil
}

func newTasksCacheKeyspace(p *namespace) (*tasksCacheKeyspace, errors.Error) {
	b := new(tasksCacheKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_TASKS_CACHE)

	primary := &tasksCacheIndex{
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
		nodes := &tasksCacheIndex{
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

type tasksCacheIndex struct {
	indexBase
	name     string
	keyspace *tasksCacheKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *tasksCacheIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *tasksCacheIndex) Id() string {
	return pi.Name()
}

func (pi *tasksCacheIndex) Name() string {
	return pi.name
}

func (pi *tasksCacheIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *tasksCacheIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *tasksCacheIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *tasksCacheIndex) Condition() expression.Expression {
	return nil
}

func (pi *tasksCacheIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *tasksCacheIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *tasksCacheIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *tasksCacheIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *tasksCacheIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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
				scheduler.TasksForeach(func(name string, task *scheduler.TaskEntry) bool {
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
				distributed.RemoteAccess().GetRemoteKeys(nodes, "tasks_cache", func(id string) bool {
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

						scheduler.TasksForeach(func(name string, task *scheduler.TaskEntry) bool {
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
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "tasks_cache", func(id string) bool {
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

func (pi *tasksCacheIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	scheduler.TasksForeach(func(name string, task *scheduler.TaskEntry) bool {
		entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
		return true
	}, func() bool {
		return sendSystemKey(conn, entry)
	})
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "tasks_cache", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
