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

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
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

func (this *InitialProject) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_ string `json:"#operator"`
		//Terms    []json.RawMessage `json:"result_terms"`
		Terms []struct {
			Expr string `json:"expr"`
			As   string `json:"as"`
			Star bool   `json:"star"`
		} `json:"result_terms"`
		Distinct bool `json:"distinct"`
		Raw      bool `json:"raw"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms := make(algebra.ResultTerms, len(_unmarshalled.Terms))
	for i, term_data := range _unmarshalled.Terms {
		/*var term_data struct {
			Expr string `json:"expr"`
			As   string `json:"as"`
			Star bool   `json:"star"`
		}
		err := json.Unmarshal(raw_term, &term_data)
		if err != nil {
			return err
		}
		*/
		expr, err := parser.Parse(term_data.Expr)
		if err != nil {
			return err
		}

		terms[i] = algebra.NewResultTerm(expr, term_data.Star, term_data.As)
	}
	projection := algebra.NewProjection(_unmarshalled.Distinct, terms)
	results := projection.Terms()
	project_terms := make(ProjectTerms, len(results))

	for i, res := range results {
		project_terms[i] = &ProjectTerm{
			result: res,
		}
	}

	this.projection = projection
	this.terms = project_terms

	return nil
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

func (this *FinalProject) UnmarshalJSON([]byte) error {
	// NOP: FinalProject has no data structure
	return nil
}

type ProjectTerms []*ProjectTerm

type ProjectTerm struct {
	result *algebra.ResultTerm
}

func (this *ProjectTerm) Result() *algebra.ResultTerm {
	return this.result
}
