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

	var er error
	for b, e := range this.bindings {
		m[b], er = e.Evaluate(item, context)
		if er != nil {
			return nil, er
		}
	}

	return value.NewValue(m), nil
}

func (this *ObjectLiteral) EquivalentTo(other Expression) bool {
	ol, ok := other.(*ObjectLiteral)
	if !ok {
		return false
	}

	if (len(this.bindings) != len(ol.bindings)) {
		return false
	}

	for b, e := range this.bindings {
		oe, ok := ol.bindings[b]
		if !ok || !e.EquivalentTo(oe) {
			return false
		}
	}

	return true
}

func (this *ObjectLiteral) Fold() (Expression, error) {
	v, e := this.VisitChildren(&Folder{})
	if e != nil {
		return v, e
	}

	c := make(map[string]interface{}, len(this.bindings))
	for b, e := range this.bindings {
		switch e := e.(type) {
		case *Constant:
			c[b] = e.Value()
		default:
			return this, nil
		}
	}

	return NewConstant(value.NewValue(c)), nil
}

func (this *ObjectLiteral) Children() Expressions {
	rv := make(Expressions, 0, len(this.bindings))
	for _, e := range this.bindings {
		rv = append(rv, e)
	}

	return rv
}

func (this *ObjectLiteral) VisitChildren(visitor Visitor) (Expression, error) {
	var ve Expression
	var re error
	for v, e := range this.bindings {
		ve, re = visitor.Visit(e)
		if re != nil {
			return nil, re
		}

		this.bindings[v] = ve
	}

	return this, nil
}
