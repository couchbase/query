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
	equals algebra.Expressions
}

func NewEqualScan(index catalog.EqualIndex, equals algebra.Expressions) *EqualScan {
	return &EqualScan{index, equals}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

func (this *EqualScan) Index() catalog.EqualIndex {
	return this.index
}

type RangeScan struct {
	index  catalog.RangeIndex
	ranges Ranges
}

type Ranges []*Range

type Range struct {
	Low       algebra.Expressions
	High      algebra.Expressions
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

type DualScan struct {
	index catalog.DualIndex
	duals Duals
}

type Duals []*Dual

type Dual struct {
	equal  algebra.Expression
	ranges Ranges
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

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	term *algebra.BucketTerm
}

func NewKeyScan(term *algebra.BucketTerm) *KeyScan {
	return &KeyScan{term}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func (this *KeyScan) Term() *algebra.BucketTerm {
	return this.term
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

// DummyScan is used for SELECTs with no FROM clause.
type DummyScan struct {
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

func NewDummyScan() *DummyScan {
	return &DummyScan{}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}
