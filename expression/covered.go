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
	"github.com/couchbase/query/value"
)

type Covered struct {
	expr Expression
}

func NewCovered(expr Expression) *Covered {
	rv := &Covered{
		expr: expr,
	}

	return rv
}

func (this *Covered) String() string {
	return this.expr.String()
}

func (this *Covered) MarshalJSON() ([]byte, error) {
	return this.expr.MarshalJSON()
}

func (this *Covered) Accept(visitor Visitor) (interface{}, error) {
	return this.expr.Accept(visitor)
}

func (this *Covered) Type() value.Type {
	return this.expr.Type()
}

func (this *Covered) Evaluate(item value.Value, context Context) (value.Value, error) {
	return nil, nil
}

func (this *Covered) EvaluateForIndex(item value.Value, context Context) (value.Value, value.Values, error) {
	return nil, nil, nil
}

func (this *Covered) Value() value.Value {
	return this.expr.Value()
}

func (this *Covered) Static() Expression {
	return this.expr.Static()
}

func (this *Covered) Alias() string {
	return this.expr.Alias()
}

func (this *Covered) Indexable() bool {
	return this.expr.Indexable()
}

func (this *Covered) PropagatesMissing() bool {
	return this.expr.PropagatesMissing()
}

func (this *Covered) PropagatesNull() bool {
	return this.expr.PropagatesNull()
}

func (this *Covered) EquivalentTo(other Expression) bool {
	return this.expr.EquivalentTo(other)
}

func (this *Covered) DependsOn(other Expression) bool {
	return this.expr.DependsOn(other)
}

func (this *Covered) CoveredBy(exprs Expressions) bool {
	return this.expr.CoveredBy(exprs)
}

func (this *Covered) Children() Expressions {
	return this.expr.Children()
}

func (this *Covered) MapChildren(mapper Mapper) error {
	return this.expr.MapChildren(mapper)
}

func (this *Covered) Copy() Expression {
	return NewCovered(this.expr.Copy())
}
