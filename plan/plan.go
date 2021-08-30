//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

/*
Package plan provides query plans.
*/
package plan

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

const REPREPARE_CHECK uint64 = math.MaxUint64

type Operators []Operator

type Operator interface {
	json.Marshaler   // JSON encoding; used by EXPLAIN and PREPARE
	json.Unmarshaler // JSON decoding: used by EXECUTE

	MarshalBase(f func(map[string]interface{})) map[string]interface{} // JSON encoding helper for execution

	Accept(visitor Visitor) (interface{}, error) // Visitor pattern
	Readonly() bool                              // Used to determine read-only compliance
	New() Operator                               // Dynamic constructor; used for unmarshaling

	verify(prepared *Prepared) bool // Check that the operator can reference keyspaces and indexes

	Cost() float64
	Cardinality() float64
	Size() int64
	FrCost() float64
}

type CoveringOperator interface {
	Operator

	Covers() expression.Covers
	FilterCovers() map[*expression.Cover]value.Value
	Covering() bool
	SetCovers(covers expression.Covers)

	GroupAggs() *IndexGroupAggregates
}

type SecondaryScan interface {
	CoveringOperator
	fmt.Stringer

	Limit() expression.Expression
	SetLimit(limit expression.Expression)

	Offset() expression.Expression
	SetOffset(offset expression.Expression)

	CoverJoinSpanExpressions(coverer *expression.Coverer) error

	IsUnderNL() bool

	OrderTerms() IndexKeyOrders

	// get index pointer if single index used, nil if multiple indexes used
	GetIndex() datastore.Index
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
