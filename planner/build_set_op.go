//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/plan"
)

func (this *builder) VisitUnion(node *algebra.Union) (interface{}, error) {
	// Inject DISTINCT into both terms
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevNode := this.node
	prevCover := this.cover

	defer func() {
		this.node = prevNode
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()

	this.node = node.First()
	this.cover = node.First()
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)

	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	compatible := false
	if this.useCBO {
		compatible = compatibleResultTerms(node.First(), node.Second())
		cost, cardinality, size, frCost = getUnionAllCost(first.(plan.Operator), second.(plan.Operator),
			compatible)
	}

	this.maxParallelism = 0
	unionAll := plan.NewUnionAll(cost, cardinality, size, frCost, first.(plan.Operator), second.(plan.Operator))
	if this.useCBO {
		cost, cardinality = getUnionDistinctCost(cost, cardinality,
			first.(plan.Operator), second.(plan.Operator), compatible)
	}
	return plan.NewSequence(unionAll, plan.NewDistinct(cost, cardinality, size, cost)), nil
}

func (this *builder) VisitUnionAll(node *algebra.UnionAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevNode := this.node
	prevCover := this.cover
	defer func() {
		this.node = prevNode
		this.cover = prevCover
	}()
	this.node = node.First()
	this.cover = node.First()

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)
	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getUnionAllCost(first.(plan.Operator), second.(plan.Operator),
			compatibleResultTerms(node.First(), node.Second()))
	}

	this.maxParallelism = 0
	return plan.NewUnionAll(cost, cardinality, size, frCost, first.(plan.Operator), second.(plan.Operator)), nil
}

func (this *builder) VisitIntersect(node *algebra.Intersect) (interface{}, error) {
	// Inject DISTINCT into first term
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevNode := this.node
	prevCover := this.cover
	defer func() {
		this.node = prevNode
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()

	this.node = node.First()
	this.cover = node.First()
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	// Do not inject DISTINCT into second term (done at run time)
	this.setOpDistinct = false

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)
	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getIntersectAllCost(first.(plan.Operator), second.(plan.Operator),
			compatibleResultTerms(node.First(), node.Second()))
	}

	this.maxParallelism = 0
	return plan.NewIntersectAll(first.(plan.Operator), second.(plan.Operator), true, cost, cardinality, size, frCost), nil
}

func (this *builder) VisitIntersectAll(node *algebra.IntersectAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevNode := this.node
	prevCover := this.cover
	defer func() {
		this.node = prevNode
		this.cover = prevCover
	}()
	this.node = node.First()
	this.cover = node.First()

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)
	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getIntersectAllCost(first.(plan.Operator), second.(plan.Operator),
			compatibleResultTerms(node.First(), node.Second()))
	}

	this.maxParallelism = 0
	return plan.NewIntersectAll(first.(plan.Operator), second.(plan.Operator), false, cost, cardinality, size, frCost), nil
}

func (this *builder) VisitExcept(node *algebra.Except) (interface{}, error) {
	// Inject DISTINCT into first term
	setOpDistinct := this.setOpDistinct
	this.setOpDistinct = true
	prevNode := this.node
	prevCover := this.cover
	defer func() {
		this.node = prevNode
		this.cover = prevCover
		this.setOpDistinct = setOpDistinct
	}()
	this.node = node.First()
	this.cover = node.First()

	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	// Do not inject DISTINCT into second term (done at run time)
	this.setOpDistinct = false

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)
	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getExceptAllCost(first.(plan.Operator), second.(plan.Operator),
			compatibleResultTerms(node.First(), node.Second()))
	}

	this.maxParallelism = 0
	return plan.NewExceptAll(first.(plan.Operator), second.(plan.Operator), true, cost, cardinality, size, frCost), nil
}

func (this *builder) VisitExceptAll(node *algebra.ExceptAll) (interface{}, error) {
	this.resetOrderOffsetLimit()
	this.delayProjection = false // Disable ORDER BY non-projected expressions

	prevNode := this.node
	prevCover := this.cover
	defer func() {
		this.node = prevNode
		this.cover = prevCover
	}()
	this.node = node.First()
	this.cover = node.First()

	first, ferr := node.First().Accept(this)
	if ferr != nil && !this.indexAdvisor {
		return nil, ferr
	}

	this.node = node.Second()
	this.cover = node.Second()
	second, serr := node.Second().Accept(this)
	if ferr != nil && this.indexAdvisor {
		return nil, ferr
	}
	if serr != nil {
		return nil, serr
	}

	cost := OPT_COST_NOT_AVAIL
	cardinality := OPT_CARD_NOT_AVAIL
	size := OPT_SIZE_NOT_AVAIL
	frCost := OPT_COST_NOT_AVAIL
	if this.useCBO {
		cost, cardinality, size, frCost = getExceptAllCost(first.(plan.Operator), second.(plan.Operator),
			compatibleResultTerms(node.First(), node.Second()))
	}

	this.maxParallelism = 0
	return plan.NewExceptAll(first.(plan.Operator), second.(plan.Operator), false, cost, cardinality, size, frCost), nil
}

/*
Checks whether the two result terms are compatible with each other,
i.e. could there be duplicates from different arms that can be eliminated
*/
func compatibleResultTerms(first, second algebra.Subresult) bool {
	firstTerms := first.ResultTerms()
	secondTerms := second.ResultTerms()

	if len(firstTerms) != len(secondTerms) || first.Raw() != second.Raw() {
		return false
	}

	// if both sides have Raw projections, then assume they are compatible
	if first.Raw() {
		return true
	}

	// if both sides not Raw, compare the alias of each term
	// since that's part of the result value
	for i := 0; i < len(firstTerms); i++ {
		if firstTerms[i].Alias() != secondTerms[i].Alias() || firstTerms[i].Star() != secondTerms[i].Star() {
			return false
		}
	}

	return true
}
