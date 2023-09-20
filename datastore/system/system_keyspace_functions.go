//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	functionsStorage "github.com/couchbase/query/functions/storage"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

type functionsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *functionsKeyspace) Release(close bool) {
}

func (b *functionsKeyspace) NamespaceId() string {
	return b.namespace.Id()
}

func (b *functionsKeyspace) Id() string {
	return b.Name()
}

func (b *functionsKeyspace) Name() string {
	return b.name
}

func (b *functionsKeyspace) Count(context datastore.QueryContext) (int64, errors.Error) {
	var count int64

	internal, external := hasGlobalFunctionsAccess(context)
	lastScope := ""
	hasScopeInternal := false
	hasScopeExternal := false
	isAdmin := datastore.IsAdmin(context.Credentials())
	if internal || external {
		err := functionsStorage.Foreach("", func(path string, v value.Value) error {
			if !isAdmin {
				i, err := functionsStorage.IsInternal(v)
				if err != nil {
					return err
				}
				if (i && !internal) || (!i && !external) {
					return nil
				}
			}
			count++
			return nil
		})
		if err != nil {
			return 0, errors.NewStorageAccessError("count", err)
		}
	}
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())
	for _, b := range buckets {
		err := functionsStorage.Foreach(b, func(path string, v value.Value) error {
			if !isAdmin {
				parts := algebra.ParsePath(path)
				scope := parts[1] + "." + parts[2]
				if scope != lastScope {
					hasScopeInternal, hasScopeExternal = hasScopeFunctionsAccess(path, context)
					lastScope = scope
				}
				i, err := functionsStorage.IsInternal(v)
				if err != nil {
					return err
				}
				if (i && !hasScopeInternal) || (!i && !hasScopeExternal) {
					return nil
				}
			}
			count++
			return nil
		})
		if err != nil {
			return 0, errors.NewStorageAccessError("count", err)
		}
	}
	return count, nil
}

