//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type preparedsKeyspace struct {
	keyspaceBase
	indexer datastore.Indexer
}

func (b *preparedsKeyspace) Release(close bool) {
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
	var creds distributed.Creds

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		creds = distributed.Creds(userName)
	}
	count = 0
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "prepareds", func(id string) bool {
		count++
		return true
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			context.Warning(warn)
		}
	}, creds, "")
	if userName == "" {
		count += prepareds.CountPrepareds()
	} else {
		tenantName := getTenant(context.TenantCtx())
		prepareds.PreparedsForeach(func(name string, prepared *prepareds.CacheEntry) bool {
			if checkCacheEntry(prepared, tenantName) {
				count++
			}
			return true
		}, nil)
	}
	return int64(count), nil
}

func (b *preparedsKeyspace) Size(context datastore.QueryContext) (int64, errors.Error) {
	return -1, nil
}

func (b *preparedsKeyspace) Indexer(name datastore.IndexType) (datastore.Indexer, errors.Error) {
	return b.indexer, nil
}

func (b *preparedsKeyspace) Indexers() ([]datastore.Indexer, errors.Error) {
	return []datastore.Indexer{b.indexer}, nil
}

func (b *preparedsKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPaths []string, projection []string, useSubDoc bool) (errs errors.Errors) {
	var creds distributed.Creds
	var tenantName string

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		tenantName = getTenant(context.TenantCtx())
		creds = distributed.Creds(userName)
	}

	formData := map[string]interface{}{"duration_style": context.DurationStyle().String()}

	// now that the node name can change in flight, use a consistent one across fetches
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for _, key := range keys {
		node, localKey := distributed.RemoteAccess().SplitKey(key)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {
			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey, "prepareds", "POST",
				func(doc map[string]interface{}) {
					remoteValue := value.NewAnnotatedValue(doc)
					remoteValue.SetField("node", node)

					remoteValue.SetMetaField(value.META_KEYSPACE, b.fullName)
					remoteValue.SetMetaField(value.META_PLAN, doc["plan"])
					remoteValue.SetMetaField(value.META_TXPLANS, doc["txPlans"])
					if planVer, ok := doc["planVersion"]; ok {
						if planVersion, ok := planVer.(int); ok {
							remoteValue.SetMetaField(value.META_PLAN_VERSION, int32(planVersion))
						}
						remoteValue.UnsetField("planVersion")
					}

					// Subquery plans
					if _, ok := doc["subqueryPlans"]; ok {
						remoteValue.SetMetaField(value.META_SUBQUERY_PLANS, doc["subqueryPlans"])
						remoteValue.UnsetField("subqueryPlans")
					}

					remoteValue.UnsetField("plan")
					remoteValue.UnsetField("txPlans")
					remoteValue.SetId(key)
					keysMap[key] = remoteValue
				},
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				}, creds, "", formData)
		} else {

			// local entry
			prepareds.PreparedDo(localKey, func(entry *prepareds.CacheEntry) {
				if userName != "" && !checkCacheEntry(entry, tenantName) {
					return
				}
				itemMap, txPlans := formatPrepared(entry, localKey, node, context)
				item := value.NewAnnotatedValue(itemMap)
				item.SetMetaField(value.META_KEYSPACE, b.fullName)
				if _, ok := itemMap["txPrepareds"]; ok {
					item.SetMetaField(value.META_TXPLANS, txPlans)
				}
				item.SetMetaField(value.META_PLAN, value.NewMarshalledValue(entry.Prepared.Operator))
				planVersion := entry.Prepared.PlanVersion()
				if planVersion >= util.MIN_PLAN_VERSION {
					item.SetMetaField(value.META_PLAN_VERSION, int32(planVersion))
				}

				// Subquery plans
				sqPlans := entry.Prepared.GetSubqueryPlansEntry()
				if len(sqPlans) > 0 {
					item.SetMetaField(value.META_SUBQUERY_PLANS, sqPlans)
				}

				item.SetId(key)
				keysMap[key] = item
			})
		}
	}
	return
}

