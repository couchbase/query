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
)

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
		as: as,
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
