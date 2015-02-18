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
Represents Construction expressions.
Objects can be constructed with arbitrary structure, nesting,
and embedded expressions, as represented by the construction
expressions in the N1QL specs. Type ObjectConstruct is a
struct that implements ExpressionBase and has field bindings
that is a map from string to Expression.
*/
type ObjectConstruct struct {
	ExpressionBase
	bindings map[string]Expression
}

/*
Create and return a new ObjectConstruct. Set its bindings field
as a new map from string to expressions with length of
input argument bindings. It ranges over these bindings and sets
the value to Expression() for the key Variable() for the map.
*/
func NewObjectConstruct(bindings Bindings) Expression {
	rv := &ObjectConstruct{
		bindings: make(map[string]Expression, len(bindings)),
	}

	for _, b := range bindings {
		rv.bindings[b.Variable()] = b.Expression()
	}

	rv.expr = rv
	return rv
}

/*
It calls the VisitObjectConstruct method by passing in the receiver,
and returns the interface. It is a visitor pattern.
*/
func (this *ObjectConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitObjectConstruct(this)
}

/*
Returns OBJECT value.
*/
func (this *ObjectConstruct) Type() value.Type { return value.OBJECT }

/*
Range over the bindings and evaluate each expression individually
using the Evaluate method. For all returned values excpt missing,
set the map[key] to the return value of the Evaluate method.
Return the map.
*/
func (this *ObjectConstruct) Evaluate(item value.Value, context Context) (value.Value, error) {
	m := make(map[string]interface{}, len(this.bindings))

	for key, expr := range this.bindings {
		val, err := expr.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if val.Type() != value.MISSING {
			m[key] = val
		}
	}

	return value.NewValue(m), nil
}

/*
Check if the input expression other is equivalent to the receiver
expressions. Cast the other expr to a pointer to ObjectConstruct.
If the length of the receivers bindings and other's bindings are
not equal return false. Range over the receivers bindings and
compare the expression values for each objectconstructs bindings
by calling equivalent to for those expressions. If not equal
return false. If all child expressions in the bindings are
equal, return true.
*/
func (this *ObjectConstruct) EquivalentTo(other Expression) bool {
	if this.ValueEquals(other) {
		return true
	}

	ol, ok := other.(*ObjectConstruct)
	if !ok {
		return false
	}

	if len(this.bindings) != len(ol.bindings) {
		return false
	}

	for key, expr := range this.bindings {
		oexpr, ok := ol.bindings[key]
		if !ok || !expr.EquivalentTo(oexpr) {
			return false
		}
	}

	return true
}

/*
Range over the bindings and append each value to a slice of
expressions. Return this slice. (Expressions is a slice of
expression).
*/
func (this *ObjectConstruct) Children() Expressions {
	rv := make(Expressions, 0, len(this.bindings))
	for _, expr := range this.bindings {
		rv = append(rv, expr)
	}

	return rv
}

/*
Range over the bindings and map the expressions to another expression.
Reset the expression to be the new expression at its corresponding key.
If mapping is successful return nil error.
*/
func (this *ObjectConstruct) MapChildren(mapper Mapper) (err error) {
	for key, expr := range this.bindings {
		vexpr, err := mapper.Map(expr)
		if err != nil {
			return err
		}

		this.bindings[key] = vexpr
	}

	return nil
}

func (this *ObjectConstruct) Copy() Expression {
	copies := make(Bindings, 0, len(this.bindings))
	for key, expr := range this.bindings {
		copies = append(copies, NewBinding(key, expr.Copy()))
	}

	return NewObjectConstruct(copies)
}
