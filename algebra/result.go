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
	"strconv"

	"github.com/couchbaselabs/query/expression"
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

	a := 0
	for _, term := range terms {
		if !term.Star() && term.Alias() == "" {
			a++
			term.auto = "$" + strconv.Itoa(a)
		}
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
	auto string                `json:"auto"`
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
	} else if this.auto != "" {
		return this.auto
	} else {
		return this.expr.Alias()
	}
}
