//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

type readonly struct {
	planContext *planContext
}

func (this *readonly) Readonly() bool {
	return true
}

func (this *readonly) verify(prepared *Prepared) errors.Error {
	return nil
}

func (this *readonly) keyspaceReferences(prepared *Prepared) {
}

func (this *readonly) SetImplicitArrayKey(arrayKey *expression.All) {
}

func (this *readonly) ImplicitArrayKey() *expression.All {
	return nil
}

func (this *readonly) PlanContext() *planContext {
	return this.planContext
}

func (this *readonly) SetPlanContext(planContext *planContext) {
	this.planContext = planContext
}

type readwrite struct {
	planContext *planContext
}

func (this *readwrite) Readonly() bool {
	return false
}

func (this *readwrite) verify(prepared *Prepared) errors.Error {
	return nil
}

func (this *readwrite) keyspaceReferences(prepared *Prepared) {
}

func (this *readwrite) PlanContext() *planContext {
	return this.planContext
}

func (this *readwrite) SetPlanContext(planContext *planContext) {
	this.planContext = planContext
}

// optimizer estimates
type optEstimate struct {
	cost        float64
	cardinality float64
	size        int64   // cumulative document size
	frCost      float64 // first-result cost
}

func (this *optEstimate) Cost() float64 {
	return this.cost
}

func (this *optEstimate) Cardinality() float64 {
	return this.cardinality
}

func (this *optEstimate) Size() int64 {
	return this.size
}

func (this *optEstimate) FrCost() float64 {
	return this.frCost
}

func setOptEstimate(oe *optEstimate, cost, cardinality float64, size int64, frCost float64) {
	oe.cost = cost
	oe.cardinality = cardinality
	oe.size = size
	oe.frCost = frCost
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

func (this *ddl) Size() int64 {
	return PLAN_SIZE_NOT_AVAIL
}

func (this *ddl) FrCost() float64 {
	return PLAN_COST_NOT_AVAIL
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

func (this *legacy) Size() int64 {
	return PLAN_SIZE_NOT_AVAIL
}

func (this *legacy) FrCost() float64 {
	return PLAN_COST_NOT_AVAIL
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

func (this *execution) Size() int64 {
	return PLAN_SIZE_NOT_AVAIL
}

func (this *execution) FrCost() float64 {
	return PLAN_COST_NOT_AVAIL
}