func formatPrepared(entry *prepareds.CacheEntry, key string, node string, context datastore.QueryContext) (
	map[string]interface{}, map[string]interface{}) {

	itemMap := map[string]interface{}{
		"name":            entry.Prepared.Name(),
		"uses":            entry.Uses,
		"statement":       entry.Prepared.Text(),
		"indexApiVersion": entry.Prepared.IndexApiVersion(),
		"featuresControl": entry.Prepared.FeatureControls(),
	}
	if entry.Prepared.Namespace() != "" {
		itemMap["namespace"] = entry.Prepared.Namespace()
	}
	if entry.Prepared.QueryContext() != "" {
		itemMap["queryContext"] = entry.Prepared.QueryContext()
	}
	if entry.Prepared.EncodedPlan() != "" {
		itemMap["encoded_plan"] = entry.Prepared.EncodedPlan()
	}
	if entry.Prepared.OptimHints() != nil {
		itemMap["optimizer_hints"] = value.NewMarshalledValue(entry.Prepared.OptimHints())
	}
	if entry.Prepared.Persist() {
		itemMap["persist"] = entry.Prepared.Persist()
	}
	if entry.Prepared.AdHoc() {
		itemMap["adHocStatement"] = entry.Prepared.AdHoc()
	}
	if entry.Prepared.HasFatalError() {
		itemMap["verificationFatalError"] = entry.Prepared.HasFatalError()
	} else if entry.Prepared.ErrorCount() != 0 {
		itemMap["verificationErrorCount"] = entry.Prepared.ErrorCount()
	}
	if ksRefs := entry.Prepared.GetKeyspaceReferences(); len(ksRefs) > 0 {
		keyspaceRefs := make([]interface{}, len(ksRefs))
		for i := range ksRefs {
			keyspaceRefs[i] = ksRefs[i]
		}
		itemMap["keyspaceReferences"] = keyspaceRefs
	}

	isks := entry.Prepared.IndexScanKeyspaces()
	if len(isks) > 0 {
		itemMap["indexScanKeyspaces"] = isks
	}

	txPrepareds, txPlans := entry.Prepared.TxPrepared()
	if len(txPrepareds) > 0 {
		itemMap["txPrepareds"] = txPrepareds
	}

	if node != "" {
		itemMap["node"] = node
	}

	if !entry.Prepared.PreparedTime().IsZero() {
		itemMap["planPreparedTime"] = entry.Prepared.PreparedTime().Format(util.DEFAULT_FORMAT)
	}

	if entry.Prepared.Users() != "" {
		itemMap["users"] = entry.Prepared.Users()
	}
	if entry.Prepared.RemoteAddr() != "" {
		itemMap["remoteAddr"] = entry.Prepared.RemoteAddr()
	}
	if entry.Prepared.UserAgent() != "" {
		if entry.Prepared.Reprepared() {
			itemMap["creatingUserAgent"] = entry.Prepared.UserAgent()
		} else {
			itemMap["userAgent"] = entry.Prepared.UserAgent()
		}
	}
	// only give times for entries that have completed at least one execution
	if entry.Uses > 0 && entry.RequestTime > 0 {
		itemMap["lastUse"] = entry.LastUse.Format(util.DEFAULT_FORMAT)
		itemMap["avgElapsedTime"] = context.FormatDuration(time.Duration(entry.RequestTime) /
			time.Duration(entry.Uses))
		itemMap["avgServiceTime"] = context.FormatDuration(time.Duration(entry.ServiceTime) /
			time.Duration(entry.Uses))
		itemMap["minElapsedTime"] = context.FormatDuration(time.Duration(entry.MinRequestTime))
		itemMap["minServiceTime"] = context.FormatDuration(time.Duration(entry.MinServiceTime))
		itemMap["maxElapsedTime"] = context.FormatDuration(time.Duration(entry.MaxRequestTime))
		itemMap["maxServiceTime"] = context.FormatDuration(time.Duration(entry.MaxServiceTime))
	}
	return itemMap, txPlans
}

