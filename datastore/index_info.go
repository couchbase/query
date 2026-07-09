//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"encoding/json"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
)

// IndexInfoDoc builds the descriptive document for a single index -- the same fields
// system:indexes and the INDEXINFO() function report (id, name, index key, using, state,
// condition, partitioning, WITH options, and metadata/stats). scopeId/bucketId may be ""
// for a keyspace with no owning scope/bucket path (pre-collections style).
func IndexInfoDoc(index Index, keyspaceId, scopeId, bucketId, namespaceName, namespaceId, datastoreId string) (
	map[string]interface{}, errors.Error) {

	state, msg, err := index.State()
	if err != nil {
		return nil, err
	}

	doc := map[string]interface{}{
		"id":           index.Id(),
		"name":         index.Name(),
		"keyspace_id":  keyspaceId,
		"namespace":    namespaceName,
		"namespace_id": namespaceId,
		"datastore_id": datastoreId,
		"index_key":    jsonSafe(indexKeyStrings(index)),
		"using":        jsonSafe(index.Type()),
		"state":        string(state),
	}

	if scopeId != "" {
		doc["scope_id"] = scopeId
	}
	if bucketId != "" {
		doc["bucket_id"] = bucketId
	}
	if msg != "" {
		doc["message"] = msg
	}
	if cond := index.Condition(); cond != nil {
		doc["condition"] = cond.String()
	}
	if index.IsPrimary() {
		doc["is_primary"] = true
	}
	if partition := indexPartitionString(index); partition != "" {
		doc["partition"] = partition
	}
	if ixm, ok := index.(interface{ IndexMetadata() map[string]interface{} }); ok {
		if md, ok := jsonSafe(ixm.IndexMetadata()).(map[string]interface{}); ok {
			doc["metadata"] = indexStats(md)
		}
	}
	if ixw, ok := index.(interface{ With() map[string]interface{} }); ok {
		if w, ok := jsonSafe(ixw.With()).(map[string]interface{}); ok {
			doc["with"] = w
		}
	}

	return doc, nil
}

func indexKeyStrings(index Index) []string {
	if index2, ok := index.(Index2); ok {
		keys := index2.RangeKey2()
		rv := make([]string, len(keys))
		for i, kp := range keys {
			stringer := expression.NewStringer()
			stringer.VisitShared(kp.Expr)
			if i == 0 && kp.HasAttribute(IK_MISSING) {
				stringer.WriteString(" INCLUDE MISSING")
			}
			if kp.HasAttribute(IK_DESC) {
				stringer.WriteString(" DESC")
			}
			if kp.HasAttribute(IK_DENSE_VECTOR) {
				stringer.WriteString(" DENSE VECTOR")
			} else if kp.HasAttribute(IK_SPARSE_VECTOR) {
				stringer.WriteString(" SPARSE VECTOR")
			} else if kp.HasAttribute(IK_MULTI_VECTOR) {
				stringer.WriteString(" MULTI VECTOR")
			}
			rv[i] = stringer.String()
		}
		return rv
	}

	rangeKey := index.RangeKey()
	rv := make([]string, len(rangeKey))
	for i, kp := range rangeKey {
		rv[i] = kp.String()
	}
	return rv
}

func indexPartitionString(index Index) string {
	index3, ok := index.(Index3)
	if !ok {
		return ""
	}
	partition, _ := index3.PartitionKeys()
	if partition == nil || partition.Strategy == NO_PARTITION {
		return ""
	}

	stringer := expression.NewStringer()
	stringer.WriteString(string(partition.Strategy))
	stringer.WriteString("(")
	for i, expr := range partition.Exprs {
		if i > 0 {
			stringer.WriteString(",")
		}
		stringer.VisitShared(expr)
	}
	stringer.WriteString(")")
	return stringer.String()
}

func jsonSafe(obj interface{}) interface{} {
	var rv interface{}
	if bytes, err := json.Marshal(obj); err == nil {
		json.Unmarshal(bytes, &rv)
	}
	return rv
}

// indexStats adds a last_scan_time field derived from the raw metadata's stats.
func indexStats(m map[string]interface{}) map[string]interface{} {
	if _, ok := m["last_scan_time"]; ok {
		return m
	}
	m["last_scan_time"] = nil
	stats, ok := m["stats"].(map[string]interface{})
	if !ok {
		return m
	}
	lastKnownScanTime, ok := stats["last_known_scan_time"].(float64)
	if !ok || lastKnownScanTime == 0 {
		return m
	}
	m["last_scan_time"] = time.UnixMicro(int64(lastKnownScanTime) / 1000).Format(util.DEFAULT_FORMAT)
	return m
}
