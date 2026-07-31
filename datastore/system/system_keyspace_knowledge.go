//  Copyright 2026-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/natural/knowledge"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type knowledgeKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *knowledgeKeyspace) Release(close bool) {
}

func (b *knowledgeKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *knowledgeKeyspace) Id() string {
	return b.Name()
}

func (b *knowledgeKeyspace) Name() string {
	return b.name
}

func (b *knowledgeKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int64

	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())
	for _, bucket := range buckets {
		err := knowledge.Scan(bucket, func(string) error {
			count++
			return nil
		})
		if err != nil {
			return 0, errors.NewStorageAccessError("count", err)
		}
	}
	return count, nil
}

func (b *knowledgeKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *knowledgeKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *knowledgeKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *knowledgeKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	// unlike Scan/Count, USE KEYS bypasses the buckets loop those use to restrict results to
	// datastore.GetUserBuckets(), so this needs its own check: without it, any authenticated user
	// could USE KEYS a composite key naming a bucket they have no privileges on and read its
	// knowledge entries - or even just learn whether that bucket exists from
	// the error FetchEntry would return for it. So the check happens before FetchEntry is ever
	// called, and a disallowed bucket is skipped silently, exactly like an ordinary not-found key.
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())
	allowed := make(map[string]bool, len(buckets))
	for _, bucket := range buckets {
		allowed[bucket] = true
	}

	for _, k := range keys {
		bk, e := knowledge.BucketFromExtKey(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}
		if !allowed[bk] {
			keysMap[k] = nil
			continue
		}

		item, e := knowledge.FetchEntry(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			item.SetMetaField(value.META_KEYSPACE, b.fullName)
			item.SetId(k)
		}
		keysMap[k] = item
	}

	return
}

func newKnowledgeKeyspace(p *namespace) (*knowledgeKeyspace, errors.Error) {
	b := new(knowledgeKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_KNOWLEDGE)

	primary := &knowledgeIndex{name: PRIMARY_INDEX_NAME, keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `bucket`
	expr, err := parser.Parse("`bucket`")

	if err == nil {
		key := expression.Expressions{expr}
		buckets := &knowledgeIndex{
			name:     "#buckets",
			keyspace: b,
			primary:  false,
			idxKey:   key,
		}
		setIndexBase(&buckets.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(buckets.name, buckets)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}
	return b, nil
}

type knowledgeIndex struct {
	indexBase
	name     string
	keyspace *knowledgeKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *knowledgeIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *knowledgeIndex) Id() string {
	return pi.Name()
}

func (pi *knowledgeIndex) Name() string {
	return pi.name
}

func (pi *knowledgeIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *knowledgeIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *knowledgeIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *knowledgeIndex) Condition() expression.Expression {
	return nil
}

func (pi *knowledgeIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *knowledgeIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *knowledgeIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *knowledgeIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *knowledgeIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	var spanEvaluator compiledSpans
	var err errors.Error

	if span != nil && !pi.primary {
		spanEvaluator, err = compileSpan(span)
		if err != nil {
			conn.Error(err)
			return
		}
	}
	pi.scanEntries(requestId, spanEvaluator, limit, cons, vector, conn)
}

func (pi *knowledgeIndex) Scan2(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection,
	ordered bool, projection *datastore.IndexProjection, offset, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var spanEvaluator compiledSpans
	var err errors.Error

	if spans != nil && !pi.primary {
		spanEvaluator, err = compileSpan2(spans)
		if err != nil {
			conn.Error(err)
			return
		}
	}
	pi.scanEntries(requestId, spanEvaluator, limit, cons, vector, conn)
}

func (pi *knowledgeIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.scanEntries(requestId, nil, limit, cons, vector, conn)
}

func (pi *knowledgeIndex) scanEntries(requestId string, spanEvaluator compiledSpans, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	context := conn.QueryContext()
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())
	for _, b := range buckets {
		if len(spanEvaluator) > 0 && !spanEvaluator.evaluate(b) {
			continue
		}
		err := knowledge.Scan(b, func(key string) error {
			entry := datastore.IndexEntry{PrimaryKey: key}
			sendSystemKey(conn, &entry)
			return nil
		})
		if err != nil {
			conn.Error(errors.NewStorageAccessError("scan", err))
			return
		}
	}
}
