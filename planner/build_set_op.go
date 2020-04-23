//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitUnion(node *algebra.Union) (interface{}, error) {
	// Inject DISTINCT into both terms
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevCover := this.cover

	defer func() {
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()

	this.cover = node.First()
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	unionAll := plan.NewUnionAll(first.(plan.Operator), second.(plan.Operator))
	return plan.NewSequence(unionAll, plan.NewDistinct()), nil
}

func (this *builder) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevCover := this.cover
	defer func() {
		this.cover = prevCover
	}()
	this.cover = node.First()

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	return plan.NewUnionAll(first.(plan.Operator), second.(plan.Operator)), nil
}

func (this *builder) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	// Inject DISTINCT into first term
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevCover := this.cover
	defer func() {
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()

	this.cover = node.First()
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	// Do not inject DISTINCT into second term (done at run time)
	this.setOpDistinct = false

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	return plan.NewIntersectAll(first.(plan.Operator), second.(plan.Operator), true), nil
}

func (this *builder) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevCover := this.cover
	defer func() {
		this.cover = prevCover
	}()
	this.cover = node.First()

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	return plan.NewIntersectAll(first.(plan.Operator), second.(plan.Operator), false), nil
}

func (this *builder) VisitExcept(node *algebra.Except) (interface{}, error) {
	// Inject DISTINCT into first term
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevCover := this.cover
	defer func() {
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()
	this.cover = node.First()

	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	// Do not inject DISTINCT into second term (done at run time)
	this.setOpDistinct = false

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	return plan.NewExceptAll(first.(plan.Operator), second.(plan.Operator), true), nil
}

func (this *builder) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevCover := this.cover
	defer func() {
		this.cover = prevCover
	}()
	this.cover = node.First()

	first, err := node.First().Accept(this)
	if err != nil {
		return nil, err
	}

	this.cover = node.Second()
	second, err := node.Second().Accept(this)
	if err != nil {
		return nil, err
	}

	this.maxParallelism = 0
	return plan.NewExceptAll(first.(plan.Operator), second.(plan.Operator), false), nil
}
