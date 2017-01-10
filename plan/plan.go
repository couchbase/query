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

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

type Operators []Operator

type Operator interface {
	json.Marshaler   // JSON encoding; used by EXPLAIN and PREPARE
	json.Unmarshaler // JSON decoding: used by EXECUTE

	MarshalBase(f func(map[string]interface{})) map[string]interface{} // JSON encoding helper for execution

	Accept(visitor Visitor) (interface{}, error) // Visitor pattern
	Readonly() bool                              // Used to determine read-only compliance
	New() Operator                               // Dynamic constructor; used for unmarshaling
}

type CoveringOperator interface {
	Operator

	Covers() expression.Covers
	FilterCovers() map[*expression.Cover]value.Value
	Covering() bool
}

type SecondaryScan interface {
	CoveringOperator
	fmt.Stringer
}
