//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

/*
Type collMap represents a struct that implements ExpressionBase.
It refers to the fields or attributes of a collection or map
used for Range transforms. Contains fields mapping and
bindings, and a when expression.
*/
type collMap struct {
	ExpressionBase
	mapping  Expression
	bindings Bindings
	when     Expression
}

/*
Returns the children as expressions of the collMap.
Append the mapping, binding expressions and the
when condition if present.
*/
func (this *collMap) Children() Expressions {
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
func (this *collMap) MapChildren(mapper Mapper) (err error) {
	this.mapping, err = mapper.Map(this.mapping)
	if err != nil {
		return
	}

	if mapper.MapBindings() {
		err = this.bindings.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.when != nil {
		this.when, err = mapper.Map(this.when)
		if err != nil {
			return
		}
	}

	return
}

/*
Return receiver bindings.
*/
func (this *collMap) Bindings() Bindings {
	return this.bindings
}

/*
Type collPred represents a struct that implements ExpressionBase.
It refers to the fields or attributes of a collection or map
used for Range predicates. Contains fields bindings, and satisfies
of type expression.
*/
type collPred struct {
	ExpressionBase
	bindings  Bindings
	satisfies Expression
}

func (this *collPred) Children() Expressions {
	d := make(Expressions, 0, 1+len(this.bindings))

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	d = append(d, this.satisfies)
	return d
}

/*
Map one set of expressions to another expression.
(Map Expresions associated with bindings and
the satisfies expression if it exists ).
*/
func (this *collPred) MapChildren(mapper Mapper) (err error) {
	if mapper.MapBindings() {
		err = this.bindings.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	this.satisfies, err = mapper.Map(this.satisfies)
	if err != nil {
		return
	}

	return
}

/*
Return receiver bindings.
*/
func (this *collPred) Bindings() Bindings {
	return this.bindings
}
