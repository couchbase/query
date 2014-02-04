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
	_ "fmt"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/catalog"
)

type Range struct {
	Low       algebra.ExpressionList
	High      algebra.ExpressionList
	Inclusion catalog.RangeInclusion
}

type RangeList []*Range

type Dual struct {
	equal  algebra.Expression
	ranges RangeList
}

type DualList []*Dual

type FullScan struct {
	bucket catalog.Bucket
}

// For subqueries.
type ParentScan struct {
	project algebra.Path
	as      string
}

type EqualScan struct {
	index  catalog.EqualIndex
	equals algebra.ExpressionList
}

type RangeScan struct {
	index  catalog.RangeIndex
	ranges RangeList
}

type DualScan struct {
	index catalog.DualIndex
	duals DualList
}

type KeyScan struct {
	bucket catalog.Bucket
	keys   algebra.Expression
}

type ValueScan struct {
	values algebra.Expression
}

// Generates a single empty object. Used if there is no FROM clause.
type DummyScan struct {
}

func NewFullScan(bucket catalog.Bucket) *FullScan {
	return &FullScan{bucket}
}

func (this *FullScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFullScan(this)
}

func NewParentScan(project algebra.Path, as string) *ParentScan {
	return &ParentScan{project, as}
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func NewEqualScan(index catalog.EqualIndex, equals algebra.ExpressionList) *EqualScan {
	return &EqualScan{index, equals}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

func NewRangeScan(index catalog.RangeIndex, ranges RangeList) *RangeScan {
	return &RangeScan{index, ranges}
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

func NewDualScan(index catalog.DualIndex, duals DualList) *DualScan {
	return &DualScan{index, duals}
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

func NewKeyScan(bucket catalog.Bucket, keys algebra.Expression) *KeyScan {
	return &KeyScan{bucket, keys}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
}

func NewValueScan(values algebra.Expression) *ValueScan {
	return &ValueScan{values}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func NewDummyScan() *DummyScan {
	return &DummyScan{}
}

func (this *DummyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDummyScan(this)
}
