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

// bucketScan is common to scans that read a bucket.
type bucketScan struct {
	term *algebra.BucketTerm
}

func (this *bucketScan) Term() *algebra.BucketTerm {
	return this.term
}

func (this *bucketScan) Alias() string {
	return this.term.Alias()
}

type FullScan struct {
	bucketScan
	bucket catalog.Bucket
}

func NewFullScan(term *algebra.BucketTerm, bucket catalog.Bucket) *FullScan {
	return &FullScan{bucketScan{term}, bucket}
}

func (this *FullScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFullScan(this)
}

type EqualScan struct {
	bucketScan
	index  catalog.EqualIndex
	equals algebra.ExpressionList
}

func NewEqualScan(term *algebra.BucketTerm, index catalog.EqualIndex, equals algebra.ExpressionList) *EqualScan {
	return &EqualScan{bucketScan{term}, index, equals}
}

func (this *EqualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitEqualScan(this)
}

type RangeScan struct {
	bucketScan
	index  catalog.RangeIndex
	ranges RangeList
}

type Range struct {
	Low       algebra.ExpressionList
	High      algebra.ExpressionList
	Inclusion catalog.RangeInclusion
}

type RangeList []*Range

func NewRangeScan(term *algebra.BucketTerm, index catalog.RangeIndex, ranges RangeList) *RangeScan {
	return &RangeScan{bucketScan{term}, index, ranges}
}

func (this *RangeScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitRangeScan(this)
}

type DualScan struct {
	bucketScan
	index catalog.DualIndex
	duals DualList
}

type Dual struct {
	equal  algebra.Expression
	ranges RangeList
}

type DualList []*Dual

func NewDualScan(term *algebra.BucketTerm, index catalog.DualIndex, duals DualList) *DualScan {
	return &DualScan{bucketScan{term}, index, duals}
}

func (this *DualScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDualScan(this)
}

// KeyScan is used for KEYS clauses (except after JOIN / NEST).
type KeyScan struct {
	bucketScan
	bucket catalog.Bucket
}

func NewKeyScan(term *algebra.BucketTerm, bucket catalog.Bucket) *KeyScan {
	return &KeyScan{bucketScan{term}, bucket}
}

func (this *KeyScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyScan(this)
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
