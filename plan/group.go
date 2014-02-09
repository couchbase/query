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
	initials   algebra.InitialAggregates
}

func NewInitialGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *InitialGroup {
	rv := &InitialGroup{
		keys:       keys,
		aggregates: aggregates,
	}

	rv.initials = make(algebra.InitialAggregates, len(aggregates))
	for i, agg := range aggregates {
		rv.initials[i] = agg.Initial()
	}

	return rv
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

// Grouping of groups. Recursable and parallelizable.
type IntermediateGroup struct {
	keys          algebra.Expressions
	aggregates    algebra.Aggregates
	intermediates algebra.IntermediateAggregates
}

func NewIntermediateGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *IntermediateGroup {
	rv := &IntermediateGroup{
		keys:       keys,
		aggregates: aggregates,
	}

	rv.intermediates = make(algebra.IntermediateAggregates, len(aggregates))
	for i, agg := range aggregates {
		rv.intermediates[i] = agg.Intermediate()
	}

	return rv
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

// Final grouping and aggregation.
type FinalGroup struct {
	keys       algebra.Expressions
	aggregates algebra.Aggregates
	finals     algebra.FinalAggregates
}

func NewFinalGroup(keys algebra.Expressions, aggregates algebra.Aggregates) *FinalGroup {
	rv := &FinalGroup{
		keys:       keys,
		aggregates: aggregates,
	}

	rv.finals = make(algebra.FinalAggregates, len(aggregates))
	for i, agg := range aggregates {
		rv.finals[i] = agg.Final()
	}

	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}
