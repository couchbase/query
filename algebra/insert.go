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

func (this *Insert) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

func (this *Insert) MapExpressions(mapper expression.Mapper) (err error) {
	if this.key != nil {
		this.key, err = mapper.Map(this.key)
		if err != nil {
			return
		}
	}

	if this.values != nil {
		err = this.values.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.query != nil {
		err = this.query.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		err = this.returning.MapExpressions(mapper)
	}

	return
}

func (this *Insert) Formalize() (err error) {
	if this.values != nil {
		err = this.values.MapExpressions(expression.EMPTY_FORMALIZER)
		if err != nil {
			return
		}
	}

	if this.query != nil {
		err = this.query.Formalize()
		if err != nil {
			return
		}
	}

	f, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	if this.returning != nil {
		err = this.returning.MapExpressions(f)
	}

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
