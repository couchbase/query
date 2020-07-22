//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

import (
	"github.com/couchbase/query/datastore"
)

type IdxPushDown uint32

const (
	IDX_PD_ORDER IdxPushDown = 1 << iota
)

type IndexCost struct {
	index       datastore.Index
	cost        float64
	cardinality float64
	selectivity float64
	idxPushDown IdxPushDown
	skipKeys    []bool
}

func NewIndexCost(index datastore.Index, cost, cardinality, selectivity float64,
	skipKeys []bool) *IndexCost {

	return &IndexCost{
		index:       index,
		cost:        cost,
		cardinality: cardinality,
		selectivity: selectivity,
		skipKeys:    skipKeys,
	}
}

func (this *IndexCost) Copy() *IndexCost {
	rv := &IndexCost{
		index:       this.index,
		cost:        this.cost,
		cardinality: this.cardinality,
		selectivity: this.selectivity,
		idxPushDown: this.idxPushDown,
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

func (this *IndexCost) SetCost(cost float64) {
	this.cost = cost
}

func (this *IndexCost) SetCardinality(cardinality float64) {
	this.cardinality = cardinality
}

func (this *IndexCost) SetSelectivity(selectivity float64) {
	this.selectivity = selectivity
}

func (this *IndexCost) HasOrder() bool {
	return (this.idxPushDown & IDX_PD_ORDER) != 0
}

func (this *IndexCost) SetOrder() {
	this.idxPushDown |= IDX_PD_ORDER
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
