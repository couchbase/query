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
	"fmt"
	"strings"
)

// Helper function to create a specific operator given its name
// (used as a key by GetOperator) and body in raw bytes
func MakeOperator(name string, body []byte) (Operator, error) {
	which_op, has_op := GetOperator(name)

	if !has_op {
		return nil, fmt.Errorf("No operator for name %s", name)
	}

	new_op := which_op.New()
	err := new_op.UnmarshalJSON(body)

	return new_op, err
}

// GetOperator exposes the operators map to other packages
func GetOperator(name string) (Operator, bool) {
	rv, ok := _OPERATORS[name]
	return rv, ok
}

// _OPERATORS is a global map of all plan.Operator implementations
// It is used by implementations of json.Unmarshal to access the
// correct implementation given the name of an implementation via
// the "#operator" key in a marshalled object.
var _OPERATORS = map[string]Operator{
	"Alias":              &Alias{},
	"Channel":            &Channel{},
	"Collect":            &Collect{},
	"Delete":             &SendDelete{},
	"Discard":            &Discard{},
	"Distinct":           &Distinct{},
	"ExceptAll":          &ExceptAll{},
	"Explain":            &Explain{},
	"Fetch":              &Fetch{},
	"Filter":             &Filter{},
	"InitialGroup":       &InitialGroup{},
	"IntermediateGroup":  &IntermediateGroup{},
	"FinalGroup":         &FinalGroup{},
	"CreatePrimaryIndex": &CreatePrimaryIndex{},
	"CreateIndex":        &CreateIndex{},
	"DropIndex":          &DropIndex{},
	"AlterIndex":         &AlterIndex{},
	"Insert":             &SendInsert{},
	"IntersectAll":       &IntersectAll{},
	"Join":               &Join{},
	"Nest":               &Nest{},
	"Unnest":             &Unnest{},
	"Let":                &Let{},
	"Merge":              &Merge{},
	"Order":              &Order{},
	"Offset":             &Offset{},
	"Limit":              &Limit{},
	"Parallel":           &Parallel{},
	"Prepare":            &Prepare{},
	"InitialProject":     &InitialProject{},
	"FinalProject":       &FinalProject{},
	"PrimaryScan":        &PrimaryScan{},
	"IndexScan":          &IndexScan{},
	"KeyScan":            &KeyScan{},
	"ParentScan":         &ParentScan{},
	"ValueScan":          &ValueScan{},
	"CountScan":          &CountScan{},
	"DummyScan":          &DummyScan{},
	"IntersectScan":      &IntersectScan{},
	"Sequence":           &Sequence{},
	"Stream":             &Stream{},
	"UnionAll":           &UnionAll{},
	"Clone":              &Clone{},
	"Set":                &Set{},
	"Unset":              &Unset{},
	"SendUpdate":         &SendUpdate{},
	"SendUpsert":         &SendUpsert{},
}
