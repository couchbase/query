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

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

const (
	RANGE_CHECK_SPECIAL_SPAN = 1 << iota
	RANGE_SELF_SPAN
	RANGE_FULL_SPAN
	RANGE_WHOLE_SPAN
	RANGE_VALUED_SPAN
	RANGE_EMPTY_SPAN
	RANGE_MISSING_SPAN
	RANGE_NULL_SPAN
	RANGE_DERIVED_FROM_LIKE
	RANGE_FROM_IN_EXPR
	RANGE_ARRAY_ANY
	RANGE_ARRAY_ANY_EVERY
	RANGE_DEFAULT_LIKE
	RANGE_NOT_VALUED_SPAN
)

const RANGE_SPECIAL_SPAN = (RANGE_SELF_SPAN | RANGE_FULL_SPAN | RANGE_WHOLE_SPAN | RANGE_VALUED_SPAN | RANGE_EMPTY_SPAN |
	RANGE_MISSING_SPAN | RANGE_NULL_SPAN | RANGE_NOT_VALUED_SPAN)

type Ranges2 []*Range2

type Range2 struct {
	Low       expression.Expression
	High      expression.Expression
	Inclusion datastore.Inclusion
	Selec1    float64
	Selec2    float64
	Flags     uint32
	IndexKey  string
}

func NewRange2(low, high expression.Expression, incl datastore.Inclusion, selec1, selec2 float64, flags uint32) *Range2 {
	return &Range2{
		Low:       low,
		High:      high,
		Inclusion: incl,
		Selec1:    selec1,
		Selec2:    selec2,
		Flags:     flags,
	}
}

func (this *Range2) Copy() *Range2 {
	return &Range2{
		Low:       expression.Copy(this.Low),
		High:      expression.Copy(this.High),
		Inclusion: this.Inclusion,
		Selec1:    this.Selec1,
		Selec2:    this.Selec2,
		Flags:     this.Flags,
	}
}

func (this *Range2) EquivalentTo(other *Range2) bool {
	return this == other || (this.Inclusion == other.Inclusion &&
		expression.Equivalent(this.Low, other.Low) &&
		expression.Equivalent(this.High, other.High))
}

func (this *Range2) EqualRange() bool {
	return ((this.Inclusion == datastore.BOTH) &&
		(this.Low != nil && this.High != nil && (this.Low == this.High || this.Low.EquivalentTo(this.High)))) ||
		this.IsMissingRange()
}

func (this *Range2) IsMissingRange() bool {
	return this.Inclusion == datastore.NEITHER && this.Low == nil &&
		expression.Equivalent(this.High, expression.NULL_EXPR)
}

func (this *Range2) HasCheckSpecialSpan() bool {
	return (this.Flags & RANGE_CHECK_SPECIAL_SPAN) != 0
}

func (this *Range2) SetCheckSpecialSpan() {
	this.Flags |= RANGE_CHECK_SPECIAL_SPAN
}

func (this *Range2) UnsetCheckSpecialSpan() {
	this.Flags &^= RANGE_CHECK_SPECIAL_SPAN
}

func (this *Range2) HasSpecialSpan() bool {
	return (this.Flags & RANGE_SPECIAL_SPAN) != 0
}

func (this *Range2) ClearSpecialSpan() {
	this.Flags &^= RANGE_SPECIAL_SPAN
}

func (this *Range2) HasFlag(flag uint32) bool {
	return (this.Flags & flag) != 0
}

func (this *Range2) SetFlag(flag uint32) {
	this.Flags |= flag
}

// inherit any flags that's not a special span flag
func (this *Range2) InheritFlags(other *Range2) {
	this.Flags |= (other.Flags &^ RANGE_SPECIAL_SPAN)
}

func (this *Range2) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Range2) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"inclusion": this.Inclusion,
	}

	if this.IsDynamicIn() {
		r["dynamic_in"] = true
	}

	if this.Low != nil {
		r["low"] = this.Low
	}

	if this.High != nil {
		r["high"] = this.High
	}

	if len(this.IndexKey) > 0 {
		r["index_key"] = this.IndexKey
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
		DynamicIn bool                `json:"dynamic_in"`
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

	if _unmarshalled.DynamicIn {
		if low, ok1 := this.Low.(*expression.ArrayMin); ok1 {
			if high, ok2 := this.High.(*expression.ArrayMax); ok2 {
				low.Operand().SetExprFlag(expression.EXPR_DYNAMIC_IN)
				high.Operand().SetExprFlag(expression.EXPR_DYNAMIC_IN)
			}
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

func (this *Range2) IsDynamicIn() bool {
	var op1, op2 expression.Expression
	if low, ok1 := this.Low.(*expression.ArrayMin); ok1 && low.Operand().HasExprFlag(expression.EXPR_DYNAMIC_IN) {
		op1 = low.Operand()
	}
	if high, ok2 := this.High.(*expression.ArrayMax); ok2 && high.Operand().HasExprFlag(expression.EXPR_DYNAMIC_IN) {
		op2 = high.Operand()
	}
	if op1 != nil && op2 != nil && op1.EquivalentTo(op2) {
		return true
	}

	return false
}

func (this *Range2) GetDynamicInExpr() expression.Expression {
	if this.IsDynamicIn() {
		return this.Low.(*expression.ArrayMin).Operand()
	}
	return nil
}

type Spans2 []*Span2

type Span2 struct {
	Seek   expression.Expressions
	Ranges Ranges2
	Exact  bool
	Static bool
}

func NewSpan2(seek expression.Expressions, ranges Ranges2, exact bool) *Span2 {
	return &Span2{
		Seek:   seek,
		Ranges: ranges,
		Exact:  exact,
	}
}

func NewStaticSpan2(seek expression.Expressions, ranges Ranges2, exact bool) *Span2 {
	rv := NewSpan2(seek, ranges, exact)
	rv.Static = true
	return rv
}

func (this *Span2) Copy() *Span2 {
	return &Span2{
		Seek:   expression.CopyExpressions(this.Seek),
		Ranges: this.Ranges.Copy(),
		Exact:  this.Exact,
		Static: false,
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
		r["seek"] = this.Seek
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

func (this Spans2) HasStatic() bool {
	for _, sp := range this {
		if sp.Static {
			return true
		}
	}
	return false
}

func (this Spans2) HasDynamicIn() bool {
	for _, sp := range this {
		for _, rg := range sp.Ranges {
			if rg.IsDynamicIn() {
				return true
			}
		}
	}

	return false
}

func setRangeIndexKey(spans Spans2, index datastore.Index) {
	keys := index.RangeKey()
	flattenKeys := make(expression.Expressions, 0, len(keys))
	for _, expr := range keys {
		if all, ok := expr.(*expression.All); ok && all.Flatten() {
			for _, fk := range all.FlattenKeys().Operands() {
				flattenKeys = append(flattenKeys, fk)
			}
		} else {
			flattenKeys = append(flattenKeys, expr)
		}
	}

	for n, s := range spans {
		// duplicate static spans so we can update with the information-only field
		if s.Static {
			s = s.Copy()
			spans[n] = s
		}
		for i, r := range s.Ranges {
			if i >= len(flattenKeys) {
				break
			}
			r.IndexKey = flattenKeys[i].String()
		}
	}
}
