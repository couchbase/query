//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package virtual

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

//Implement Index{} interface
type VirtualIndex struct {
	keyspace  datastore.Keyspace
	name      string
	primary   bool
	condition expression.Expression
	indexKeys expression.Expressions
	desc      []bool
	partnExpr expression.Expressions //partition key expressions
}

func NewVirtualIndex(keyspace datastore.Keyspace, name string, condition expression.Expression, indexKeys expression.Expressions, desc []bool, partnExpr expression.Expressions, isPrimary bool) datastore.Index {
	return &VirtualIndex{
		keyspace:  keyspace,
		name:      name,
		primary:   isPrimary,
		condition: expression.Copy(condition),
		indexKeys: expression.CopyExpressions(indexKeys),
		desc:      desc,
		partnExpr: expression.CopyExpressions(partnExpr),
	}
}

func (this *VirtualIndex) BucketId() string {
	return ""
}

func (this *VirtualIndex) ScopeId() string {
	return ""
}

func (this *VirtualIndex) KeyspaceId() string {
	return this.keyspace.Name()
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

//Virtual index may be in virtualindexer for virtual keyspace or normal keyspace indexer.
func (this *VirtualIndex) Indexer() datastore.Indexer {
	indexer, err := this.keyspace.Indexer(this.Type())
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

//Implement CountIndex{} interface
func (this *VirtualIndex) Count(span *datastore.Span, cons datastore.ScanConsistency, vector timestamp.Vector) (int64, errors.Error) {
	return 0, nil
}

//Implement Index2{} interface
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

//Impelment CountIndex2 interface
func (this *VirtualIndex) Count2(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency, vector timestamp.Vector) (int64, errors.Error) {
	return 0, nil
}

func (this *VirtualIndex) CanCountDistinct() bool {
	return true
}

func (this *VirtualIndex) CountDistinct(requestId string, spans datastore.Spans2, cons datastore.ScanConsistency, vector timestamp.Vector) (int64, errors.Error) {
	return 0, nil
}

//Implement Index3{} interface
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

//Implement PrimaryIndex3{} interface
func (this *VirtualIndex) ScanEntries3(requestId string, projection *datastore.IndexProjection, offset, limit int64,
	groupAggs *datastore.IndexGroupAggregates, indexOrders datastore.IndexKeyOrders, cons datastore.ScanConsistency,
	vector timestamp.Vector, conn *datastore.IndexConnection) {
}

//Implement Index4 interface

func (this *VirtualIndex) StorageMode() (datastore.IndexStorageMode, errors.Error) {
	return datastore.INDEX_MODE_VIRTUAL, nil
}

func (this *VirtualIndex) LeadKeyHistogram(requestId string) (*datastore.Histogram, errors.Error) {
	return nil, errors.NewVirtualIdxNotImplementedError(nil, "Index4 LeadKeyHistogram")
}

func (this *VirtualIndex) StorageStatistics(requestid string) ([]map[datastore.IndexStatType]value.Value, errors.Error) {
	return nil, errors.NewVirtualIdxNotImplementedError(nil, "Storage Statistics")
}
