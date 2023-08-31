//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
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
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			context.Warning(warn)
		}
	}, distributed.NO_CREDS, "")
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
	context datastore.QueryContext, subPaths []string, projection []string) (errs errors.Errors) {

	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey, "tasks_cache", "POST",
				func(doc map[string]interface{}) {

					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					remoteValue.NewMeta()["keyspace"] = b.fullName
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				}, distributed.NO_CREDS, "", formData)
		} else {

			// local entry
			scheduler.TaskDo(localKey, func(entry *scheduler.TaskEntry) {
				itemMap := map[string]interface{}{
					"class":        entry.Class,
					"subClass":     entry.SubClass,
					"name":         entry.Name,
					"id":           entry.Id,
					"state":        entry.State,
					"queryContext": entry.QueryContext,
					"submitTime":   entry.PostTime.Format(util.DEFAULT_FORMAT),
					"delay":        entry.Delay.String(),
				}
				if entry.Results != nil {
					itemMap["results"] = entry.Results
				}
				if len(entry.Errors) > 0 {
					errors := make([]interface{}, 0, len(entry.Errors))
					for _, err := range entry.Errors {
						if err != nil {
							errors = append(errors, err.Object())
						}
					}
					itemMap["errors"] = errors
				}
				if !entry.StartTime.IsZero() {
					itemMap["startTime"] = entry.StartTime.Format(util.DEFAULT_FORMAT)
				}
				if !entry.EndTime.IsZero() {
					itemMap["stopTime"] = entry.EndTime.Format(util.DEFAULT_FORMAT)
				}
				if node != "" {
					itemMap["node"] = node
				}
				if entry.Description != "" {
					itemMap["description"] = entry.Description
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

func (b *tasksCacheKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (int, value.Pairs, errors.Errors) {

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey,
				"tasks_cache", "DELETE", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				distributed.NO_CREDS, "", nil)

		} else {
			// local entry
			scheduler.DeleteTask(localKey)
		}
	}

	if preserveMutations {
		return len(deletes), deletes, nil
	} else {
		return len(deletes), nil, nil
	}
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

		// now that the node name can change in flight, use a consistent one across the scan
		whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())

		idx := spanEvaluator.isEquals()
		if idx >= 0 {
			if spanEvaluator.key(idx) == whoAmI {
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
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "tasks_cache", func(id string) bool {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					return sendSystemKey(conn, &indexEntry)
				}, func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, distributed.NO_CREDS, "")
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				encodedNode := encodeNodeName(node)
				if spanEvaluator.evaluate(encodedNode) {
					if encodedNode == whoAmI {

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
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						conn.Warning(warn)
					}
				}, distributed.NO_CREDS, "")
			}
		}
	}
}

func (pi *tasksCacheIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())
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
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			conn.Warning(warn)
		}
	}, distributed.NO_CREDS, "")
}
