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

/*
Logical terms allow for combining other expressions using boolean logic.
Standard OR operators are supported.
*/
type Or struct {
	CommutativeFunctionBase
}

func NewOr(operands ...Expression) *Or {
	rv := &Or{
		*NewCommutativeFunctionBase("or", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Or) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitOr(this)
}

func (this *Or) Type() value.Type { return value.BOOLEAN }

/*
Return TRUE if any input has a truth value of TRUE, else return NULL,
MISSING, or FALSE in that order.
*/
func (this *Or) Evaluate(item value.Value, context Context) (value.Value, error) {
	missing := false
	null := false

	for _, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		switch arg.Type() {
		case value.NULL:
			null = true
		case value.MISSING:
			missing = true
		default:
			if arg.Truth() {
				return value.TRUE_VALUE, nil
			}
		}
	}

	if null {
		return value.NULL_VALUE, nil
	} else if missing {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
}

func (this *Or) Value() value.Value {
	if this.value != nil {
		return *this.value
	}

	if this.volatile() {
		this.value = &_NIL_VALUE
		return nil
	}

	var valMissing, valNull value.Value
	foundOther := false
	hasValue := true

	for _, child := range this.Children() {
		cv := child.Value()
		if child.HasExprFlag(EXPR_VALUE_MISSING) {
			valMissing = cv
		} else if child.HasExprFlag(EXPR_VALUE_NULL) {
			valNull = cv
		} else {
			if cv == nil {
				hasValue = false
			}
			foundOther = true
		}
	}

	// MB-28605 if one subterm of OR has MISSING or NULL value, check
	// other subterms if available
	if valMissing != nil && !foundOther {
		this.value = &valMissing
		return *this.value
	}
	if valNull != nil && !foundOther {
		this.value = &valNull
		return *this.value
	}

	if hasValue {
		var orExpr Expression
		orExpr = this
		if valMissing != nil || valNull != nil {
			subterms := make(Expressions, 0, len(this.Children()))
			for _, child := range this.Children() {
				if !child.HasExprFlag(EXPR_VALUE_MISSING) && !child.HasExprFlag(EXPR_VALUE_NULL) {
					subterms = append(subterms, child)
				}
			}
			if len(subterms) == 0 {
				orExpr = FALSE_EXPR
			} else {
				orExpr = NewOr(subterms...)
			}
		}

		defer func() {
			err := recover()
			if err != nil {
				this.value = &_NIL_VALUE
			}
		}()

		val, err := orExpr.Evaluate(nil, nil)
		if err != nil {
			this.value = &_NIL_VALUE
			return nil
		}
		this.value = &val
	} else {
		this.value = &_NIL_VALUE
	}

	return *this.value
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For OR, intersect the implicit covers of each child operand.
*/
func (this *Or) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	c := _COVERS_POOL.Get()
	defer _COVERS_POOL.Put(c)

	c = this.operands[0].FilterCovers(c)
	if len(c) == 0 {
		return covers
	}

	for i := 1; i < len(this.operands); i++ {
		ci := _COVERS_POOL.Get()
		defer _COVERS_POOL.Put(ci)

		ci = this.operands[i].FilterCovers(ci)
		if len(ci) == 0 {
			return covers
		}

		for s, v := range c {
			vi, ok := ci[s]
			if !ok || !v.Equals(vi).Truth() {
				delete(c, s)
			}
		}
	}

	for s, v := range c {
		covers[s] = v
	}

	return covers
}

var _COVERS_POOL = value.NewStringValuePool(16)

/*
Return TRUE for OR. This will include false positives.
*/
func (this *Or) MayOverlapSpans() bool {
	return true
}

/*
Factory method pattern.
*/
func (this *Or) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewOr(operands...)
	}
}
