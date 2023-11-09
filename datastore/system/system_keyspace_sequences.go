//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"math"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/sequences"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type sequenceKeyspace struct {
	keyspaceBase
	skipSystem bool
	store      datastore.Datastore
	indexer    datastore.Indexer
}

func (b *sequenceKeyspace) Release(close bool) {
}

func (b *sequenceKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *sequenceKeyspace) Id() string {
	return b.Name()
}

func (b *sequenceKeyspace) Name() string {
	return b.name
}

func (b *sequenceKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var namespace datastore.Namespace
	var bucket datastore.Bucket
	var scope datastore.Scope
	var objects []datastore.Object
	var keys []string

	count := int64(0)
	namespaceIds, err := b.store.NamespaceIds()
	if err == nil {
		canAccessAll := canAccessSystemTables(context)
		includeOnDisk := b.Name() == KEYSPACE_NAME_ALL_SEQUENCES
		for _, namespaceId := range namespaceIds {
			namespace, err = b.store.NamespaceById(namespaceId)
			if err != nil {
				continue
			}
			ds := namespace.Datastore()

			objects, err = namespace.Objects(context.Credentials(), nil, true)
			if err != nil {
				continue
			}
			for _, object := range objects {
				if object.IsBucket {
					bucket, err = namespace.BucketById(object.Id)
					if err != nil {
						continue
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, err = bucket.ScopeById(scopeId)
						if scope != nil {
							if canAccessAll || canRead(context, ds, namespaceId, object.Id, scopeId) ||
								canListSequences(context, ds, namespaceId, object.Id, scopeId) {

								keys, err = sequences.ListSequenceKeys(namespaceId, object.Id, scopeId, !includeOnDisk,
									math.MaxInt64)
								count += int64(len(keys))
								keys = nil
							} else if includeOnDisk {
								context.Warning(errors.NewSystemFilteredRowsWarning("system:sequences"))
							}
						}
					}
				}
			}
		}
	}

	if err == nil {
		return count, nil
	}
	return 0, errors.NewSystemDatastoreError(err, "")
}

func (b *sequenceKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *sequenceKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *sequenceKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *sequenceKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	whoAmI := distributed.RemoteAccess().WhoAmI()
	nodes := distributed.RemoteAccess().GetNodeNames()

	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	for _, key := range keys {
		av, err := sequences.FetchSequence(key, false)
		if err != nil {
			errs = append(errs, err)
			continue
		} else if av != nil {
			if len(nodes) > 1 {
				val, _ := av.Field("value")
				for _, node := range nodes {
					if node != whoAmI {
						nodeUUID := distributed.RemoteAccess().NodeUUID(node)
						distributed.RemoteAccess().GetRemoteDoc(node, key, "sequences_cache", "GET",
							func(doc map[string]interface{}) {
								if v, ok := doc["value"]; ok {
									if vm, ok := v.(map[string]interface{}); ok {
										if vv, ok := vm[nodeUUID]; ok {
											val.SetField(nodeUUID, vv)
										}
									}
								}
							}, nil, distributed.NO_CREDS, "", formData)
					}
				}
				av.SetField("value", val)
			}
			av.SetId(key)
			keysMap[key] = av
		}
	}

	return
}

