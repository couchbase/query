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
Base for ANY, EVERY, and ANY AND EVERY collection predicates.
*/
type collPred interface {
	Expression
	Bindings() Bindings
	Satisfies() Expression
}

type collPredBase struct {
	ExpressionBase
	bindings  Bindings
	satisfies Expression
}

func (this *collPredBase) EquivalentTo(other Expression) bool {
	if this.ValueEquals(other) {
		return true
	}

	if reflect.TypeOf(this.expr) != reflect.TypeOf(other) {
		return false
	}

	o := other.(collPred)
	return this.bindings.EquivalentTo(o.Bindings()) &&
		this.satisfies.EquivalentTo(o.Satisfies())
}

func (this *collPredBase) Children() Expressions {
	d := make(Expressions, 0, 1+len(this.bindings))

	for _, b := range this.bindings {
		d = append(d, b.Expression())
	}

	d = append(d, this.satisfies)
	return d
}

func (this *collPredBase) MapChildren(mapper Mapper) (err error) {
	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.satisfies, err = mapper.Map(this.satisfies)
	if err != nil {
		return
	}

	return
}

func (this *collPredBase) Bindings() Bindings {
	return this.bindings
}

func (this *collPredBase) Satisfies() Expression {
	return this.satisfies
}
