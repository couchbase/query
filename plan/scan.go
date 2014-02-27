//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/expression"
)

type PrimaryScan struct {
	index catalog.PrimaryIndex
}

func NewPrimaryScan(index catalog.PrimaryIndex) *PrimaryScan {
	return &PrimaryScan{index}
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) Index() catalog.PrimaryIndex {
	return this.index
}

type IndexScan struct {
	index catalog.Index
	spans expression.Spans
}

func NewIndexScan(index catalog.Index, spans expression.Spans) *IndexScan {
	return &IndexScan{index, spans}
}

func (this *IndexScan) Index() catalog.Index {
	return this.index
}

func (this *IndexScan) Spans() expression.Spans {
	return this.spans
}

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	keys expression.Expression
}

func NewKeyScan(keys expression.Expression) *KeyScan {
	return &KeyScan{keys}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Keys() expression.Expression {
	return this.keys
}

// ParentScan is used for UNNEST subqueries.
type ParentScan struct {
}

func NewParentScan() *ParentScan {
	return &ParentScan{}
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

// ValueScan is used for VALUES clauses, e.g. in INSERTs.
type ValueScan struct {
	values expression.Expression
}

func NewValueScan(values expression.Expression) *ValueScan {
	return &ValueScan{values}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Values() expression.Expression {
	return this.values
}

// DummyScan is used for SELECTs with no FROM clause.
type DummyScan struct {
}

func NewDummyScan() *DummyScan {
	return &DummyScan{}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}

// CountScan is used for SELECT COUNT(*) with no WHERE clause.
type CountScan struct {
	bucket catalog.Bucket
}

func NewCountScan(bucket catalog.Bucket) *CountScan {
	return &CountScan{bucket}
}

func (this *CountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCountScan(this)
}

func (this *CountScan) Bucket() catalog.Bucket {
	return this.bucket
}

// MultipleScan scans multiple indexes and intersects the results.
type MultipleScan struct {
	scans []Operator
}

func NewMultipleScan(scans ...Operator) *MultipleScan {
	return &MultipleScan{scans}
}

func (this *MultipleScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMultipleScan(this)
}

func (this *MultipleScan) Scans() []Operator {
	return this.scans
}
