//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type Cover struct {
	ExpressionBase
	covered Expression
	text    string
}

func NewCover(covered Expression) *Cover {
	switch covered := covered.(type) {
	case *Cover:
		return covered
	}

	rv := &Cover{
		covered: covered,
		text:    covered.String(),
	}

	rv.expr = rv
	return rv
}

func (this *Cover) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCover(this)
}

func (this *Cover) Type() value.Type {
	return this.covered.Type()
}

func (this *Cover) Evaluate(item value.Value, context Context) (value.Value, error) {
	var rv value.Value
	switch item := item.(type) {
	case value.AnnotatedValue:
		rv = item.GetCover(this.text)
	}

	if rv == nil {
		return value.MISSING_VALUE, errors.NewEvaluationError(nil, "cover("+this.text+")")
	}

	return rv, nil
}

func (this *Cover) Value() value.Value {
	return this.covered.Value()
}

func (this *Cover) Static() Expression {
	return this.covered.Static()
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
	return this.covered.EquivalentTo(other)
}

func (this *Cover) DependsOn(other Expression) bool {
	return this.covered.DependsOn(other)
}

func (this *Cover) CoveredBy(keyspace string, exprs Expressions) bool {
	return this.covered.CoveredBy(keyspace, exprs)
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
	return NewCover(this.covered.Copy())
}

func (this *Cover) Covered() Expression {
	return this.covered
}

func (this *Cover) Text() string {
	return this.text
}
