//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type Covers []*Cover

const (
	_COVER_NONE = int32(iota)
	_COVER_FULL
	_COVER_KEY
	_COVER_COND
)

/*
Internal Expression to support covering indexing.
*/
type Cover struct {
	ExpressionBase
	covered   Expression
	text      string
	coverType int32
}

func NewCover(covered Expression) *Cover {
	switch covered := covered.(type) {
	case *Cover:
		return covered
	}

	rv := &Cover{
		covered:   covered,
		text:      covered.String(),
		coverType: _COVER_FULL,
	}

	rv.expr = rv
	return rv
}

func NewIndexKey(covered Expression) *Cover {
	switch covered := covered.(type) {
	case *Cover:
		return covered
	}

	rv := &Cover{
		covered:   covered,
		text:      covered.String(),
		coverType: _COVER_KEY,
	}

	rv.expr = rv
	return rv
}

func NewIndexCondition(covered Expression) *Cover {
	switch covered := covered.(type) {
	case *Cover:
		return covered
	}

	rv := &Cover{
		covered:   covered,
		text:      covered.String(),
		coverType: _COVER_COND,
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Cover) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCover(this)
}

func (this *Cover) Type() value.Type {
	return this.covered.Type()
}

func (this *Cover) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	if item == nil {
		return nil, errors.NewNilEvaluateParamError("item")
	}
	switch item := item.(type) {
	case value.AnnotatedValue:
		rv = item.GetCover(this.text)
	}

	if rv == nil {
		return value.MISSING_VALUE, nil
	}

	return rv, nil
}

func (this *Cover) Value() value.Value {
	return this.covered.Value()
}

func (this *Cover) Static() Expression {
	return this.covered.Static()
}

func (this *Cover) StaticNoVariable() Expression {
	return this.covered.StaticNoVariable()
}

func (this *Cover) Alias() string {
	return this.covered.Alias()
}

func (this *Cover) Indexable() bool {
	return this.covered.Indexable()
}

func (this *Cover) PropagatesMissing() bool {
	return this.covered.PropagatesMissing()
}

func (this *Cover) PropagatesNull() bool {
	return this.covered.PropagatesNull()
}

func (this *Cover) EquivalentTo(other Expression) bool {
	if this.covered.EquivalentTo(other) {
		return true
	}

	oc, ok := other.(*Cover)
	return ok && this.covered.EquivalentTo(oc.covered) && this.coverType == oc.coverType
}

func (this *Cover) DependsOn(other Expression) bool {
	return this.covered.DependsOn(other)
}

func (this *Cover) CoveredBy(keyspace string, exprs Expressions, options CoveredOptions) Covered {
	return this.covered.CoveredBy(keyspace, exprs, options)
}

func (this *Cover) Children() Expressions {
	return Expressions{this.covered}
}

func (this *Cover) MapChildren(mapper Mapper) error {
	c, err := mapper.Map(this.covered)
	if err == nil && c != this.covered {
		this.covered = c
		this.text = c.String()
	}

	return err
}

func (this *Cover) Copy() Expression {
	var rv *Cover
	if this.coverType == _COVER_FULL {
		rv = NewCover(this.covered.Copy())
	} else if this.coverType == _COVER_KEY {
		rv = NewIndexKey(this.covered.Copy())
	} else if this.coverType == _COVER_COND {
		rv = NewIndexCondition(this.covered.Copy())
	}
	rv.BaseCopy(this)
	return rv
}

func (this *Cover) Covered() Expression {
	return this.covered
}

func (this *Cover) Text() string {
	return this.text
}

func (this *Cover) FullCover() bool {
	return this.coverType == _COVER_FULL
}

func (this *Cover) IsIndexKey() bool {
	return this.coverType == _COVER_KEY
}

func (this *Cover) IsIndexCond() bool {
	return this.coverType == _COVER_COND
}
