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
	"github.com/couchbase/query/value"
)

type Ranges []*Range

type Range struct {
	Low       expression.Expressions
	High      expression.Expressions
	Inclusion datastore.Inclusion
}

func isNotNull(e expression.Expressions) bool {
	for _, elem := range e {
		if elem.Value() != value.NULL_VALUE {
			return true
		}
	}
	return false
}

func (this *Range) Copy() *Range {
	return &Range{
		Low:       expression.CopyExpressions(this.Low),
		High:      expression.CopyExpressions(this.High),
		Inclusion: this.Inclusion,
	}
}

func (this *Range) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"Inclusion": this.Inclusion,
	}

	if this.Low != nil {
		r["Low"] = this.Low
	}

	if this.High != nil {
		r["High"] = this.High
	}

	return json.Marshal(r)
}

func (this *Range) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Low       []string
		High      []string
		Inclusion datastore.Inclusion
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Low != nil {
		this.Low = make(expression.Expressions, len(_unmarshalled.Low))
		for l, lowExpr := range _unmarshalled.Low {
			this.Low[l], err = parser.Parse(lowExpr)
			if err != nil {
				return err
			}
		}
	}

	if _unmarshalled.High != nil {
		this.High = make(expression.Expressions, len(_unmarshalled.High))
		for h, hiExpr := range _unmarshalled.High {
			this.High[h], err = parser.Parse(hiExpr)
			if err != nil {
				return err
			}
		}
	}

	this.Inclusion = _unmarshalled.Inclusion
	return nil
}

type Spans []*Span

type Span struct {
	Seek  expression.Expressions
	Range Range
	Exact bool
}

func (this *Span) Copy() *Span {
	return &Span{
		Seek:  expression.CopyExpressions(this.Seek),
		Range: *(this.Range.Copy()),
		Exact: this.Exact,
	}
}

func (this *Span) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{
		"Range": &this.Range,
	}

	if this.Seek != nil && isNotNull(this.Seek) {
		r["Seek"] = this.Seek
	}

	return json.Marshal(r)
}

func (this *Span) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Seek  []string
		Range *Range
	}

	_unmarshalled.Range = &this.Range

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Seek != nil {
		this.Seek = make(expression.Expressions, len(_unmarshalled.Seek))
		for j, seekExpr := range _unmarshalled.Seek {
			this.Seek[j], err = parser.Parse(seekExpr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (this *Span) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
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