func (b *preparedsKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var err errors.Error
	var creds distributed.Creds
	var tenantName string

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		tenantName = getTenant(context.TenantCtx())
		creds = distributed.Creds(userName)
	}

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, pair := range deletes {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey,
				"prepareds", "DELETE", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", nil)

			// local entry
		} else {
			err = prepareds.DeletePreparedFunc(localKey, func(entry *prepareds.CacheEntry) bool {
				return userName == "" || checkCacheEntry(entry, tenantName)
			})
		}

		if err != nil {
			errs := errors.Errors{err}
			if preserveMutations {
				deleted := make([]value.Pair, i)
				if i > 0 {
					copy(deleted, deletes[0:i-1])
				}
				return i, deleted, errs
			} else {
				return i, nil, errs
			}
		}
	}

	if preserveMutations {
		return len(deletes), deletes, nil
	} else {
		return len(deletes), nil, nil
	}
}

func newPreparedsKeyspace(p *namespace) (*preparedsKeyspace, errors.Error) {
	b := new(preparedsKeyspace)
	setKeyspaceBase(&b.keyspaceBase, p, KEYSPACE_NAME_PREPAREDS)

	primary := &preparedsIndex{
		name:     PRIMARY_INDEX_NAME,
		keyspace: b,
		primary:  true,
	}
	b.indexer = newSystemIndexer(b, primary)
	setIndexBase(&primary.indexBase, b.indexer)

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
		setIndexBase(&nodes.indexBase, b.indexer)
		b.indexer.(*systemIndexer).AddIndex(nodes.name, nodes)
	} else {
		return nil, errors.NewSystemDatastoreError(err, "")
	}

	return b, nil
}

type preparedsIndex struct {
	indexBase
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
		var entry *datastore.IndexEntry
		var creds distributed.Creds
		var process func(name string, prepared *prepareds.CacheEntry) bool
		var send func() bool
		var doSend bool
		var tenantName string
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
			process = func(name string, prepared *prepareds.CacheEntry) bool {
				entry = &datastore.IndexEntry{
					PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
					EntryKey:   value.Values{value.NewValue(whoAmI)},
				}
				return true
			}
			send = func() bool {
				return sendSystemKey(conn, entry)
			}
		} else {
			creds = distributed.Creds(userName)
			tenantName = getTenant(conn.Context().TenantCtx())
			process = func(name string, prepared *prepareds.CacheEntry) bool {
				doSend = checkCacheEntry(prepared, tenantName)
				if doSend {
					entry = &datastore.IndexEntry{
						PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name),
						EntryKey:   value.Values{value.NewValue(whoAmI)},
					}
				}
				return true
			}
			send = func() bool {
				if doSend {
					return sendSystemKey(conn, entry)
				}
				return true
			}
		}
		idx := spanEvaluator.isEquals()
		if idx >= 0 {
			if spanEvaluator.key(idx) == whoAmI {
				prepareds.PreparedsForeach(process, send)
			} else {
				nodes := []string{decodeNodeName(spanEvaluator.key(idx))}
				distributed.RemoteAccess().GetRemoteKeys(nodes, "prepareds", func(id string) bool {
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
				}, creds, "")
			}
		} else {
			nodes := distributed.RemoteAccess().GetNodeNames()
			eligibleNodes := []string{}
			for _, node := range nodes {
				encodedNode := encodeNodeName(node)
				if spanEvaluator.evaluate(encodedNode) {
					if encodedNode == whoAmI {
						prepareds.PreparedsForeach(process, send)
					} else {
						eligibleNodes = append(eligibleNodes, node)
					}
				}
			}
			if len(eligibleNodes) > 0 {
				distributed.RemoteAccess().GetRemoteKeys(eligibleNodes, "prepareds", func(id string) bool {
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
				}, creds, "")
			}
		}
	}
}