func (b *functionsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *functionsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *functionsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *functionsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {

	internal, external := hasGlobalFunctionsAccess(context)
	lastScope := ""
	hasScopeInternal := false
	hasScopeExternal := false
	isAdmin := datastore.IsAdmin(context.Credentials())
	for _, k := range keys {
		item, e := b.fetchOne(k)
		if e != nil {
			if errs == nil {
				errs = make([]errors.Error, 0, 1)
			}
			errs = append(errs, e)
			continue
		}

		if item != nil {
			if !isAdmin {
				parts := algebra.ParsePath(k)
				switch len(parts) {
				case 2:
					i, err := functionsStorage.IsInternal(item)
					if err != nil {
						if errs == nil {
							errs = make([]errors.Error, 0, 1)
						}
						errs = append(errs, errors.NewStorageAccessError("Fetch", err))
						continue
					}
					if (i && !internal) || (!i && !external) {
						continue
					}
				case 4:
					scope := parts[1] + "." + parts[2]
					if scope != lastScope {
						hasScopeInternal, hasScopeExternal = hasScopeFunctionsAccess(k, context)
						lastScope = scope
					}
					i, err := functionsStorage.IsInternal(item)
					if err != nil {
						if errs == nil {
							errs = make([]errors.Error, 0, 1)
						}
						errs = append(errs, errors.NewStorageAccessError("Fetch", err))
						continue
					}
					if (i && !hasScopeInternal) || (!i && !hasScopeExternal) {
						continue
					}
				default:
				}
			}

			item.NewMeta()["keyspace"] = b.fullName
			item.SetId(k)
		}
		keysMap[k] = item
	}

	return
}

func (b *functionsKeyspace) fetchOne(key string) (value.AnnotatedValue, errors.Error) {
	body, err := functionsStorage.Get(key)

	// get does not return is not found, but nil, nil instead
	if err == nil && body == nil {
		return nil, errors.NewSystemDatastoreError(nil, "Key Not Found "+key)
	}
	if err != nil {
		return nil, errors.NewStorageAccessError("Fetch", err)
	}
	return value.NewAnnotatedValue(body), nil
}

func newFunctionsKeyspace(p *namespace) (*functionsKeyspace, errors.Error) {
	b := new(functionsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_FUNCTIONS)

	primary := &functionsIndex{name: "#primary", keyspace: b, primary: true}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

	// add a secondary index on `bucket_id`
	expr, err := parser.Parse("`identity`.`bucket`")

	if err == nil {
		key := expression.Expressions{expr}
		buckets := &functionsIndex{
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

type functionsIndex struct {
	indexBase
	name     string
	keyspace *functionsKeyspace
	primary  bool
	idxKey   expression.Expressions
}

func (pi *functionsIndex) KeyspaceId() string {
	return pi.name
}

func (pi *functionsIndex) Id() string {
	return pi.Name()
}

func (pi *functionsIndex) Name() string {
	return pi.name
}

func (pi *functionsIndex) Type() datastore.IndexType {
	return datastore.SYSTEM
}

func (pi *functionsIndex) SeekKey() expression.Expressions {
	return nil
}

func (pi *functionsIndex) RangeKey() expression.Expressions {
	return pi.idxKey
}

func (pi *functionsIndex) RangeKey2() datastore.IndexKeys {
	if !pi.primary {
		rangeKey := &datastore.IndexKey{
			Expr: pi.idxKey[0],
		}
		rangeKey.SetAttribute(datastore.IK_MISSING, true)
		return datastore.IndexKeys{rangeKey}
	}
	return nil
}

func (pi *functionsIndex) Condition() expression.Expression {
	return nil
}

func (pi *functionsIndex) IsPrimary() bool {
	return pi.primary
}

func (pi *functionsIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (pi *functionsIndex) Statistics(requestId string, span *datastore.Span) (
	datastore.Statistics, errors.Error) {
	return nil, nil
}

func (pi *functionsIndex) Drop(requestId string) errors.Error {
	return errors.NewSystemIdxNoDropError(nil, pi.Name())
}

func (pi *functionsIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64,
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

func (pi *functionsIndex) Scan2(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection,
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

func hasGlobalFunctionsAccess(context datastore.QueryContext) (bool, bool) {
	privs1 := auth.NewPrivileges()
	privs1.Add("", auth.PRIV_QUERY_MANAGE_FUNCTIONS, auth.PRIV_PROPS_NONE)
	privs2 := auth.NewPrivileges()
	privs2.Add("", auth.PRIV_QUERY_EXECUTE_FUNCTIONS, auth.PRIV_PROPS_NONE)
	err1 := datastore.GetDatastore().Authorize(privs1, context.Credentials())
	err2 := datastore.GetDatastore().Authorize(privs2, context.Credentials())
	internal := err1 == nil || err2 == nil

	privs1 = auth.NewPrivileges()
	privs1.Add("", auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
	privs2 = auth.NewPrivileges()
	privs2.Add("", auth.PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
	err1 = datastore.GetDatastore().Authorize(privs1, context.Credentials())
	err2 = datastore.GetDatastore().Authorize(privs2, context.Credentials())
	external := err1 == nil || err2 == nil
	return internal, external
}

func hasScopeFunctionsAccess(path string, context datastore.QueryContext) (bool, bool) {
	privs1 := auth.NewPrivileges()
	privs1.Add(path, auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS, auth.PRIV_PROPS_NONE)
	privs2 := auth.NewPrivileges()
	privs2.Add(path, auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS, auth.PRIV_PROPS_NONE)
	err1 := datastore.GetDatastore().Authorize(privs1, context.Credentials())
	err2 := datastore.GetDatastore().Authorize(privs2, context.Credentials())
	internal := err1 == nil || err2 == nil

	privs1 = auth.NewPrivileges()
	privs1.Add(path, auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
	privs2 = auth.NewPrivileges()
	privs2.Add(path, auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
	err1 = datastore.GetDatastore().Authorize(privs1, context.Credentials())
	err2 = datastore.GetDatastore().Authorize(privs2, context.Credentials())
	external := err1 == nil || err2 == nil
	return internal, external
}

func (pi *functionsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	pi.scanEntries(requestId, nil, limit, cons, vector, conn)
}

func (pi *functionsIndex) scanEntries(requestId string, spanEvaluator compiledSpans, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	defer conn.Sender().Close()

	context := conn.QueryContext()
	internal, external := hasGlobalFunctionsAccess(context)
	if (internal || external) && (len(spanEvaluator) == 0 || spanEvaluator.acceptMissing()) {
		err := functionsStorage.Scan("", func(path string) error {
			entry := datastore.IndexEntry{PrimaryKey: path}
			sendSystemKey(conn, &entry)
			return nil
		})
		if err != nil {
			conn.Error(errors.NewStorageAccessError("scan", err))
			return
		}
	}
	buckets := datastore.GetDatastore().GetUserBuckets(context.Credentials())
	for _, b := range buckets {
		if len(spanEvaluator) > 0 && !spanEvaluator.evaluate(b) {
			continue
		}
		err := functionsStorage.Scan(b, func(path string) error {
			entry := datastore.IndexEntry{PrimaryKey: path}
			sendSystemKey(conn, &entry)
			return nil
		})
		if err != nil {
			conn.Error(errors.NewStorageAccessError("scan", err))
			return
		}
	}
}
