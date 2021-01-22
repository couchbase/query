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
Represents object construction.
*/
type ObjectConstruct struct {
	ExpressionBase
	mapping  map[Expression]Expression
	bindings map[string]Expression
}

func NewObjectConstruct(mapping map[Expression]Expression) Expression {
	rv := &ObjectConstruct{
		mapping:  mapping,
		bindings: make(map[string]Expression, len(mapping)),
	}

	for name, value := range mapping {
		rv.bindings[name.String()] = value
	}

	rv.expr = rv
	rv.Value() // Initialize value
	return rv
}

/*
Visitor pattern.
*/
func (this *ObjectConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitObjectConstruct(this)
}

func (this *ObjectConstruct) Type() value.Type { return value.OBJECT }

func (this *ObjectConstruct) Evaluate(item value.Value, context Context) (value.Value, error) {
	if this.value != nil && *this.value != nil {
		return *this.value, nil
	}

	m := make(map[string]interface{}, len(this.mapping))

	for name, val := range this.mapping {
		n, err := name.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if n.Type() == value.MISSING || n.Type() == value.NULL {
			continue
		} else if n.Type() != value.STRING {
			return value.NULL_VALUE, nil
		}

		v, err := val.Evaluate(item, context)
		if err != nil {
			return nil, err
		}

		if v.Type() != value.MISSING {
			m[n.ToString()] = v
		}
	}

	return value.NewValue(m), nil
}

func (this *ObjectConstruct) PropagatesMissing() bool {
	return this.value != nil && *this.value != nil
}

func (this *ObjectConstruct) PropagatesNull() bool {
	return this.value != nil && *this.value != nil
}

func (this *ObjectConstruct) EquivalentTo(other Expression) bool {
	if this.valueEquivalentTo(other) {
		return true
	}

	ol, ok := other.(*ObjectConstruct)
	if !ok {
		return false
	}

	if len(this.bindings) != len(ol.bindings) {
		return false
	}

	for name, value := range this.bindings {
		ovalue, ok := ol.bindings[name]
		if !ok || !value.EquivalentTo(ovalue) {
			return false
		}
	}

	return true
}

func (this *ObjectConstruct) Children() Expressions {
	rv := make(Expressions, 0, 2*len(this.mapping))
	for name, value := range this.mapping {
		rv = append(rv, name, value)
	}

	return rv
}

func (this *ObjectConstruct) MapChildren(mapper Mapper) (err error) {
	mapped := make(map[Expression]Expression, len(this.mapping))

	for name, value := range this.mapping {
		n, err := mapper.Map(name)
		if err != nil {
			return err
		}

		v, err := mapper.Map(value)
		if err != nil {
			return err
		}

		// Expression.String() may change after Map()
		sname := name.String()
		sn := n.String()
		if sn == sname {
			this.bindings[sn] = v
		} else {
			this.bindings[sname] = nil
			delete(this.bindings, sname)
			this.bindings[sn] = v
		}

		mapped[n] = v
	}

	this.mapping = mapped
	return nil
}

func (this *ObjectConstruct) Copy() Expression {
	copies := make(map[Expression]Expression, len(this.mapping))
	for name, value := range this.mapping {
		copies[name.Copy()] = value.Copy()
	}

	rv := NewObjectConstruct(copies)
	rv.BaseCopy(this)
	return rv
}

func (this *ObjectConstruct) Mapping() map[Expression]Expression {
	return this.mapping
}
