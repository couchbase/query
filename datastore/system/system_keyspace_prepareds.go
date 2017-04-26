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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type preparedsKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *preparedsKeyspace) Release() {
}

func (b *preparedsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *preparedsKeyspace) Id() string {
	return b.Name()
}

func (b *preparedsKeyspace) Name() string {
	return b.name
}

func (b *preparedsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "prepareds", func(id string) {
		count++
	}, func(warn errors.Error) {
		context.Warning(warn)
	})
	return int64(plan.CountPrepareds() + count), nil
}

func (b *preparedsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *preparedsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *preparedsKeyspace) Fetch(keys []string, context datastore.QueryContext) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys))

	creds, authToken := credsFromContext(context)

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"prepareds", "POST",
				func(doc map[string]interface{}) {

					plan := doc["plan"]
					delete(doc, "plan")
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					remoteValue.SetAttachment("meta", map[string]interface{}{
						"id":   key,
						"plan": plan,
					})
					rv = append(rv, value.AnnotatedPair{
						Name:  key,
						Value: remoteValue,
					})
				},
				func(warn errors.Error) {
					context.Warning(warn)
				}, creds, authToken)
		} else {

			// local entry
			plan.PreparedDo(localKey, func(entry *plan.CacheEntry) {
				itemMap := map[string]interface{}{
					"name":         localKey,
					"uses":         entry.Uses,
					"statement":    entry.Prepared.Text(),
					"encoded_plan": entry.Prepared.EncodedPlan(),
				}
				if node != "" {
					itemMap["node"] = node
				}
				if entry.Uses > 0 {
					itemMap["lastUse"] = entry.LastUse.String()
					itemMap["avgElapsedTime"] = (time.Duration(entry.RequestTime) /
						time.Duration(entry.Uses)).String()
					itemMap["avgServiceTime"] = (time.Duration(entry.ServiceTime) /
						time.Duration(entry.Uses)).String()
					itemMap["minElapsedTime"] = time.Duration(entry.MinRequestTime).String()
					itemMap["minServiceTime"] = time.Duration(entry.MinServiceTime).String()
					itemMap["maxElapsedTime"] = time.Duration(entry.MaxRequestTime).String()
					itemMap["maxServiceTime"] = time.Duration(entry.MaxServiceTime).String()
				}
				item := value.NewAnnotatedValue(itemMap)
				bytes, _ := json.Marshal(entry.Prepared.Operator)
				item.SetAttachment("meta", map[string]interface{}{
					"id":   key,
					"plan": bytes,
				})
				rv = append(rv, value.AnnotatedPair{
					Name:  key,
					Value: item,
				})
			})
		}
	}
	return rv, errs
}

func (b *preparedsKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *preparedsKeyspace) Delete(deletes []string, context datastore.QueryContext) ([]string, errors.Error) {
	var err errors.Error

	creds, authToken := credsFromContext(context)

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, name := range deletes {
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"prepareds", "DELETE", nil,
				func(warn errors.Error) {
					context.Warning(warn)
				},
				creds, authToken)

			// local entry
		} else {
			err = plan.DeletePrepared(localKey)
		}
		if err != nil {
			deleted := make([]string, i)
			if i > 0 {
				copy(deleted, deletes[0:i-1])
			}
			return deleted, err
		}
	}
	return deletes, nil
}

func newPreparedsKeyspace(p *namespace) (*preparedsKeyspace, errors.Error) {
	b := new(preparedsKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_PREPAREDS

	primary := &preparedsIndex{
		name:     "#primary",
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)

	// add a secondary index on `node`
	expr, err := parser.Parse(`node`)

	if err == nil {
		key := expression.Expressions{expr}
		nodes := &preparedsIndex{
			name:     "#nodes",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type preparedsIndex struct {
	name     string
	keyspace *preparedsKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *preparedsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *preparedsIndex) Id() string {
	return pi.Name()
}

func (pi *preparedsIndex) Name() string {
	return pi.name
}

func (pi *preparedsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *preparedsIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *preparedsIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *preparedsIndex) Condition() expression.Expression {
	return nil
}

func (pi *preparedsIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *preparedsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *preparedsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *preparedsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *preparedsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer close(conn.EntryChannel())

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		if spanEvaluator.isEquals() {

			// now that the node name can change in flight, use a consistent one across the scan
			whoAmI := distributed.RemoteAccess().WhoAmI()
			if spanEvaluator.key() == whoAmI {
				names := plan.NamePrepareds()

				for _, name := range names {
					indexEntry := datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
						EntryKey:   value.Values{value.NewValue(whoAmI)},
					}
					conn.EntryChannel() <- &indexEntry
				}
			} else {
				nodes := []string{spanEvaluator.key()}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "prepareds", func(id string) {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					conn.EntryChannel() <- &indexEntry
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
						names := plan.NamePrepareds()

						for _, name := range names {
							indexEntry := datastore.IndexEntry{
								PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
								EntryKey:   value.Values{value.NewValue(whoAmI)},
							}
							conn.EntryChannel() <- &indexEntry
						}
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "prepareds", func(id string) {
					n, _ := distributed.RemoteAccess().SplitKey(id)
					indexEntry := datastore.IndexEntry{
						PrimaryKey: id,
						EntryKey:   value.Values{value.NewValue(n)},
					}
					conn.EntryChannel() <- &indexEntry
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		}
	}
}

func (pi *preparedsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())
	names := plan.NamePrepareds()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, name := range names {
		entry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
		conn.EntryChannel() <- &entry
	}
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "prepareds", func(id string) {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &indexEntry
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
