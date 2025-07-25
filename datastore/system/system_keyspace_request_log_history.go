//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type requestLogHistoryKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *requestLogHistoryKeyspace) Release(close bool) {
}

func (b *requestLogHistoryKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *requestLogHistoryKeyspace) Id() string {
	return b.Name()
}

func (b *requestLogHistoryKeyspace) Name() string {
	return b.name
}

func (b *requestLogHistoryKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests_history", func(id string) bool {
		_, localKey := distributed.RemoteAccess().SplitKey(id)
		if _, c, ok := splitHistoryKey(localKey); ok {
			count += int(c)
		}
		return true
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			context.Warning(warn)
		}
	}, creds, "")

	info := server.RequestsFileStreamFileInfo()
	if userName == "" {
		for i := 0; i < len(info); i += 2 {
			count += int(info[i+1])
		}
	} else {
		// we have to go load and check every entry
		for i := 0; i < len(info); i += 2 {
			server.RequestsFileStreamRead(info[i], 0, 0, userName, func(m map[string]interface{}) bool {
				count++
				return true
			})
		}
	}
	return int64(count), nil
}

func (b *requestLogHistoryKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *requestLogHistoryKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *requestLogHistoryKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *requestLogHistoryKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	rangeStart := uint64(0)
	rangeEnd := uint64(0)
	rangeNum := uint64(0)
	var rangeKeys []int
	n := 0

	var node string
	var nodeName string
	var localKey string

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()

	local := func(doc map[string]interface{}) bool {
		if n >= len(rangeKeys) {
			return false
		}
		key := keys[rangeKeys[n]]
		node, _ = distributed.RemoteAccess().SplitKey(key)
		docKey, ok := doc["requestId"]
		if ok {
			docKey = distributed.RemoteAccess().MakeKey(whoAmI, docKey.(string))
		} else {
			docKey = keys[rangeKeys[n]]
		}
		t, ok := doc["timings"]
		if ok {
			delete(doc, "timings")
		}
		o, ook := doc["optimizerEstimates"]
		if ook {
			delete(doc, "optimizerEstimates")
		}
		localValue := value.ApplyDurationStyleToValue(context.DurationStyle(),
			value.NewAnnotatedValue(doc)).(value.AnnotatedValue)
		meta := localValue.NewMeta()
		meta["keyspace"] = b.fullName
		if ok {
			meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), value.NewValue(t))
		}
		if ook {
			meta["optimizerEstimates"] = value.NewValue(o)
		}
		if node != "" {
			localValue.SetField("node", node)
		}
		localValue.SetField("~file", rangeNum)
		localValue.SetId(docKey)
		keysMap[key] = localValue
		n++
		return true
	}

	for keyNum, key := range keys {
		node, localKey = distributed.RemoteAccess().SplitKey(key)
		nodeName = decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey, "completed_requests_history", "GET",
				func(doc map[string]interface{}) {
					docKey, ok := doc["requestId"]
					if ok {
						docKey = distributed.RemoteAccess().MakeKey(node, docKey.(string))
					} else {
						docKey = key
					}
					t, ok := doc["timings"]
					if ok {
						delete(doc, "timings")
					}
					o, ook := doc["optimizerEstimates"]
					if ook {
						delete(doc, "optimizerEstimates")
					}
					remoteValue := value.ApplyDurationStyleToValue(context.DurationStyle(),
						value.NewAnnotatedValue(doc)).(value.AnnotatedValue)
					meta := remoteValue.NewMeta()
					meta["keyspace"] = b.fullName
					if ok {
						meta["plan"] = value.ApplyDurationStyleToValue(context.DurationStyle(), value.NewValue(t))
					}
					if ook {
						meta["optimizerEstimates"] = value.NewValue(o)
					}
					remoteValue.SetField("node", node)
					remoteValue.SetId(docKey)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) && !warn.ContainsText("object not found") {
						context.Warning(warn)
					}
				},
				creds, "", formData)
		} else {
			// local entry
			if fileNum, recNum, ok := splitHistoryKey(localKey); ok {
				if len(rangeKeys) != 0 && fileNum == rangeNum && recNum == rangeEnd {
					rangeEnd++
					rangeKeys = append(rangeKeys, keyNum)
				} else {
					if len(rangeKeys) > 0 {
						n = 0
						server.RequestsFileStreamRead(rangeNum, rangeStart, rangeEnd-rangeStart, userName, local)
					}
					rangeNum = fileNum
					rangeStart = recNum
					rangeEnd = recNum + 1
					if rangeKeys != nil {
						rangeKeys = rangeKeys[:0]
					} else {
						rangeKeys = make([]int, 0, 10)
					}
					rangeKeys = append(rangeKeys, keyNum)
				}
			}
		}
	}
	if rangeStart < rangeEnd {
		n = 0
		server.RequestsFileStreamRead(rangeNum, rangeStart, rangeEnd-rangeStart, userName, local)
	}
	return
}

