//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type nodeKeyspace struct {
	keyspaceBase
	si datastore.Indexer
}

func (b *nodeKeyspace) Release(close bool) {
}

func (b *nodeKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *nodeKeyspace) Id() string {
	return b.Name()
}

func (b *nodeKeyspace) Name() string {
	return b.name
}

func (b *nodeKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var err errors.Error

	topology, errs := b.namespace.store.actualStore.Info().Topology()
	if errs != nil {
		err = errs[0]
	}
	return int64(len(topology)), err
}

func (b *nodeKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *nodeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.si, nil
}

func (b *nodeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.si}, nil
}

func (b *nodeKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	info := b.namespace.store.actualStore.Info()

	for _, k := range keys {

		nodeServices, errList := info.Services(k)

		if nodeServices != nil {
			item := value.NewAnnotatedValue(nodeServices)
			item.SetMetaField(value.META_KEYSPACE, b.fullName)
			item.SetId(k)
			keysMap[k] = item
			continue
		} else if errList != nil {
			for _, err := range errList {
				errs = appendError(errs, err)
			}
		} else {
			errs = appendError(errs, errors.NewSystemDatastoreError(nil, "Key Not Found "+k))
		}
	}

	return
}

func appendError(errs []errors.Error, err errors.Error) []errors.Error {
	if errs == nil {
		errs = make([]errors.Error, 0, 1)
	}
	errs = append(errs, err)
	return errs
}

func newNodesKeyspace(p *namespace) (*nodeKeyspace, errors.Error) {
	b := new(nodeKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_NODES)

	primary := &nodeIndex{name: PRIMARY_INDEX_NAME, keyspace: b}
	b.si = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.si)

	return b, nil
}

type nodeIndex struct {
	indexBase
	name     string
	keyspace *nodeKeyspace
}

func (pi *nodeIndex) KeyspaceId() string {
	return pi.name
}

func (pi *nodeIndex) Id() string {
	return pi.Name()
}

func (pi *nodeIndex) Name() string {
	return pi.name
}

func (pi *nodeIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *nodeIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *nodeIndex) RangeKey() expression.Expressions {
	return nil
}

func (pi *nodeIndex) Condition() expression.Expression {
	return nil
}

func (pi *nodeIndex) IsPrimary() bool {
	return true
}

func (pi *nodeIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *nodeIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *nodeIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *nodeIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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
		info := pi.keyspace.namespace.store.actualStore.Info()
		topology, errs := info.Topology()
		for _, key := range topology {
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
		for _, err = range errs {
			conn.Error(err)
		}
	}
}

func (pi *nodeIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var numProduced int64 = 0

	defer conn.Sender().Close()
	info := pi.keyspace.namespace.store.actualStore.Info()
	topology, errs := info.Topology()
	for _, key := range topology {
		entry := datastore.IndexEntry{PrimaryKey: key}
		if !sendSystemKey(conn, &entry) {
			return
		}
		numProduced++
		if limit > 0 && numProduced >= limit {
			break
		}
	}
	for _, err := range errs {
		conn.Error(err)
	}
}
