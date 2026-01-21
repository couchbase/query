//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Package plan provides query plans.
*/
package plan

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

const REPREPARE_CHECK uint64 = math.MaxUint64

type QueryPlan struct {
	op         Operator
	subqueries map[*algebra.Select]Operator
	extraPrivs *auth.Privileges
}

func NewQueryPlan(op Operator) *QueryPlan {
	return &QueryPlan{
		op: op,
	}
}

func (this *QueryPlan) PlanOp() Operator {
	return this.op
}

func (this *QueryPlan) SetPlanOp(op Operator) {
	this.op = op
}

func (this *QueryPlan) AddSubquery(subq *algebra.Select, op Operator) {
	if this.subqueries == nil {
		this.subqueries = make(map[*algebra.Select]Operator, 8)
	}
	this.subqueries[subq] = op
}

func (this *QueryPlan) HasSubquery(subq *algebra.Select) bool {
	_, ok := this.subqueries[subq]
	return ok
}

func (this *QueryPlan) Subqueries() map[*algebra.Select]Operator {
	return this.subqueries
}

func (this *QueryPlan) Verify(prepared *Prepared) errors.Error {
	return this.op.verify(prepared)
}

func (this *QueryPlan) SetExtraPrivs(privs *auth.Privileges) {
	this.extraPrivs = privs
}

func (this *QueryPlan) Authorize(creds *auth.Credentials) errors.Error {
	if this.extraPrivs == nil || len(this.extraPrivs.List) == 0 {
		return nil
	}
	ds := datastore.GetDatastore()
	return ds.Authorize(this.extraPrivs, creds)
}

type Operators []Operator

type Operator interface {
	json.Marshaler   // JSON encoding; used by EXPLAIN and PREPARE
	json.Unmarshaler // JSON decoding: used by EXECUTE

	MarshalBase(f func(map[string]interface{})) map[string]interface{} // JSON encoding helper for execution

	Accept(visitor Visitor) (interface{}, error) // Visitor pattern
	Readonly() bool                              // Used to determine read-only compliance
	New() Operator                               // Dynamic constructor; used for unmarshaling

	verify(prepared *Prepared) errors.Error // Check that the operator can reference keyspaces and indexes
	keyspaceReferences(prepared *Prepared)  // Gather keyspace references

	Cost() float64
	Cardinality() float64
	Size() int64
	FrCost() float64

	PlanContext() *planContext
	SetPlanContext(planContext *planContext)
}

type CoveringOperator interface {
	Operator

	Covers() expression.Covers
	FilterCovers() map[*expression.Cover]value.Value
	Covering() bool
	SetCovers(covers expression.Covers)
	SetImplicitArrayKey(arrayKey *expression.All)
	ImplicitArrayKey() *expression.All

	GroupAggs() *IndexGroupAggregates
}

type SecondaryScan interface {
	CoveringOperator
	fmt.Stringer

	Limit() expression.Expression
	SetLimit(limit expression.Expression)

	Offset() expression.Expression
	SetOffset(offset expression.Expression)

	CoverJoinSpanExpressions(coverer *expression.Coverer,
		implicitArrayKey *expression.All) error

	IsUnderNL() bool

	OrderTerms() IndexKeyOrders

	// get index pointer if single index used, nil if multiple indexes used
	GetIndex() datastore.Index

	Equals(interface{}) bool
}

func CopyOperators(ops []Operator) []Operator {
	size := len(ops)
	if size < 16 {
		size = 16
	}
	newOps := make([]Operator, 0, size)
	for _, op := range ops {
		newOps = append(newOps, op)
	}
	return newOps
}

func CopyCoveringOperators(ops []CoveringOperator) []CoveringOperator {
	size := len(ops)
	if size < 8 {
		size = 8
	}
	newOps := make([]CoveringOperator, 0, size)
	for _, op := range ops {
		newOps = append(newOps, op)
	}
	return newOps
}
