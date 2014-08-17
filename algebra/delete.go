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

type Delete struct {
	keyspace  *KeyspaceRef          `json:"keyspace"`
	keys      expression.Expression `json:"keys"`
	where     expression.Expression `json:"where"`
	limit     expression.Expression `json:"limit"`
	returning *Projection           `json:"returning"`
}

func NewDelete(keyspace *KeyspaceRef, keys, where, limit expression.Expression,
	returning *Projection) *Delete {
	return &Delete{keyspace, keys, where, limit, returning}
}

func (this *Delete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDelete(this)
}

func (this *Delete) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.keys != nil {
		expr, err := visitor.Visit(this.keys)
		if err != nil {
			return err
		}

		this.keys = expr.(expression.Expression)
	}

	if this.where != nil {
		expr, err := visitor.Visit(this.where)
		if err != nil {
			return err
		}

		this.where = expr.(expression.Expression)
	}

	if this.limit != nil {
		expr, err := visitor.Visit(this.limit)
		if err != nil {
			return err
		}

		this.limit = expr.(expression.Expression)
	}

	if this.returning != nil {
		err = this.returning.VisitExpressions(visitor)
	}

	return
}

func (this *Delete) Formalize() (err error) {
	return
}

func (this *Delete) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Delete) Keys() expression.Expression {
	return this.keys
}

func (this *Delete) Where() expression.Expression {
	return this.where
}

func (this *Delete) Limit() expression.Expression {
	return this.limit
}

func (this *Delete) Returning() *Projection {
	return this.returning
}
