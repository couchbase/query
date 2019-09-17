//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plannerbase

type IndexCost struct {
	cost        float64
	cardinality float64
	selectivity float64
}

func NewIndexCost(cost, cardinality, selectivity float64) *IndexCost {
	return &IndexCost{
		cost:        cost,
		cardinality: cardinality,
		selectivity: selectivity,
	}
}

func (this *IndexCost) Copy() *IndexCost {
	return &IndexCost{
		cost:        this.cost,
		cardinality: this.cardinality,
		selectivity: this.selectivity,
	}
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
