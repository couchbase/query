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
	"encoding/json"
	"reflect"

	"github.com/couchbase/query/value"
)

/*
ExpressionBase is a base class for all expressions.
*/
type ExpressionBase struct {
	expr        Expression
	value       *value.Value
	conditional bool
	volatile    bool
}

var _NIL_VALUE value.Value

func (this *ExpressionBase) String() string {
	return NewStringer().Visit(this.expr)
}

func (this *ExpressionBase) MarshalJSON() ([]byte, error) {
	s := NewStringer().Visit(this.expr)
	return json.Marshal(s)
}

/*
Evaluate the expression for an indexing context. Support multiple
return values for array indexing.

By default, just call Evaluate().
*/
func (this *ExpressionBase) EvaluateForIndex(item value.Value, context Context) (
	value.Value, value.Values, error) {
	val, err := this.expr.Evaluate(item, context)
	return val, nil, err
}

/*
This method indicates if the expression is an array index key, and
if so, whether it is distinct.
*/
func (this *ExpressionBase) IsArrayIndexKey() (bool, bool) {
	return false, false
}

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *ExpressionBase) Value() value.Value {
	if this.value != nil {
		return *this.value
	}

	if this.volatile {
		this.value = &_NIL_VALUE
		return nil
	}

	propMissing := this.expr.PropagatesMissing()
	propNull := this.expr.PropagatesNull()

	for _, child := range this.expr.Children() {
		cv := child.Value()
		if cv == nil {
			if this.value == nil {
				this.value = &_NIL_VALUE
			}

			continue
		}

		if propMissing && cv.Type() == value.MISSING {
			this.value = &cv
			return *this.value
		}

		if propNull && cv.Type() == value.NULL {
			this.value = &cv
		}
	}

	if this.value != nil {
		return *this.value
	}

	defer func() {
		err := recover()
		if err != nil {
			this.value = &_NIL_VALUE
		}
	}()

	val, err := this.expr.Evaluate(nil, nil)
	if err != nil {
		this.value = &_NIL_VALUE
		return nil
	}

	this.value = &val
	return *this.value
}

/*
Returns a Constant or nil.
*/
func (this *ExpressionBase) Static() Expression {
	v := this.expr.Value()
	if v != nil {
		return NewConstant(v)
	}

	return nil
}

/*
It returns an empty string or the terminal identifier of
the expression.
*/
func (this *ExpressionBase) Alias() string {
	return ""
}

/*
Range over the children of the expression, and check if each
child is indexable. If not then return false as the expression
is not indexable. If all children are indexable, then return
true.
*/
func (this *ExpressionBase) Indexable() bool {
	for _, child := range this.expr.Children() {
		if !child.Indexable() {
			return false
		}
	}

	return true
}

/*
Returns false if any child's PropagatesMissing() returns false.
*/
func (this *ExpressionBase) PropagatesMissing() bool {
	if this.conditional {
		return false
	}

	for _, child := range this.expr.Children() {
		if !child.PropagatesMissing() {
			return false
		}
	}

	return true
}

/*
Returns false if any child's PropagatesNull() returns false.
*/
func (this *ExpressionBase) PropagatesNull() bool {
	if this.conditional {
		return false
	}

	for _, child := range this.expr.Children() {
		if !child.PropagatesNull() {
			return false
		}
	}

	return true
}

func (this *ExpressionBase) EquivalentTo(other Expression) bool {
	if this.ValueEquals(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	ours := this.expr.Children()
	theirs := other.Children()

	if len(ours) != len(theirs) {
		return false
	}

	for i, child := range ours {
		if !child.EquivalentTo(theirs[i]) {
			return false
		}
	}

	return true
}

func (this *ExpressionBase) DependsOn(other Expression) bool {
	if this.conditional || other.Value() != nil {
		return false
	}

	if this.expr.EquivalentTo(other) {
		return true
	}

	for _, child := range this.expr.Children() {
		if child.DependsOn(other) {
			return true
		}
	}

	return false
}

func (this *ExpressionBase) CoveredBy(keyspace string, exprs Expressions) bool {
	for _, expr := range exprs {
		if this.expr.EquivalentTo(expr) {
			return true
		}
	}

	children := this.expr.Children()
	for _, child := range children {
		if !child.CoveredBy(keyspace, exprs) {
			return false
		}
	}

	return true
}

/*
Return true if the receiver Expression value and the input
expression value are equal and not nil; else false.
*/
func (this *ExpressionBase) ValueEquals(other Expression) bool {
	thisValue := this.expr.Value()
	otherValue := other.Value()

	return thisValue != nil && otherValue != nil &&
		thisValue.Equals(otherValue).Truth()
}

/*
Set the receiver expression to the input expression.
*/
func (this *ExpressionBase) SetExpr(expr Expression) {
	if this.expr == nil {
		this.expr = expr
	}
}

/*
Range over the children of the expression, and check if each
child is limit pushable to index. If not then return false as
the limit is not pushable to index. If all children are limit
pushable, then return true.
*/
func (this *ExpressionBase) IsLimitPushable() bool {
	for _, child := range this.expr.Children() {
		if !child.IsLimitPushable() {
			return false
		}
	}

	return true
}
