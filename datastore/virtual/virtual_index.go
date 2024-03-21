//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package virtual

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

// Implement Index{} interface
type VirtualIndex struct {
	keyspace     datastore.Keyspace
	name         string
	primary      bool
	condition    expression.Expression
	indexKeys    expression.Expressions
	desc         []bool
	lkMissing    bool
	partnExpr    expression.Expressions //partition key expressions
	storageMode  datastore.IndexStorageMode
	storageStats []map[datastore.IndexStatType]value.Value
}

func NewVirtualIndex(keyspace datastore.Keyspace, name string, condition expression.Expression,
	indexKeys expression.Expressions, desc []bool, partnExpr expression.Expressions, isPrimary, lkMissing bool,
	sm datastore.IndexStorageMode, storageStats []map[datastore.IndexStatType]value.Value) datastore.Index {
	rv := &VirtualIndex{
		keyspace:  keyspace,
		name:      name,
		primary:   isPrimary,
		condition: expression.Copy(condition),
		indexKeys: expression.CopyExpressions(indexKeys),
		desc:      desc,
		lkMissing: lkMissing,
		partnExpr: expression.CopyExpressions(partnExpr),
	}

	if sm != "" {
		rv.storageMode = sm
	}
	if len(storageStats) > 0 {
		rv.storageStats = storageStats
	}

	return rv
}

func (this *VirtualIndex) BucketId() string {
	scope := this.keyspace.Scope()
	if scope == nil {
		return ""
	}
	return scope.BucketId()
}

func (this *VirtualIndex) ScopeId() string {
	return this.keyspace.ScopeId()
}

func (this *VirtualIndex) KeyspaceId() string {
	return this.keyspace.Id()
}

func (this *VirtualIndex) Id() string {
	return this.Name()
}

func (this *VirtualIndex) Name() string {
	return this.name
}

func (this *VirtualIndex) Type() datastore.IndexType {
	return datastore.VIRTUAL
}

// Virtual index may be in virtualindexer for virtual keyspace or normal keyspace indexer.
func (this *VirtualIndex) Indexer() datastore.Indexer {
	indexer, err := this.keyspace.Indexer(datastore.DEFAULT)
	if err == nil {
		return indexer
	}
	return nil
}

func (this *VirtualIndex) SeekKey() expression.Expressions {
	return nil
}

func (this *VirtualIndex) RangeKey() expression.Expressions {
	if this != nil {
		return this.indexKeys
	}
	return nil
}

func (this *VirtualIndex) Condition() expression.Expression {
	if this != nil {
		return this.condition
	}
	return nil
}

func (this *VirtualIndex) IsPrimary() bool {
	return this.primary
}

func (this *VirtualIndex) State() (state datastore.IndexState, msg string, err errors.Error) {
	return datastore.ONLINE, "", nil
}

func (this *VirtualIndex) Statistics(requestId string, span *datastore.Span) (datastore.Statistics, errors.Error) {
	return nil, errors.NewVirtualIdxNotImplementedError(nil, "Statistics for virtual index")
}

func (this *VirtualIndex) Drop(requestId string) errors.Error {
	return errors.NewVirtualIdxNotSupportedError(nil, "DROP for virtual index")
}

func (this *VirtualIndex) Scan(requestId string, span *datastore.Span, distinct bool, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
}

// Implement CountIndex{} interface
func (this *VirtualIndex) Count(span *datastore.Span, cons datastore.ScanConsistency, vector timestamp.Vector) (
	int64, errors.Error) {

	return 0, nil
}

// Implement Index2{} interface
func (this *VirtualIndex) RangeKey2() datastore.IndexKeys {
	if this != nil && this.indexKeys != nil {
		rangeKeys := make(datastore.IndexKeys, 0, len(this.indexKeys))
		for i, expr := range this.indexKeys {
			rangeKey := &datastore.IndexKey{
				Expr: expr,
			}
			if this.desc != nil && this.desc[i] {
				rangeKey.SetAttribute(datastore.IK_DESC, true)
			}
			if i == 0 && this.lkMissing {
				rangeKey.SetAttribute(datastore.IK_MISSING, true)
			}
			rangeKeys = append(rangeKeys, rangeKey)
		}
		return rangeKeys
	}
	return nil
}

