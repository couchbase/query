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
)

/*
Type collMap represents a struct that implements ExpressionBase.
It refers to the fields or attributes of a collection or map
used for Range transforms. Contains fields mapping and
bindings, and a when expression.
*/
type collMap interface {
	Expression
	Mapping() Expression
	Bindings() Bindings
	When() Expression
}

type collMapBase struct {
	ExpressionBase
	mapping  Expression
	bindings Bindings
	when     Expression
}

func (this *collMapBase) EquivalentTo(other Expression) bool {
	if this.ValueEquals(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	o := other.(collMap)
	return this.mapping.EquivalentTo(o.Mapping()) &&
		this.bindings.EquivalentTo(o.Bindings()) &&
		Equivalent(this.when, o.When())
}

/*
Returns the children as expressions of the collMap.
Append the mapping, binding expressions and the
when condition if present.
*/
func (this *collMapBase) Children() Expressions {
	d := make(Expressions, 0, 2+len(this.bindings))
	d = append(d, this.mapping)

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	if this.when != nil {
		d = append(d, this.when)
	}

	return d
}

/*
Map one set of expressions to another expression.
(Map Expresions associated with bindings and
the when expression if it exists. ).
*/
func (this *collMapBase) MapChildren(mapper Mapper) (err error) {
	this.mapping, err = mapper.Map(this.mapping)
	if err != nil {
		return
	}

	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.when != nil {
		this.when, err = mapper.Map(this.when)
		if err != nil {
			return
		}
	}

	return
}

func (this *collMapBase) Mapping() Expression {
	return this.mapping
}

func (this *collMapBase) Bindings() Bindings {
	return this.bindings
}

func (this *collMapBase) When() Expression {
	return this.when
}
