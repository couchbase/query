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
	"encoding/json"

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/expression"
)

type InitialProject struct {
	readonly
	projection *algebra.Projection
	terms      ProjectTerms
}

func NewInitialProject(projection *algebra.Projection) *InitialProject {
	results := projection.Terms()
	terms := make(ProjectTerms, len(results))

	for i, res := range results {
		terms[i] = &ProjectTerm{
			result: res,
		}
	}

	return &InitialProject{
		projection: projection,
		terms:      terms,
	}
}

func (this *InitialProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialProject(this)
}

func (this *InitialProject) New() Operator {
	return &InitialProject{}
}

func (this *InitialProject) Projection() *algebra.Projection {
	return this.projection
}

func (this *InitialProject) Terms() ProjectTerms {
	return this.terms
}

func (this *InitialProject) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "InitialProject"}

	if this.projection.Distinct() {
		r["distinct"] = this.projection.Distinct()
	}

	if this.projection.Raw() {
		r["raw"] = this.projection.Raw()
	}

	s := make([]interface{}, 0, len(this.terms))
	for _, term := range this.terms {
		t := make(map[string]interface{})

		if term.Result().Star() {
			t["star"] = term.Result().Star()
		}

		if term.Result().As() != "" {
			t["as"] = term.Result().As()
		}

		expr := term.Result().Expression()
		if expr != nil {
			t["expr"] = expression.NewStringer().Visit(expr)
		}

		s = append(s, t)
	}
	r["result_terms"] = s
	return json.Marshal(r)
}

type FinalProject struct {
	readonly
}

func NewFinalProject() *FinalProject {
	return &FinalProject{}
}

func (this *FinalProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalProject(this)
}

func (this *FinalProject) New() Operator {
	return &FinalProject{}
}

func (this *FinalProject) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "FinalProject"}
	return json.Marshal(r)
}

type ProjectTerms []*ProjectTerm

type ProjectTerm struct {
	result *algebra.ResultTerm
}

func (this *ProjectTerm) Result() *algebra.ResultTerm {
	return this.result
}
