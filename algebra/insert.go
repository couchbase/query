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

type Insert struct {
	keyspace  *KeyspaceRef           `json:"keyspace"`
	key       expression.Expression  `json:"key"`
	values    expression.Expressions `json:"values"`
	query     *Select                `json:"select"`
	returning *Projection            `json:"returning"`
}

func NewInsertValues(keyspace *KeyspaceRef, key expression.Expression,
	values expression.Expressions, returning *Projection) *Insert {
	return &Insert{
		keyspace:  keyspace,
		key:       key,
		values:    values,
		query:     nil,
		returning: returning,
	}
}

func NewInsertSelect(keyspace *KeyspaceRef, key expression.Expression,
	query *Select, returning *Projection) *Insert {
	return &Insert{
		keyspace:  keyspace,
		key:       key,
		values:    nil,
		query:     query,
		returning: returning,
	}
}

func (this *Insert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInsert(this)
}

func (this *Insert) VisitExpressions(visitor expression.Visitor) (err error) {
	if this.key != nil {
		expr, err := visitor.Visit(this.key)
		if err != nil {
			return err
		}

		this.key = expr.(expression.Expression)
	}

	if this.values != nil {
		err = this.values.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.query != nil {
		err = this.query.VisitExpressions(visitor)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		err = this.returning.VisitExpressions(visitor)
	}

	return
}

func (this *Insert) Formalize() (err error) {
	return
}

func (this *Insert) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Insert) Key() expression.Expression {
	return this.key
}

func (this *Insert) Values() expression.Expressions {
	return this.values
}

func (this *Insert) Select() *Select {
	return this.query
}

func (this *Insert) Returning() *Projection {
	return this.returning
}
