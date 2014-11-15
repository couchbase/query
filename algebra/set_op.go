//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type setOp struct {
	first  Subresult `json:"first"`
	second Subresult `json:"second"`
}

func (this *setOp) Signature() value.Value {
	return this.first.Signature()
}

func (this *setOp) IsCorrelated() bool {
	return this.first.IsCorrelated() || this.second.IsCorrelated()
}

func (this *setOp) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.first.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.second.MapExpressions(mapper)
}

func (this *setOp) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	var ff, sf *expression.Formalizer
	ff, err = this.first.Formalize(parent)
	if err != nil {
		return nil, err
	}

	sf, err = this.second.Formalize(parent)
	if err != nil {
		return nil, err
	}

	// Intersection
	fa := ff.Allowed.Fields()
	sa := sf.Allowed.Fields()
	for field, _ := range fa {
		_, ok := sa[field]
		if !ok {
			delete(fa, field)
		}
	}

	ff.Allowed = value.NewValue(fa)
	if ff.Keyspace != sf.Keyspace {
		ff.Keyspace = ""
	}

	return ff, nil
}

func (this *setOp) First() Subresult {
	return this.first
}

func (this *setOp) Second() Subresult {
	return this.second
}

type unionSubresult struct {
	setOp
}

func (this *unionSubresult) Signature() value.Value {
	first := this.first.Signature()
	second := this.second.Signature()

	if first.Equals(second) {
		return first
	}

	if first.Type() != value.OBJECT ||
		second.Type() != value.OBJECT {
		return _JSON_SIGNATURE
	}

	rv := first.Copy()
	sa := second.Actual().(map[string]interface{})
	for k, v := range sa {
		cv, ok := rv.Field(k)
		if ok {
			if !value.NewValue(cv).Equals(value.NewValue(v)) {
				rv.SetField(k, _JSON_SIGNATURE)
			}
		} else {
			rv.SetField(k, v)
		}
	}

	return rv
}

var _JSON_SIGNATURE = value.NewValue(value.JSON.String())

type Union struct {
	unionSubresult
}

func NewUnion(first, second Subresult) Subresult {
	return &Union{
		unionSubresult{
			setOp{
				first:  first,
				second: second,
			},
		},
	}
}

func (this *Union) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitUnion(this)
}

type UnionAll struct {
	unionSubresult
}

func NewUnionAll(first, second Subresult) Subresult {
	return &UnionAll{
		unionSubresult{
			setOp{
				first:  first,
				second: second,
			},
		},
	}
}

func (this *UnionAll) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitUnionAll(this)
}

type Intersect struct {
	setOp
}

func NewIntersect(first, second Subresult) Subresult {
	return &Intersect{
		setOp{
			first:  first,
			second: second,
		},
	}
}

func (this *Intersect) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitIntersect(this)
}

type IntersectAll struct {
	setOp
}

func NewIntersectAll(first, second Subresult) Subresult {
	return &IntersectAll{
		setOp{
			first:  first,
			second: second,
		},
	}
}

func (this *IntersectAll) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitIntersectAll(this)
}

type Except struct {
	setOp
}

func NewExcept(first, second Subresult) Subresult {
	return &Except{
		setOp{
			first:  first,
			second: second,
		},
	}
}

func (this *Except) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitExcept(this)
}

type ExceptAll struct {
	setOp
}

func NewExceptAll(first, second Subresult) Subresult {
	return &ExceptAll{
		setOp{
			first:  first,
			second: second,
		},
	}
}

func (this *ExceptAll) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitExceptAll(this)
}
