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

type ObjectLiteral struct {
	ExpressionBase
	bindings map[string]Expression
}

func NewObjectLiteral(bindings Bindings) Expression {
	rv := &ObjectLiteral{
		bindings: make(map[string]Expression, len(bindings)),
	}

	for _, b := range bindings {
		rv.bindings[b.Variable()] = b.Expression()
	}

	return rv
}

func (this *ObjectLiteral) Evaluate(item value.Value, context Context) (value.Value, error) {
	m := make(map[string]interface{}, len(this.bindings))

	var err error
	for key, expr := range this.bindings {
		m[key], err = expr.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}

	return value.NewValue(m), nil
}

func (this *ObjectLiteral) EquivalentTo(other Expression) bool {
	ol, ok := other.(*ObjectLiteral)
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

func (this *ObjectLiteral) Fold() (Expression, error) {
	_, err := this.VisitChildren(&Folder{})
	if err != nil {
		return nil, err
	}

	c := make(map[string]interface{}, len(this.bindings))
	for key, expr := range this.bindings {
		switch expr := expr.(type) {
		case *Constant:
			c[key] = expr.Value()
		default:
			return this, nil
		}
	}

	return NewConstant(value.NewValue(c)), nil
}

func (this *ObjectLiteral) Children() Expressions {
	rv := make(Expressions, 0, len(this.bindings))
	for _, expr := range this.bindings {
		rv = append(rv, expr)
	}

	return rv
}

func (this *ObjectLiteral) VisitChildren(visitor Visitor) (Expression, error) {
	for key, expr := range this.bindings {
		vexpr, err := visitor.Visit(expr)
		if err != nil {
			return nil, err
		}

		this.bindings[key] = vexpr
	}

	return this, nil
}
