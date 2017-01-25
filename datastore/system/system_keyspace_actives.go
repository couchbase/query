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
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type activeRequestsKeyspace struct {
	namespace *namespace
	name      string
	indexer   datastore.Indexer
}

func (b *activeRequestsKeyspace) Release() {
}

func (b *activeRequestsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *activeRequestsKeyspace) Id() string {
	return b.Name()
}

func (b *activeRequestsKeyspace) Name() string {
	return b.name
}

func (b *activeRequestsKeyspace) Count() (int64, errors.Error) {
	var count int

	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) {
		count++
	}, func(warn errors.Error) {

		// FIXME Count does not handle warnings
	})
	c, err := server.ActiveRequestsCount()
	return int64(c + count), err
}

func (b *activeRequestsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *activeRequestsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *activeRequestsKeyspace) Fetch(keys []string) ([]value.AnnotatedPair, []errors.Error) {
	var errs []errors.Error
	rv := make([]value.AnnotatedPair, 0, len(keys))

	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)

		// remote entry
		if len(node) != 0 && node != distributed.RemoteAccess().WhoAmI() {
			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"active_requests", "POST",
				func(doc map[string]interface{}) {

					meta := map[string]interface{}{
						"id": key,
					}
					t, ok := doc["timings"]
					if ok {
						meta["plan"] = t
						delete(doc, "timings")
					}
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)
					remoteValue.SetAttachment("meta", meta)
					rv = append(rv, value.AnnotatedPair{
						Name:  key,
						Value: remoteValue,
					})
				},

				// FIXME Fetch() does not handle warnings
				func(warn errors.Error) {
				})
		} else {
			var item value.AnnotatedValue

			// local entry
			err := server.ActiveRequestsGet(localKey, func(request server.Request) {

				item = value.NewAnnotatedValue(map[string]interface{}{
					"requestId":       localKey,
					"requestTime":     request.RequestTime().String(),
					"elapsedTime":     time.Since(request.RequestTime()).String(),
					"executionTime":   time.Since(request.ServiceTime()).String(),
					"state":           request.State(),
					"scanConsistency": request.ScanConsistency(),
				})
				if node != "" {
					item.SetField("node", node)
				}
				cId := request.ClientID().String()
				if cId != "" {
					item.SetField("clientContextID", cId)
				}
				if request.Statement() != "" {
					item.SetField("statement", request.Statement())
				}
				p := request.Output().FmtPhaseCounts()
				if p != nil {
					item.SetField("phaseCounts", p)
				}
				p = request.Output().FmtPhaseOperators()
				if p != nil {
					item.SetField("phaseOperators", p)
				}
				if request.Prepared() != nil {
					p := request.Prepared()
					item.SetField("preparedName", p.Name())
					item.SetField("preparedText", p.Text())
				}
				prof := request.Profile()
				if prof == server.ProfUnset {
					prof = server.GetProfile()
				}
				if prof != server.ProfOff {
					item.SetField("phaseTimes", request.Output().FmtPhaseTimes())
				}

				var ctrl bool
				ctr := request.Controls()
				if ctr == value.NONE {
					ctrl = server.GetControls()
				} else {
					ctrl = (ctr == value.TRUE)
				}
				if ctrl {
					na := request.NamedArgs()
					if na != nil {
						item.SetField("namedArgs", na)
					}
					pa := request.PositionalArgs()
					if pa != nil {
						item.SetField("positionalArgs", pa)
					}
				}

				meta := map[string]interface{}{
					"id": key,
				}

				t := request.GetTimings()
				if prof == server.ProfOn && t != nil {
					bytes, _ := json.Marshal(t)
					meta["plan"] = bytes
				}

				item.SetAttachment("meta", meta)
			})
			if err != nil {
				errs = append(errs, err)
			} else if item != nil {
				rv = append(rv, value.AnnotatedPair{
					Name:  key,
					Value: item,
				})
			}
		}
	}
	return rv, errs
}

