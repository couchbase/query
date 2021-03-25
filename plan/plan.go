//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
