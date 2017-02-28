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

type Ranges2 []*Range2

type Range2 struct {
	Low       expression.Expression
	High      expression.Expression
	Inclusion datastore.Inclusion
}

func NewRange2(low, high expression.Expression, incl datastore.Inclusion) *Range2 {
	return &Range2{
		Low:       low,
		High:      high,
		Inclusion: incl,
	}
}

func (this *Range2) Copy() *Range2 {
	return &Range2{
		Low:       expression.Copy(this.Low),
		High:      expression.Copy(this.High),
		Inclusion: this.Inclusion,
	}
}

func (this *Range2) EquivalentTo(other *Range2) bool {
	return this == other || (this.Inclusion == other.Inclusion &&
		expression.Equivalent(this.Low, other.Low) &&
		expression.Equivalent(this.High, other.High))
}

func (this *Range2) EqualRange() bool {
	return (this.Inclusion == datastore.BOTH) && (this.Low != nil && this.High != nil && (this.Low == this.High || this.Low.EquivalentTo(this.High)))
}

func (this *Range2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Range2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"inclusion": this.Inclusion,
	}

	if this.Low != nil {
		r["low"] = this.Low
	}

	if this.High != nil {
		r["high"] = this.High
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Range2) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Low       string              `json:"low"`
		High      string              `json:"high"`
		Inclusion datastore.Inclusion `json:"inclusion"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Low != "" {
		this.Low, err = parser.Parse(_unmarshalled.Low)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.High != "" {
		this.High, err = parser.Parse(_unmarshalled.High)
		if err != nil {
			return err
		}
	}

	this.Inclusion = _unmarshalled.Inclusion
	return nil
}

func (this Ranges2) Copy() Ranges2 {
	ranges := make(Ranges2, len(this))
	for i, r := range this {
		if r != nil {
			ranges[i] = r.Copy()
		}
	}

	return ranges
}

func (this Ranges2) EquivalentTo(other Ranges2) bool {

	if len(this) != len(other) {
		return false
	}

	for i := 0; i < len(this); i++ {
		if this[i] == other[i] {
			continue
		}

		if this[i] == nil || other[i] == nil {
			return false
		}

		if !this[i].EquivalentTo(other[i]) {
			return false
		}
	}

	return true
}

func (this *Range2) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

type Spans2 []*Span2

type Span2 struct {
	Seek   expression.Expressions
	Ranges Ranges2
	Exact  bool
}

func NewSpan2(seek expression.Expressions, ranges Ranges2, exact bool) *Span2 {
	return &Span2{
		Seek:   seek,
		Ranges: ranges,
		Exact:  exact,
	}
}

func (this *Span2) Copy() *Span2 {
	return &Span2{
		Seek:   expression.CopyExpressions(this.Seek),
		Ranges: this.Ranges.Copy(),
		Exact:  this.Exact,
	}
}

func (this *Span2) EquivalentTo(other *Span2) bool {
	return this == other || (this.Exact == other.Exact &&
		expression.Equivalents(this.Seek, other.Seek) &&
		this.Ranges.EquivalentTo(other.Ranges))
}

func (this *Span2) Empty() bool {
	for i := 0; i < len(this.Ranges); i++ {
		r := this.Ranges[i]
		if (r.Inclusion&datastore.BOTH) != datastore.BOTH && r.Low != nil {
			if r.Low == r.High || (r.High != nil && r.Low.EquivalentTo(r.High)) {
				return true
			}
		}

		if r.Low == nil || r.High == nil {
			continue
		}

		lv := r.Low.Value()
		hv := r.High.Value()
		if lv == nil || hv == nil {
			continue
		}

		c := lv.Collate(hv)
		if c == 0 {
			if (r.Inclusion & datastore.BOTH) != datastore.BOTH {
				return true
			}
			continue
		}
		return c > 0
	}

	return false
}

func (this *Span2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Span2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"range": this.Ranges,
	}

	if this.Seek != nil && isNotNull(this.Seek) {
		r["Seek"] = this.Seek
	}

	if this.Exact {
		r["exact"] = this.Exact
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *Span2) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Seek   []string          `json:"seek"`
		Ranges []json.RawMessage `json:"range"`
		Exact  bool              `json:"exact"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.Ranges = make(Ranges2, 0, len(_unmarshalled.Ranges))
	for _, s := range _unmarshalled.Ranges {
		r := &Range2{}
		err = r.UnmarshalJSON(s)
		if err != nil {
			return err
		}
		this.Ranges = append(this.Ranges, r)
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

func (this *Span2) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this Spans2) Copy() Spans2 {
	spans := make(Spans2, len(this))
	for i, s := range this {
		if s != nil {
			spans[i] = s.Copy()
		}
	}

	return spans
}

func (this Spans2) EquivalentTo(other Spans2) bool {

	if len(this) != len(other) {
		return false
	}

	for i := 0; i < len(this); i++ {
		if this[i] == other[i] {
			continue
		}

		if this[i] == nil || other[i] == nil {
			return false
		}

		if !this[i].EquivalentTo(other[i]) {
			return false
		}
	}

	return true
}
