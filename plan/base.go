//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

type readonly struct {
}

func (this *readonly) Readonly() bool {
	return true
}

func (this *readonly) verify(prepared *Prepared) bool {
	return true
}

type readwrite struct {
}

func (this *readwrite) Readonly() bool {
	return false
}

func (this *readwrite) verify(prepared *Prepared) bool {
	return true
}

// optimizer estimates
type optEstimate struct {
	cost        float64
	cardinality float64
}

func (this *optEstimate) Cost() float64 {
	return this.cost
}

func (this *optEstimate) Cardinality() float64 {
	return this.cardinality
}

func setOptEstimate(oe *optEstimate, cost, cardinality float64) {
	oe.cost = cost
	oe.cardinality = cardinality
}

// represents DML statements, all are read-write
type dml struct {
	readwrite
}

// represents DDL statements, all are read-write, and currently have no cost/cardinality
type ddl struct {
	readwrite
}

func (this *ddl) Cost() float64 {
	return PLAN_COST_NOT_AVAIL
}

func (this *ddl) Cardinality() float64 {
	return PLAN_CARD_NOT_AVAIL
}

// represents legacy operators, all are read-only, and have no cost/cardinality
type legacy struct {
	readonly
}

func (this *legacy) Cost() float64 {
	return PLAN_COST_NOT_AVAIL
}

func (this *legacy) Cardinality() float64 {
	return PLAN_CARD_NOT_AVAIL
}

// represents operators used in execution only, all are read-only, and have no cost/cardinality
type execution struct {
	readonly
}

func (this *execution) Cost() float64 {
	return PLAN_COST_NOT_AVAIL
}

func (this *execution) Cardinality() float64 {
	return PLAN_CARD_NOT_AVAIL
}
