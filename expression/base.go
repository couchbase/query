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
	"reflect"

	"github.com/couchbaselabs/query/value"
)

/*
ExpressionBase is a base class for all expressions.
*/
type ExpressionBase struct {
	expr  Expression
	value value.Value
}

var _NIL_VALUE = value.NewValue(make([]interface{}, 0))

/*
Value() returns the static / constant value of this Expression, or
nil. Expressions that depend on data, clocks, or random numbers must
return nil.
*/
func (this *ExpressionBase) Value() value.Value {
	if this.value == _NIL_VALUE {
		return nil
	}

	if this.value != nil {
		return this.value
	}

	defer func() {
		err := recover()
		if err != nil {
			this.value = _NIL_VALUE
		}
	}()

	for _, child := range this.expr.Children() {
		if child.Value() == nil {
			this.value = _NIL_VALUE
			return nil
		}
	}

	var err error
	this.value, err = this.expr.Evaluate(nil, nil)
	if err != nil {
		this.value = _NIL_VALUE
		return nil
	}

	return this.value
}

/*
It returns an empty string for the terminal identifier of
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
Check if two expressions are equivalent. First compare the dynamic
type information of the two expressions, using reflect.TypeOf. If
it is not the same, then return false. Compare the length of the
two expressions. If they are not the same, then not equal, hence
return false. If the lengths are equal, range through the children
and check if they are equivalent by calling the EquivalentTo method
for each set of children and return false if not equal. If the
method hasnt returned till this point, then the expressions are
equal and return true.
*/
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

/*
Return true if the receiver Expression value and the input
expression value are equal and not nil; else false.
*/
func (this *ExpressionBase) ValueEquals(other Expression) bool {
	thisValue := this.expr.Value()
	otherValue := other.Value()

	return thisValue != nil && otherValue != nil &&
		thisValue.Equals(otherValue)
}

/*
Set the receiver expression to the input expression.
*/
func (this *ExpressionBase) SetExpr(expr Expression) {
	if this.expr == nil {
		this.expr = expr
	}
}
