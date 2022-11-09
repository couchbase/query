//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type InitialProject struct {
	readonly
	optEstimate
	projection      *algebra.Projection
	terms           ProjectTerms
	starTermCount   int
	discardOriginal bool
}

func NewInitialProject(projection *algebra.Projection, cost, cardinality float64,
	size int64, frCost float64, discardOriginal bool) *InitialProject {

	results := projection.Terms()
	terms := make(ProjectTerms, len(results))

	rv := &InitialProject{
		projection:      projection,
		terms:           terms,
		discardOriginal: discardOriginal,
	}

	for i, res := range results {
		terms[i] = &ProjectTerm{
			result: res,
		}

		if res.Star() {
			rv.starTermCount++
		}
	}

	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
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

func (this *InitialProject) StarTermCount() int {
	return this.starTermCount
}

func (this *InitialProject) DiscardOriginal() bool {
	return this.discardOriginal
}

func (this *InitialProject) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *InitialProject) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "InitialProject"}

	if this.projection.Distinct() {
		r["distinct"] = this.projection.Distinct()
	}

	if this.projection.Raw() {
		r["raw"] = this.projection.Raw()
	}

	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
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
	if this.discardOriginal {
		r["discard_original"] = this.discardOriginal
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *InitialProject) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Terms []*struct {
			Expr string `json:"expr"`
			As   string `json:"as"`
			Star bool   `json:"star"`
		} `json:"result_terms"`
		Distinct    bool                   `json:"distinct"`
		Raw         bool                   `json:"raw"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		DiscardOrig bool                   `json:"discard_original"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms := make(algebra.ResultTerms, len(_unmarshalled.Terms))
	for i, term_data := range _unmarshalled.Terms {
		var expr expression.Expression
		if term_data.Expr != "" {
			expr, err = parser.Parse(term_data.Expr)
			if err != nil {
				return err
			}
		}
		terms[i] = algebra.NewResultTerm(expr, term_data.Star, term_data.As)
	}
	projection := algebra.NewProjection(_unmarshalled.Distinct, terms)
	projection.SetRaw(_unmarshalled.Raw)
	results := projection.Terms()
	project_terms := make(ProjectTerms, len(results))

	for i, res := range results {
		project_terms[i] = &ProjectTerm{
			result: res,
		}
	}

	this.projection = projection
	this.terms = project_terms
	this.discardOriginal = _unmarshalled.DiscardOrig

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

// Final projection operator is left for backwards compatibility with older versions
// (just in case prepared plans on mixed versions clusters come in from older engines)
// TODO It will be retired after mad hatter goes out of support
type FinalProject struct {
	legacy
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
	return json.Marshal(this.MarshalBase(nil))
}

func (this *FinalProject) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "FinalProject"}
	if f != nil {
		f(r)
	}
	return r
}

func (this *FinalProject) UnmarshalJSON([]byte) error {
	// NOP: FinalProject has no data structure
	return nil
}

type IndexCountProject struct {
	legacy
	projection *algebra.Projection
	terms      ProjectTerms
}

func NewIndexCountProject(projection *algebra.Projection) *IndexCountProject {
	results := projection.Terms()
	terms := make(ProjectTerms, len(results))

	for i, res := range results {
		terms[i] = &ProjectTerm{
			result: res,
		}
	}

	return &IndexCountProject{
		projection: projection,
		terms:      terms,
	}
}

func (this *IndexCountProject) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountProject(this)
}

func (this *IndexCountProject) New() Operator {
	return &IndexCountProject{}
}

func (this *IndexCountProject) Projection() *algebra.Projection {
	return this.projection
}

func (this *IndexCountProject) Terms() ProjectTerms {
	return this.terms
}

func (this *IndexCountProject) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexCountProject) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IndexCountProject"}

	if this.projection.Raw() {
		r["raw"] = this.projection.Raw()
	}

	s := make([]interface{}, 0, len(this.terms))
	for _, term := range this.terms {
		t := make(map[string]interface{})

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

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexCountProject) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Terms []*struct {
			Expr string `json:"expr"`
			As   string `json:"as"`
		} `json:"result_terms"`
		Raw bool `json:"raw"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	terms := make(algebra.ResultTerms, len(_unmarshalled.Terms))
	for i, term_data := range _unmarshalled.Terms {
		var expr expression.Expression
		if term_data.Expr != "" {
			expr, err = parser.Parse(term_data.Expr)
			if err != nil {
				return err
			}
		}
		terms[i] = algebra.NewResultTerm(expr, false, term_data.As)
	}
	projection := algebra.NewProjection(false, terms)
	projection.SetRaw(_unmarshalled.Raw)
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

type ProjectTerms []*ProjectTerm

type ProjectTerm struct {
	result *algebra.ResultTerm
}

func (this *ProjectTerm) Result() *algebra.ResultTerm {
	return this.result
}
