//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type Projection struct {
	distinct bool        `json:"distinct"`
	raw      bool        `json:"raw"`
	terms    ResultTerms `json:terms`
}

func NewProjection(distinct bool, terms ResultTerms) *Projection {
	rv := &Projection{
		distinct: distinct,
		raw:      false,
		terms:    terms,
	}

	return rv
}

func NewRawProjection(distinct bool, expr expression.Expression) *Projection {
	return &Projection{
		distinct: distinct,
		raw:      true,
		terms:    ResultTerms{NewResultTerm(expr, false, "")},
	}
}

func (this *Projection) Formalize(forbidden, allowed value.Value, keyspace string) (
	projection *Projection, err error) {
	projection = &Projection{
		distinct: this.distinct,
		raw:      this.raw,
		terms:    make(ResultTerms, len(this.terms)),
	}

	terms := projection.terms
	for i, term := range this.terms {
		terms[i] = &ResultTerm{
			star: term.star,
			as:   term.as,
		}

		if term.expr != nil {
			terms[i].expr, err = term.expr.Formalize(forbidden, allowed, keyspace)
			if err != nil {
				return nil, err
			}
		}
	}

	return
}

func (this *Projection) Distinct() bool {
	return this.distinct
}

func (this *Projection) Raw() bool {
	return this.raw
}

func (this *Projection) Terms() ResultTerms {
	return this.terms
}

type ResultTerms []*ResultTerm

type ResultTerm struct {
	expr expression.Expression `json:"expr"`
	star bool                  `json:"star"`
	as   string                `json:"as"`
}

func NewResultTerm(expr expression.Expression, star bool, as string) *ResultTerm {
	return &ResultTerm{
		expr: expr,
		star: star,
		as:   as,
	}
}

func (this *ResultTerm) Expression() expression.Expression {
	return this.expr
}

func (this *ResultTerm) Star() bool {
	return this.star
}

func (this *ResultTerm) As() string {
	return this.as
}

func (this *ResultTerm) Alias() string {
	if this.star {
		return ""
	} else if this.as != "" {
		return this.as
	} else {
		return this.expr.Alias()
	}
}