func newRequestsHistoryKeyspace(p *namespace) (*requestLogHistoryKeyspace, errors.Error) {
	b := new(requestLogHistoryKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_REQUESTS_HISTORY)

	primary := &requestLogHistoryIndex{
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
		nodes := &requestLogHistoryIndex{
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

type requestLogHistoryIndex struct {
	indexBase
	name     string
	keyspace *requestLogHistoryKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *requestLogHistoryIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *requestLogHistoryIndex) Id() string {
	return pi.Name()
}

func (pi *requestLogHistoryIndex) Name() string {
	return pi.name
}

func (pi *requestLogHistoryIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *requestLogHistoryIndex) SeekKey() expression.Expressions {
	return pi.idxKey
}

func (pi *requestLogHistoryIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *requestLogHistoryIndex) Condition() expression.Expression {
	return nil
}

func (pi *requestLogHistoryIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *requestLogHistoryIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	if pi.primary || distributed.RemoteAccess().WhoAmI() != "" {
		return datastore.ONLINE, "", nil
	} else {
		return datastore.OFFLINE, "", nil
	}
}

func (pi *requestLogHistoryIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *requestLogHistoryIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *requestLogHistoryIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {

	if span == nil || pi.primary {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		var creds distributed.Creds
		var local func()

		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}

		// now that the node name can change in flight, use a consistent one across the scan
		whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())
		userName := credsFromContext(conn.Context())
		if userName == "" {
			creds = distributed.NO_CREDS
			local = func() {
				info := server.RequestsFileStreamFileInfo()
				for i := 0; i < len(info); i += 2 {
					for j := uint64(0); j < info[i+1]; j++ {
						indexEntry := datastore.IndexEntry{
							PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, fmt.Sprintf("%d-%d", info[i], j)),
							EntryKey:   value.Values{value.NewValue(whoAmI)},
						}
						if !sendSystemKey(conn, &indexEntry) {
							break
						}
					}
				}
			}
		} else {
			creds = distributed.Creds(userName)
			local = func() {
				info := server.RequestsFileStreamFileInfo()
				n := 0
				for i := 0; i < len(info); i += 2 {
					server.RequestsFileStreamRead(info[i], 0, 0, userName, func(m map[string]interface{}) bool {
						indexEntry := datastore.IndexEntry{
							PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, fmt.Sprintf("%d-%d", i, n)),
							EntryKey:   value.Values{value.NewValue(whoAmI)},
						}
						n++
						return sendSystemKey(conn, &indexEntry)
					})
				}
			}
		}
		remote := func(id string) bool {
			n, localKey := distributed.RemoteAccess().SplitKey(id)
			if fileNum, c, ok := splitHistoryKey(localKey); ok {
				// generate individual keys
				for j := uint64(0); j < c; j++ {
					indexEntry := datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(n, fmt.Sprintf("%d-%v", fileNum, j)),
						EntryKey:   value.Values{value.NewValue(n)},
					}
					if !sendSystemKey(conn, &indexEntry) {
						return false
					}
				}
			}
			return true
		}

		idx := spanEvaluator.isEquals()
		if idx >= 0 {
			if spanEvaluator.key(idx) == whoAmI {
				local()
			} else {
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "completed_requests_history", remote,
					func(warn errors.Error) {
						if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
							conn.Warning(warn)
						}
					}, creds, "")
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				encodedNode := encodeNodeName(node)
				if spanEvaluator.evaluate(encodedNode) {
					if encodedNode == whoAmI {
						local()
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "completed_requests_history", remote,
					func(warn errors.Error) {
						if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
							conn.Warning(warn)
						}
					}, creds, "")
			}
		}
	}
}

func (pi *requestLogHistoryIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	var creds distributed.Creds
	var local func()

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())

	userName := credsFromContext(conn.Context())
	if userName == "" {
		creds = distributed.NO_CREDS
		local = func() {
			info := server.RequestsFileStreamFileInfo()
			for i := 0; i < len(info); i += 2 {
				for j := uint64(0); j < info[i+1]; j++ {
					indexEntry := datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, fmt.Sprintf("%d-%d", info[i], j)),
					}
					if !sendSystemKey(conn, &indexEntry) {
						break
					}
				}
			}
		}
	} else {
		creds = distributed.Creds(userName)
		local = func() {
			info := server.RequestsFileStreamFileInfo()
			n := 0
			for i := 0; i < len(info); i += 2 {
				server.RequestsFileStreamRead(info[i], 0, 0, userName, func(m map[string]interface{}) bool {
					indexEntry := datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, fmt.Sprintf("%d-%d", info[i], n)),
					}
					n++
					return sendSystemKey(conn, &indexEntry)
				})
			}
		}
	}
	remote := func(id string) bool {
		n, localKey := distributed.RemoteAccess().SplitKey(id)
		if fileNum, c, ok := splitHistoryKey(localKey); ok {
			// generate individual keys
			for j := uint64(0); j < c; j++ {
				indexEntry := datastore.IndexEntry{
					PrimaryKey: distributed.RemoteAccess().MakeKey(n, fmt.Sprintf("%d-%v", fileNum, j)),
				}
				if !sendSystemKey(conn, &indexEntry) {
					return false
				}
			}
		}
		return true
	}

	local()
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "completed_requests_history", remote,
		func(warn errors.Error) {
			if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
				conn.Warning(warn)
			}
		}, creds, "")
}

func splitHistoryKey(key string) (uint64, uint64, bool) {
	i := strings.LastIndexByte(key, '-')
	if i == -1 {
		return 0, 0, false
	}
	fileNum, e := strconv.ParseUint(key[:i], 10, 64)
	if e != nil {
		return 0, 0, false
	}
	recNumOrCount, e := strconv.ParseUint(key[i+1:], 10, 64)
	if e != nil {
		return 0, 0, false
	}
	return fileNum, recNumOrCount, true
}
