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
)

type Group struct {
	by      expression.Expressions `json:by`
	letting expression.Bindings    `json:"letting"`
	having  expression.Expression  `json:"having"`
}

func NewGroup(by expression.Expressions, letting expression.Bindings, having expression.Expression) *Group {
	return &Group{
		by:      by,
		letting: letting,
		having:  having,
	}
}

func (this *Group) Formalize(f *expression.Formalizer) (*expression.Formalizer, error) {
	var err error

	if this.by != nil {
		for i, b := range this.by {
			this.by[i], err = f.Map(b)
			if err != nil {
				return nil, err
			}
		}
	}

	if this.letting != nil {
		err = f.PushBindings(this.letting)
		if err != nil {
			return nil, err
		}
	}

	if this.having != nil {
		this.having, err = f.Map(this.having)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

func (this *Group) MapExpressions(mapper expression.Mapper) (err error) {
	if this.by != nil {
		err = this.by.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.letting != nil {
		err = this.letting.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.having != nil {
		this.having, err = mapper.Map(this.having)
	}

	return
}

func (this *Group) By() expression.Expressions {
	return this.by
}

func (this *Group) Letting() expression.Bindings {
	return this.letting
}

func (this *Group) Having() expression.Expression {
	return this.having
}
