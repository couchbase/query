//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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

func (this *Range) EquivalentTo(other *Range) bool {
	return this.Inclusion == other.Inclusion &&
		expression.Equivalents(this.Low, other.Low) &&
		expression.Equivalents(this.High, other.High)
}

func (this *Range) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Range) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"Inclusion": this.Inclusion,
	}

	if this.Low != nil {
		r["Low"] = this.Low
	}

	if this.High != nil {
		r["High"] = this.High
	}

	if f != nil {
		f(r)
	}
	return r
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

func (this *Span) EquivalentTo(other *Span) bool {
	return this.Exact == other.Exact &&
		expression.Equivalents(this.Seek, other.Seek) &&
		this.Range.EquivalentTo(&other.Range)
}

func (this *Span) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Span) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"Range": &this.Range,
	}

	if this.Seek != nil && isNotNull(this.Seek) {
		r["Seek"] = this.Seek
	}

	if this.Exact {
		r["Exact"] = this.Exact
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Span) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Seek  []string
		Range *Range
		Exact bool
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

	this.Exact = _unmarshalled.Exact

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
