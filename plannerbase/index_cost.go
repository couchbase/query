//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plannerbase

import (
	"github.com/couchbase/query/datastore"
)

type IdxProperty uint32

const (
	IDX_PD_ORDER IdxProperty = 1 << iota
	IDX_EARLY_ORDER
	IDX_EXACTSPANS
)

type IndexCost struct {
	index       datastore.Index
	cost        float64
	cardinality float64
	selectivity float64
	size        int64
	frCost      float64
	idxProperty IdxProperty
	skipKeys    []bool
}

func NewIndexCost(index datastore.Index, cost, cardinality, selectivity float64,
	size int64, frCost float64, skipKeys []bool) *IndexCost {

	return &IndexCost{
		index:       index,
		cost:        cost,
		cardinality: cardinality,
		selectivity: selectivity,
		size:        size,
		frCost:      frCost,
		skipKeys:    skipKeys,
	}
}

func (this *IndexCost) Copy() *IndexCost {
	rv := &IndexCost{
		index:       this.index,
		cost:        this.cost,
		cardinality: this.cardinality,
		selectivity: this.selectivity,
		size:        this.size,
		frCost:      this.frCost,
		idxProperty: this.idxProperty,
	}
	rv.skipKeys = make([]bool, len(this.skipKeys))
	copy(rv.skipKeys, this.skipKeys)
	return rv
}

func (this *IndexCost) Index() datastore.Index {
	return this.index
}

func (this *IndexCost) Cost() float64 {
	return this.cost
}

func (this *IndexCost) Cardinality() float64 {
	return this.cardinality
}

func (this *IndexCost) Selectivity() float64 {
	return this.selectivity
}

func (this *IndexCost) Size() int64 {
	return this.size
}

func (this *IndexCost) FrCost() float64 {
	return this.frCost
}

func (this *IndexCost) SetCost(cost float64) {
	this.cost = cost
}

func (this *IndexCost) SetCardinality(cardinality float64) {
	this.cardinality = cardinality
}

func (this *IndexCost) SetSelectivity(selectivity float64) {
	this.selectivity = selectivity
}

func (this *IndexCost) HasPdOrder() bool {
	return (this.idxProperty & IDX_PD_ORDER) != 0
}

func (this *IndexCost) SetPdOrder() {
	this.idxProperty |= IDX_PD_ORDER
}

func (this *IndexCost) HasEarlyOrder() bool {
	return (this.idxProperty & IDX_EARLY_ORDER) != 0
}

func (this *IndexCost) SetEarlyOrder() {
	this.idxProperty |= IDX_EARLY_ORDER
}

func (this *IndexCost) HasExactSpans() bool {
	return (this.idxProperty & IDX_EXACTSPANS) != 0
}

func (this *IndexCost) SetExactSpans() {
	this.idxProperty |= IDX_EXACTSPANS
}

func (this *IndexCost) SkipKeys() []bool {
	return this.skipKeys
}

func (this *IndexCost) HasSkipKey(i int) bool {
	return (i < len(this.skipKeys)) && this.skipKeys[i]
}

func (this *IndexCost) SetSkipKey(i int) {
	if i < len(this.skipKeys) {
		this.skipKeys[i] = true
	}
}