func newSequencesKeyspace(p *namespace, store datastore.Datastore, name string, skipSystem bool) (*sequenceKeyspace, errors.Error) {
	b := new(sequenceKeyspace)
	b.store = store
	b.skipSystem = skipSystem
	setKeyspaceBase(&b.keyspaceBase, p, name)

	primary := &sequenceIndex{name: "#primary", keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `bucket`
	expr, err := parser.Parse("`bucket`")

	if err == nil {
		key := expression.Expressions{expr}
		buckets := &sequenceIndex{
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

type sequenceIndex struct {
	indexBase
	name     string
	keyspace *sequenceKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *sequenceIndex) KeyspaceId() string {
	return pi.keyspace.Id()
}

func (pi *sequenceIndex) Id() string {
	return pi.Name()
}

func (pi *sequenceIndex) Name() string {
	return pi.name
}

func (pi *sequenceIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *sequenceIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *sequenceIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *sequenceIndex) Condition() expression.Expression {
	return nil
}

func (pi *sequenceIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *sequenceIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *sequenceIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *sequenceIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, "")
}

func (pi *sequenceIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
	var filter func(string) bool
	spanEvaluator, err := compileSpan(span)
	if err != nil {
		conn.Error(err)
		return
	}
	if !pi.primary {
		filter = func(name string) bool {
			return spanEvaluator.evaluate(name)
		}
	}
	pi.doScanEntries(requestId, filter, limit, cons, vector, conn)
}

// Do the presented credentials authorize the user to list sequences for the scope ?
func canListSequences(context datastore.QueryContext, ds datastore.Datastore, namespace string, bucket string, scope string) bool {
	path := algebra.NewPathScope(namespace, bucket, scope).FullName()
	privs := auth.NewPrivileges()
	privs.Add(path, auth.PRIV_QUERY_USE_SEQUENCES, auth.PRIV_PROPS_NONE)
	creds := context.Credentials()
	if ds.Authorize(privs, creds) == nil {
		return true
	}
	privs.List[0].Priv = auth.PRIV_QUERY_MANAGE_SEQUENCES
	return (ds.Authorize(privs, creds) == nil)
}

func (pi *sequenceIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.doScanEntries(requestId, func(string) bool { return true }, limit, cons, vector, conn)
}

func (pi *sequenceIndex) doScanEntries(requestId string, filter func(string) bool, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {

	var creds distributed.Creds

	defer conn.Sender().Close()

	dedup := make(map[string]bool)

	userName := credsFromContext(conn.Context())
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}

	includeOnDisk := pi.keyspace.Name() == KEYSPACE_NAME_ALL_SEQUENCES
	namespaceIds, err := pi.keyspace.store.NamespaceIds()
	if err == nil {
		canAccessAll := canAccessSystemTables(conn.QueryContext())
		for _, namespaceId := range namespaceIds {
			namespace, err := pi.keyspace.store.NamespaceById(namespaceId)
			if err != nil {
				continue
			}
			ds := namespace.Datastore()

			objects, err := namespace.Objects(conn.QueryContext().Credentials(), filter, true)
			if err != nil {
				continue
			}
		loop:
			for _, object := range objects {
				if !pi.primary && !filter(object.Id) {
					continue
				}
				if object.IsBucket {
					bucket, excp := namespace.BucketById(object.Id)
					if excp != nil {
						continue loop
					}
					scopeIds, _ := bucket.ScopeIds()
					for _, scopeId := range scopeIds {
						scope, _ := bucket.ScopeById(scopeId)
						if scope != nil {
							id := makeId(namespaceId, object.Id, scopeId)
							if !pi.primary || filter(id) {
								if canAccessAll || canRead(conn.QueryContext(), ds, namespaceId, object.Id, scopeId) ||
									canListSequences(conn.QueryContext(), ds, namespaceId, object.Id, scopeId) {

									keys, err := sequences.ListSequenceKeys(namespaceId, object.Id, scopeId, !includeOnDisk, limit)
									if err != nil {
										continue loop
									}
									for _, key := range keys {
										if _, ok := dedup[key]; ok {
											continue
										}
										dedup[key] = true
										entry := datastore.IndexEntry{PrimaryKey: key}
										if !sendSystemKey(conn, &entry) {
											return
										}
										limit--
									}
									keys = nil
								} else if includeOnDisk {
									conn.Warning(errors.NewSystemFilteredRowsWarning("system:sequences"))
								}
							}
						}
					}
				}
			}
		}
	}
	if !includeOnDisk {
		distributed.RemoteAccess().GetRemoteKeys([]string{}, "sequences", func(name string) bool {
			n := strings.Index(name, "]")
			if n != -1 {
				name = name[n+1:]
			}
			if _, ok := dedup[name]; ok {
				return true
			}
			dedup[name] = true
			indexEntry := datastore.IndexEntry{PrimaryKey: name}
			return sendSystemKey(conn, &indexEntry)
		}, func(warn errors.Error) {
			conn.Warning(warn)
		}, creds, "")
	}
}
