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
)

// Grouping of input data. Parallelizable.
type InitialGroup struct {
	keys       algebra.Expressions
	aggregates algebra.Aggregates
}

func NewInitialGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *InitialGroup {
	return &InitialGroup{
		keys:       keys,
		aggregates: aggregates,
	}
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

func (this *InitialGroup) Keys() algebra.Expressions {
	return this.keys
}

func (this *InitialGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}

// Grouping of groups. Recursable and parallelizable.
type IntermediateGroup struct {
	keys       algebra.Expressions
	aggregates algebra.Aggregates
}

func NewIntermediateGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *IntermediateGroup {
	return &IntermediateGroup{
		keys:       keys,
		aggregates: aggregates,
	}
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) Keys() algebra.Expressions {
	return this.keys
}

func (this *IntermediateGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}

// Final grouping and aggregation.
type FinalGroup struct {
	keys       algebra.Expressions
	aggregates algebra.Aggregates
}

func NewFinalGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *FinalGroup {
	return &FinalGroup{
		keys:       keys,
		aggregates: aggregates,
	}
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) Keys() algebra.Expressions {
	return this.keys
}

func (this *FinalGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}
