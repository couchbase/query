//  Copyright 2022-Present Couchbase, Inc.
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
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type vitalsKeyspace struct {
	keyspaceBase
	si datastore.Indexer
}

func (b *vitalsKeyspace) Release(close bool) {
}

func (b *vitalsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *vitalsKeyspace) Id() string {
	return b.Name()
}

func (b *vitalsKeyspace) Name() string {
	return b.name
}

func (b *vitalsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {

	nodes := distributed.RemoteAccess().GetNodeNames()
	return int64(len(nodes)), nil
}

func (b *vitalsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *vitalsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *vitalsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

func (b *vitalsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string) (errs errors.Errors) {

	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {

		nodeName := decodeNodeName(key)

		if nodeName == "" {
			continue
		}

		// currently we query ourselves because there isn't a direct path from datastore/system to server
		if nodeName == whoAmI {
			doc, err := b.namespace.store.acctStore.Vitals(context.DurationStyle())
			if err != nil {
				context.Error(err)
			}
			remoteValue := value.NewAnnotatedValue(doc)
			remoteValue.SetField("node", key)
			remoteValue.NewMeta()["keyspace"] = b.fullName
			remoteValue.SetId(key)
			keysMap[key] = remoteValue
		} else {
			distributed.RemoteAccess().GetRemoteDoc(nodeName, "", "vitals", "GET",
				func(doc map[string]interface{}) {
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", key)
					remoteValue.NewMeta()["keyspace"] = b.fullName
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				}, distributed.NO_CREDS, "", formData)
		}

	}

	return
}

func newVitalsKeyspace(p *namespace) (*vitalsKeyspace, errors.Error) {
	b := new(vitalsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_VITALS)

	primary := &vitalsIndex{name: "#primary", keyspace: b}
	b.si = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.si)

	return b, nil
}

type vitalsIndex struct {
	indexBase
	name     string
	keyspace *vitalsKeyspace
}

func (pi *vitalsIndex) KeyspaceId() string {
	return pi.name
}

func (pi *vitalsIndex) Id() string {
	return pi.Name()
}

func (pi *vitalsIndex) Name() string {
	return pi.name
}

func (pi *vitalsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *vitalsIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *vitalsIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *vitalsIndex) Condition() expression.Expression {
	return nil
}

func (pi *vitalsIndex) IsPrimary() bool {
	return true
}

func (pi *vitalsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *vitalsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *vitalsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *vitalsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	if span == nil {
		pi.ScanEntries(requestId, limit, cons, vector, conn)
	} else {
		var numProduced int64 = 0
		defer conn.Sender().Close()

		spanEvaluator, err := compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
		nodes := distributed.RemoteAccess().GetNodeNames()
		for _, node := range nodes {
			key := distributed.RemoteAccess().MakeKey(encodeNodeName(node), "")
			if spanEvaluator.evaluate(key) {
				entry := datastore.IndexEntry{PrimaryKey: key}
				if !sendSystemKey(conn, &entry) {
					return
				}
				numProduced++
				if limit > 0 && numProduced >= limit {
					break
				}
			}
		}
	}
}

func (pi *vitalsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	var numProduced int64 = 0

	defer conn.Sender().Close()
	nodes := distributed.RemoteAccess().GetNodeNames()
	for _, node := range nodes {
		entry = &datastore.IndexEntry{PrimaryKey: encodeNodeName(node)}
		if !sendSystemKey(conn, entry) {
			return
		}
		numProduced++
		if limit > 0 && numProduced >= limit {
			break
		}
	}
}
