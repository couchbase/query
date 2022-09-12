//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"fmt"
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
	// Scan
	"PrimaryScan":             &PrimaryScan{},
	"PrimaryScan3":            &PrimaryScan3{},
	"IndexScan":               &IndexScan{},
	"IndexScan2":              &IndexScan2{},
	"IndexScan3":              &IndexScan3{},
	"KeyScan":                 &KeyScan{},
	"ValueScan":               &ValueScan{},
	"DummyScan":               &DummyScan{},
	"CountScan":               &CountScan{},
	"IndexCountScan":          &IndexCountScan{},
	"IndexCountScan2":         &IndexCountScan2{},
	"IndexCountDistinctScan2": &IndexCountDistinctScan2{},
	"IntersectScan":           &IntersectScan{},
	"OrderedIntersectScan":    &OrderedIntersectScan{},
	"UnionScan":               &UnionScan{},
	"DistinctScan":            &DistinctScan{},
	"ExpressionScan":          &ExpressionScan{},

	// IndexFtsSearch
	"IndexFtsSearch": &IndexFtsSearch{},

	// Fetch
	"Fetch":      &Fetch{},
	"DummyFetch": &DummyFetch{},

	// Join
	"Join":           &Join{},
	"IndexJoin":      &IndexJoin{},
	"NestedLoopJoin": &NLJoin{},
	"HashJoin":       &HashJoin{},
	"Nest":           &Nest{},
	"IndexNest":      &IndexNest{},
	"NestedLoopNest": &NLNest{},
	"HashNest":       &HashNest{},
	"Unnest":         &Unnest{},

	// Let + Letting
	"Let": &Let{},

	// With
	"With": &With{},

	// Infer
	"InferKeyspace":   &InferKeyspace{},
	"InferExpression": &InferExpression{},

	// Update Statistics
	"UpdateStatistics": &UpdateStatistics{},

	// Filter
	"Filter": &Filter{},

	// Group
	"InitialGroup":      &InitialGroup{},
	"IntermediateGroup": &IntermediateGroup{},
	"FinalGroup":        &FinalGroup{},

	// Window functions
	"WindowAggregate": &WindowAggregate{},

	// Project
	"InitialProject": &InitialProject{},

	// TODO retire
	"FinalProject":      &FinalProject{},
	"IndexCountProject": &IndexCountProject{},

	// Distinct
	"Distinct": &Distinct{},

	// All
	"All": &All{},

	// Set operators
	"UnionAll":     &UnionAll{},
	"IntersectAll": &IntersectAll{},
	"ExceptAll":    &ExceptAll{},

	// Order
	"Order": &Order{},

	// Paging
	"Offset": &Offset{},
	"Limit":  &Limit{},

	// Insert
	"SendInsert": &SendInsert{},

	// Upsert
	"SendUpsert": &SendUpsert{},

	// Delete
	"SendDelete": &SendDelete{},

	// Update
	"Clone":      &Clone{},
	"Set":        &Set{},
	"Unset":      &Unset{},
	"SendUpdate": &SendUpdate{},

	// Merge
	"Merge": &Merge{},

	// Framework
	"Alias":     &Alias{},
	"Authorize": &Authorize{},
	"Parallel":  &Parallel{},
	"Sequence":  &Sequence{},
	"Discard":   &Discard{},
	"Stream":    &Stream{},
	"Collect":   &Collect{},
	"Receive":   &Receive{},

	// Index DDL
	"CreatePrimaryIndex": &CreatePrimaryIndex{},
	"CreateIndex":        &CreateIndex{},
	"DropIndex":          &DropIndex{},
	"AlterIndex":         &AlterIndex{},
	"BuildIndexes":       &BuildIndexes{},

	// Roles
	"GrantRole":  &GrantRole{},
	"RevokeRole": &RevokeRole{},

	// Explain
	"Explain": &Explain{},

	// Explain Function
	"ExplainFunction": &ExplainFunction{},

	// Prepare
	"Prepare": &Prepare{},

	// Scopes
	"CreateScope": &CreateScope{},
	"DropScope":   &DropScope{},

	// Collections
	"CreateCollection": &CreateCollection{},
	"DropCollection":   &DropCollection{},
	"FlushCollection":  &FlushCollection{},

	// Functions
	"CreateFunction":  &CreateFunction{},
	"DropFunction":    &DropFunction{},
	"ExecuteFunction": &ExecuteFunction{},

	// Index Advisor
	"Advise":      &Advise{},
	"IndexAdvice": &IndexAdvice{},

	// Transactions
	"StartTransaction":     &StartTransaction{},
	"CommitTransaction":    &CommitTransaction{},
	"RollbackTransaction":  &RollbackTransaction{},
	"TransactionIsolation": &TransactionIsolation{},
	"Savepoint":            &Savepoint{},

	// Sequences
	"CreateSequence": &CreateSequence{},
	"AlterSequence":  &AlterSequence{},
	"DropSequence":   &DropSequence{},
}