func (b *activeRequestsKeyspace) Insert(inserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Update(updates []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Upsert(upserts []value.Pair) ([]value.Pair, errors.Error) {
	// FIXME
	return nil, errors.NewSystemNotImplementedError(nil, "")
}

func (b *activeRequestsKeyspace) Delete(deletes []string) ([]string, errors.Error) {
	var done bool

	for i, name := range deletes {
		node, localKey := distributed.RemoteAccess().SplitKey(name)

		// remote entry
		if len(node) != 0 && node != distributed.RemoteAccess().WhoAmI() {

			distributed.RemoteAccess().GetRemoteDoc(node, localKey,
				"active_requests", "DELETE",
				nil,

				// FIXME Delete() doesn't do warnings
				func(warn errors.Error) {
				})
			done = true

			// local entry
		} else {
			done = server.ActiveRequestsDelete(localKey)
		}

		// save memory allocations by making a new slice only on errors
		if !done {
			deleted := make([]string, i)
			if i > 0 {
				copy(deleted, deletes[0:i-1])
			}
			return deleted, errors.NewSystemStmtNotFoundError(nil, name)
		}
	}
	return deletes, nil
}

func newActiveRequestsKeyspace(p *namespace) (*activeRequestsKeyspace, errors.Error) {
	b := new(activeRequestsKeyspace)
	b.namespace = p
	b.name = KEYSPACE_NAME_ACTIVE

	primary := &activeRequestsIndex{
		name:     "#primary",
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)

	// add a secondary index on `node`
	if distributed.RemoteAccess().WhoAmI() != "" {
		expr, err := parser.Parse(`node`)

		if err == nil {
			key := expression.Expressions{expr}
			nodes := &activeRequestsIndex{
				name:     "#nodes",
				keyspace: b,
				primary:  false,
				seekKey:  key,
			}
			b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
		} else {
			return nil, errors.NewSystemDatastoreError(err, "")
		}
	}

	return b, nil
}

type activeRequestsIndex struct {
	name     string
	keyspace *activeRequestsKeyspace
	primary  bool
	seekKey  expression.Expressions
}

func (pi *activeRequestsIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *activeRequestsIndex) Id() string {
	return pi.Name()
}

func (pi *activeRequestsIndex) Name() string {
	return pi.name
}

func (pi *activeRequestsIndex) Type() datastore.IndexType {
	return datastore.DEFAULT
}

func (pi *activeRequestsIndex) SeekKey() expression.Expressions {
	return pi.seekKey
}

func (pi *activeRequestsIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *activeRequestsIndex) Condition() expression.Expression {
	return nil
}

func (pi *activeRequestsIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *activeRequestsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *activeRequestsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *activeRequestsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *activeRequestsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil || len(span.Seek) == 0 || !pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		defer close(conn.EntryChannel())

	loop:
		for _, seek := range span.Seek {
			val := seek.Actual()
			switch t := val.(type) {
			case string:
			default:
				conn.Error(errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", t, val)))
				continue loop
			}
			key := val.(string)
			if key == distributed.RemoteAccess().WhoAmI() {
				server.ActiveRequestsForEach(func(id string, request server.Request) {
					entry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), id)}
					conn.EntryChannel() <- &entry
				})
			} else {
				distributed.RemoteAccess().GetRemoteKeys([]string{key}, "active_requests", func(id string) {
					indexEntry := datastore.IndexEntry{PrimaryKey: id}
					conn.EntryChannel() <- &indexEntry
				}, func(warn errors.Error) {
					conn.Warning(warn)
				})
			}
		}

	}
}

func (pi *activeRequestsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer close(conn.EntryChannel())

	server.ActiveRequestsForEach(func(id string, request server.Request) {
		entry := datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), id)}
		conn.EntryChannel() <- &entry
	})

	distributed.RemoteAccess().GetRemoteKeys([]string{}, "active_requests", func(id string) {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		conn.EntryChannel() <- &indexEntry
	}, func(warn errors.Error) {
		conn.Warning(warn)
	})
}
