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

	"github.com/couchbaselabs/query/algebra"
)

func Plan(node algebra.Node) (Operator, error) {
	planner := &Planner{}
	op, err := node.Accept(planner)

	if err != nil {
		return nil, err
	}

	switch op := op.(type) {
	case Operator:
		return op, nil
	default:
		panic(fmt.Sprintf("Expected plan.Operator instead of %T.", op))
	}
}

type Planner struct {
}

func (this *Planner) VisitSelect(node *algebra.Select) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitBucketTerm(node *algebra.BucketTerm) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitParentTerm(node *algebra.ParentTerm) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitJoin(node *algebra.Join) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitNest(node *algebra.Nest) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitUnnest(node *algebra.Unnest) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitInsert(node *algebra.Insert) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitDelete(node *algebra.Delete) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitUpdate(node *algebra.Update) (interface{}, error) {
	return nil, nil
}

func (this *Planner) VisitMerge(node *algebra.Merge) (interface{}, error) {
	return nil, nil
}
