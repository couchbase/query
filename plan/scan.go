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
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/catalog"
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

type EqualScan struct {
	index  catalog.EqualIndex
	equals algebra.CompositeExpressions
}

func NewEqualScan(index catalog.EqualIndex, equals algebra.CompositeExpressions) *EqualScan {
	return &EqualScan{index, equals}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

func (this *EqualScan) Index() catalog.EqualIndex {
	return this.index
}

func (this *EqualScan) Equals() algebra.CompositeExpressions {
	return this.equals
}

type RangeScan struct {
	index  catalog.RangeIndex
	ranges Ranges
}

type Ranges []*Range

type Range struct {
	Low       algebra.CompositeExpression
	High      algebra.CompositeExpression
	Inclusion catalog.RangeInclusion
}

func NewRangeScan(index catalog.RangeIndex, ranges Ranges) *RangeScan {
	return &RangeScan{index, ranges}
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

func (this *RangeScan) Index() catalog.RangeIndex {
	return this.index
}

func (this *RangeScan) Ranges() Ranges {
	return this.ranges
}

type DualScan struct {
	index catalog.DualIndex
	duals Duals
}

type Duals []*Dual

type Dual struct {
	Equal algebra.CompositeExpression
	Range
}

func NewDualScan(index catalog.DualIndex, duals Duals) *DualScan {
	return &DualScan{index, duals}
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

func (this *DualScan) Index() catalog.DualIndex {
	return this.index
}

func (this *DualScan) Duals() Duals {
	return this.duals
}

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	keys algebra.Expression
}

func NewKeyScan(keys algebra.Expression) *KeyScan {
	return &KeyScan{keys}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Keys() algebra.Expression {
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
	values algebra.Expression
}

func NewValueScan(values algebra.Expression) *ValueScan {
	return &ValueScan{values}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) Values() algebra.Expression {
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