func (pi *preparedsIndex) ScanEntries(requestId string, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
	var entry *datastore.IndexEntry
	var creds distributed.Creds
	var process func(name string, prepared *prepareds.CacheEntry) bool
	var send func() bool
	var doSend bool
	var tenantName string

	defer conn.Sender().Close()

	// now that the node name can change in flight, use a consistent one across the scan
	whoAmI := encodeNodeName(distributed.RemoteAccess().WhoAmI())

	userName := credsFromContext(conn.Context())
	if userName == "" {
		creds = distributed.NO_CREDS
		process = func(name string, prepared *prepareds.CacheEntry) bool {
			entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
			return true
		}
		send = func() bool {
			return sendSystemKey(conn, entry)
		}
	} else {
		creds = distributed.Creds(userName)
		tenantName = getTenant(conn.Context().TenantCtx())
		process = func(name string, prepared *prepareds.CacheEntry) bool {
			doSend = checkCacheEntry(prepared, tenantName)
			if doSend {
				entry = &datastore.IndexEntry{PrimaryKey: distributed.RemoteAccess().MakeKey(whoAmI, name)}
			}
			return true
		}
		send = func() bool {
			if doSend {
				return sendSystemKey(conn, entry)
			}
			return true
		}
	}
	prepareds.PreparedsForeach(process, send)
	distributed.RemoteAccess().GetRemoteKeys([]string{}, "prepareds", func(id string) bool {
		indexEntry := datastore.IndexEntry{PrimaryKey: id}
		return sendSystemKey(conn, &indexEntry)
	}, func(warn errors.Error) {
		if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
			conn.Warning(warn)
		}
	}, creds, "")
}

// TODO this needs to be amended when we support multiple tenants per each user
// if we decide to list all associated tenants and not just the current one
func getTenant(context tenant.Context) string {

	// if the user has no tenant associated, then we match no entry
	return tenant.Bucket(context)
}

func checkCacheEntry(entry *prepareds.CacheEntry, tenantName string) bool {
	return entry.Prepared.Tenant() == tenantName
}

func (b *preparedsKeyspace) Update(updates value.Pairs, context datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {

	var errs errors.Errors
	var creds distributed.Creds
	var tenantName string

	userName := credsFromContext(context)
	if userName == "" {
		creds = distributed.NO_CREDS
	} else {
		tenantName = getTenant(context.TenantCtx())
		creds = distributed.Creds(userName)
	}

	// now that the node name can change in flight, use a consistent one across deletes
	whoAmI := distributed.RemoteAccess().WhoAmI()
	for i, pair := range updates {
		name := pair.Name
		node, localKey := distributed.RemoteAccess().SplitKey(name)
		nodeName := decodeNodeName(node)

		// remote entry
		if len(nodeName) != 0 && nodeName != whoAmI {

			distributed.RemoteAccess().GetRemoteDoc(nodeName, localKey,
				"prepareds", "PATCH", nil,
				func(warn errors.Error) {
					if !warn.HasCause(errors.W_SYSTEM_REMOTE_NODE_NOT_FOUND) {
						context.Warning(warn)
					}
				},
				creds, "", nil)

			// local entry
		} else {
			prepareds.PreparedDo(localKey, func(ce *prepareds.CacheEntry) {
				if userName == "" || checkCacheEntry(ce, tenantName) {
					items, _ := formatPrepared(ce, localKey, node, context)
					for k, v := range pair.Value.Fields() {
						if cv, ok := items[k]; !ok || value.NewValue(v).Equals(value.NewValue(cv)) != value.TRUE_VALUE {
							errs = append(errs, errors.NewUpdateInvalidField(name, k))
						}
						delete(items, k)
					}
					if _, ok := items["planPreparedTime"]; ok {
						ce.Prepared.SetPreparedTime(time.Time{})
						delete(items, "planPreparedTime")
					}
					for k, _ := range items {
						errs = append(errs, errors.NewUpdateInvalidField(name, k))
					}
				}
			})
		}

		if len(errs) > 0 {
			if preserveMutations {
				updated := make([]value.Pair, i)
				if i > 0 {
					copy(updated, updates[0:i-1])
				}
				return i, updated, errs
			} else {
				return i, nil, errs
			}
		}
	}

	if preserveMutations {
		return len(updates), updates, nil
	} else {
		return len(updates), nil, nil
	}
}
