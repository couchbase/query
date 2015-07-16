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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Ranges []*Range

type Range struct {
	Low       expression.Expressions
	High      expression.Expressions
	Inclusion datastore.Inclusion
}

func (this *Range) Copy() *Range {
	return &Range{
		Low:       expression.CopyExpressions(this.Low),
		High:      expression.CopyExpressions(this.High),
		Inclusion: this.Inclusion,
	}
}

type Spans []*Span

type Span struct {
	Seek  expression.Expressions
	Range Range
}

func (this *Span) Copy() *Span {
	return &Span{
		Seek:  expression.CopyExpressions(this.Seek),
		Range: *(this.Range.Copy()),
	}
}

// Spans implements json.Unmarshaller to enable prepared statement execution
func (this Spans) UnmarshalJSON(body []byte) error {
	var _unmarshalled []*struct {
		Seek  []string
		Range struct {
			Low       []string
			High      []string
			Inclusion datastore.Inclusion
		}
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this = make(Spans, len(_unmarshalled))
	for i, span := range _unmarshalled {
		var s Span
		s.Seek = make(expression.Expressions, len(span.Seek))
		for j, seekExpr := range span.Seek {
			s.Seek[j], err = parser.Parse(seekExpr)
			if err != nil {
				return err
			}

			s.Range.Low = make(expression.Expressions, len(span.Range.Low))
			for l, lowExpr := range span.Range.Low {
				s.Range.Low[l], err = parser.Parse(lowExpr)
				if err != nil {
					return err
				}
			}

			s.Range.High = make(expression.Expressions, len(span.Range.High))
			for h, hiExpr := range span.Range.High {
				s.Range.Low[h], err = parser.Parse(hiExpr)
				if err != nil {
					return err
				}
			}

			s.Range.Inclusion = span.Range.Inclusion
		}

		this[i] = &s
	}

	return nil
}

func (this Spans) Copy() Spans {
	spans := make(Spans, len(this))
	for i, s := range this {
		if s != nil {
			spans[i] = s.Copy()
		}
	}

	return spans
}