func (this *VirtualIndex) Scan2(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection,
	ordered bool, projection *datastore.IndexProjection, offset, limit int64, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
}

// Impelment CountIndex2 interface
func (this *VirtualIndex) Count2(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency,
	vector timestamp.Vector) (int64, errors.Error) {

	return 0, nil
}

func (this *VirtualIndex) CanCountDistinct() bool {
	return true
}

func (this *VirtualIndex) CountDistinct(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency,
	vector timestamp.Vector) (int64, errors.Error) {

	return 0, nil
}

// Implement Index3{} interface
func (this *VirtualIndex) CreateAggregate(requestId string, groupAggs *datastore.IndexGroupAggregates,
	with value.Value) errors.Error {
	return errors.NewVirtualIdxNotSupportedError(nil, "CREATE AGGREGATE for virtual index")
}

func (this *VirtualIndex) DropAggregate(requestId, name string) errors.Error {
	return errors.NewVirtualIdxNotSupportedError(nil, "DROP AGGREGATE for virtual index")
}

func (this *VirtualIndex) Aggregates() ([]datastore.IndexGroupAggregates, errors.Error) {
	return nil, errors.NewVirtualIdxNotSupportedError(nil, "Precomputed Aggregates for virtual index")
}

func (this *VirtualIndex) PartitionKeys() (*datastore.IndexPartition, errors.Error) {
	if this == nil || len(this.partnExpr) == 0 {
		return nil, nil
	}

	keyPartition := &datastore.IndexPartition{
		Strategy: datastore.HASH_PARTITION,
		Exprs:    this.partnExpr.Copy(),
	}
	return keyPartition, nil
}

func (this *VirtualIndex) Scan3(requestId string, spans datastore.Spans2, reverse, distinctAfterProjection bool,
	projection *datastore.IndexProjection, offset, limit int64,
	groupAggs *datastore.IndexGroupAggregates, indexOrders datastore.IndexKeyOrders,
	cons datastore.ScanConsistency, vector timestamp.Vector, conn *datastore.IndexConnection) {
}

func (this *VirtualIndex) Alter(requestId string, with value.Value) (datastore.Index, errors.Error) {
	return nil, errors.NewVirtualIdxNotSupportedError(nil, "Alter for virtual index")
}

// Implement PrimaryIndex3{} interface
func (this *VirtualIndex) ScanEntries3(requestId string, projection *datastore.IndexProjection, offset, limit int64,
	groupAggs *datastore.IndexGroupAggregates, indexOrders datastore.IndexKeyOrders, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
}

//Implement Index4 interface

func (this *VirtualIndex) StorageMode() (datastore.IndexStorageMode, errors.Error) {
	if this.isCBOEnabledMode() {
		return this.storageMode, nil
	}
	return datastore.INDEX_MODE_VIRTUAL, nil
}

func (this *VirtualIndex) LeadKeyHistogram(requestId string) (*datastore.Histogram, errors.Error) {
	return nil, errors.NewVirtualIdxNotImplementedError(nil, "Index4 LeadKeyHistogram")
}

func (this *VirtualIndex) StorageStatistics(requestid string) ([]map[datastore.IndexStatType]value.Value, errors.Error) {
	if this.storageStats == nil {
		return nil, errors.NewVirtualIdxNotSupportedError(nil, "Storage Statistics")
	}
	return this.storageStats, nil
}

func (this *VirtualIndex) isCBOEnabledMode() bool {
	return this.storageMode == datastore.INDEX_MODE_PLASMA || this.storageMode == datastore.INDEX_MODE_MOI
}
