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

	"github.com/couchbase/query/value"
)

/*
Base for ARRAY, FIRST, and OBJECT collection expressions.
*/
type collMap interface {
	Expression
	NameMapping() Expression
	ValueMapping() Expression
	Bindings() Bindings
	When() Expression
	EquivalentCollMap(other Expression) bool
}

type collMapBase struct {
	ExpressionBase
	nameMapping  Expression
	valueMapping Expression
	bindings     Bindings
	when         Expression
}

func (this *collMapBase) PropagatesMissing() bool {
	return false
}

func (this *collMapBase) PropagatesNull() bool {
	return false
}

func (this *collMapBase) EquivalentTo(other Expression) bool {
	return this.equivalentTo(other, true)
}

func (this *collMapBase) EquivalentCollMap(other Expression) bool {
	return this.equivalentTo(other, false)
}

// strict = true: must be exactly the same
// strict = false: allow binding variable names to be different
func (this *collMapBase) equivalentTo(other Expression, strict bool) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	o := other.(collMap)
	if strict {
		return this.valueMapping.EquivalentTo(o.ValueMapping()) &&
			this.bindings.EquivalentTo(o.Bindings()) &&
			Equivalent(this.when, o.When()) &&
			Equivalent(this.nameMapping, o.NameMapping())
	}
	return equivalentBindingsWithExpression(this.bindings, o.Bindings(),
		Expressions{this.valueMapping, this.when}, Expressions{o.ValueMapping(), o.When()}) &&
		Equivalent(this.nameMapping, o.NameMapping())
}

func (this *collMapBase) Children() Expressions {
	d := make(Expressions, 0, 3+len(this.bindings))

	if this.nameMapping != nil {
		d = append(d, this.nameMapping)
	}

	d = append(d, this.valueMapping)

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	if this.when != nil {
		d = append(d, this.when)
	}

	return d
}

func (this *collMapBase) MapChildren(mapper Mapper) (err error) {
	if this.nameMapping != nil {
		this.nameMapping, err = mapper.Map(this.nameMapping)
		if err != nil {
			return
		}
	}

	this.valueMapping, err = mapper.Map(this.valueMapping)
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

func (this *collMapBase) SurvivesGrouping(groupKeys Expressions, allowed *value.ScopeValue) (
	bool, Expression) {
	for _, key := range groupKeys {
		if this.EquivalentTo(key) {
			return true, nil
		}
	}

	vars := _VARS_POOL.Get()
	defer _VARS_POOL.Put(vars)
	allowed = value.NewScopeValue(vars, allowed)
	allow_flags := value.NewValue(uint32(IDENT_IS_VARIABLE))
	for _, b := range this.bindings {
		allowed.SetField(b.Variable(), allow_flags)
	}

	for _, child := range this.Children() {
		ok, _ := child.SurvivesGrouping(groupKeys, allowed)
		if !ok {
			return ok, nil
		}
	}

	return true, nil
}

func (this *collMapBase) NameMapping() Expression {
	return this.nameMapping
}

func (this *collMapBase) ValueMapping() Expression {
	return this.valueMapping
}

func (this *collMapBase) Bindings() Bindings {
	return this.bindings
}

func (this *collMapBase) When() Expression {
	return this.when
}
