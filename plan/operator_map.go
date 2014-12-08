//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import "github.com/couchbaselabs/query/algebra"

// Operators is a global map of all plan.Operator implementations
// It is used by implementations of json.Unmarshal to access the
// correct implementation given the name of an implementation via
// the "#operator" key in a marshalled object.
var Operators_map map[string]Operator

func init() {
	Operators_map = map[string]Operator{
		"Alias":              NewAlias(""),
		"Channel":            NewChannel(),
		"Collect":            NewCollect(),
		"Delete":             NewSendDelete(nil),
		"Discard":            NewDiscard(),
		"Distinct":           NewDistinct(),
		"ExceptAll":          NewExceptAll(nil, nil),
		"Explain":            NewExplain(nil),
		"Fetch":              NewFetch(nil, nil),
		"Filter":             NewFilter(nil),
		"InitialGroup":       NewInitialGroup(nil, nil),
		"IntermediateGroup":  NewIntermediateGroup(nil, nil),
		"FinalGroup":         NewFinalGroup(nil, nil),
		"CreatePrimaryIndex": NewCreatePrimaryIndex(nil, nil),
		"CreateIndex":        NewCreateIndex(nil, nil),
		"DropIndex":          NewDropIndex(nil, nil),
		"AlterIndex":         NewAlterIndex(nil, nil),
		"Insert":             NewSendInsert(nil, nil),
		"IntersectAll":       NewIntersectAll(nil, nil),
		"Join": NewJoin(nil, algebra.NewJoin(nil, false,
			algebra.NewKeyspaceTerm("", "", nil, "", nil))),
		"Nest": NewJoin(nil, algebra.NewJoin(nil, false,
			algebra.NewKeyspaceTerm("", "", nil, "", nil))),
		"Unnest":         NewUnnest(algebra.NewUnnest(nil, false, nil, "alias")),
		"Let":            NewLet(nil),
		"Merge":          NewMerge(nil, nil, nil, nil, nil, nil),
		"Order":          NewOrder(algebra.NewOrder([]*algebra.SortTerm{})),
		"Offset":         NewOffset(nil),
		"Limit":          NewLimit(nil),
		"Parallel":       NewParallel(nil),
		"Prepare":        NewPrepare(nil),
		"InitialProject": NewInitialProject(algebra.NewProjection(false, []*algebra.ResultTerm{})),
		"FinalProject":   NewFinalProject(),
		"PrimaryScan":    NewPrimaryScan(nil, nil),
		"IndexScan":      NewIndexScan(nil, nil, nil, false, 0),
		"KeyScan":        NewKeyScan(nil),
		"ParentScan":     NewParentScan(),
		"ValueScan":      NewValueScan(nil),
		"CountScan":      NewCountScan(nil, nil),
		"DummyScan":      NewDummyScan(),
		"IntersectScan":  NewIntersectScan(),
		"Sequence":       NewSequence(),
		"Stream":         NewStream(),
		"UnionAll":       NewUnionAll(),
		"Clone":          NewClone(),
		"Set":            NewSet(nil),
		"Unset":          NewUnset(nil),
		"SendUpdate":     NewSendUpdate(nil, ""),
		"SendUpsert":     NewSendUpsert(nil, nil),
	}
}
