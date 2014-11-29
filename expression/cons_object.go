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
	"github.com/couchbaselabs/query/value"
)

type ObjectConstruct struct {
	ExpressionBase
	bindings map[string]Expression
}

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

func (this *ObjectConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitObjectConstruct(this)
}

func (this *ObjectConstruct) Type() value.Type { return value.OBJECT }

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

func (this *ObjectConstruct) EquivalentTo(other Expression) bool {
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

func (this *ObjectConstruct) Children() Expressions {
	rv := make(Expressions, 0, len(this.bindings))
	for _, expr := range this.bindings {
		rv = append(rv, expr)
	}

	return rv
}

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
