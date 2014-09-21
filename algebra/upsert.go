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

type Upsert struct {
	keyspace  *KeyspaceRef          `json:"keyspace"`
	key       expression.Expression `json:"key"`
	values    expression.Expression `json:"values"`
	query     *Select               `json:"select"`
	returning *Projection           `json:"returning"`
}

func NewUpsertValues(keyspace *KeyspaceRef, key expression.Expression,
	values expression.Expression, returning *Projection) *Upsert {
	return &Upsert{
		keyspace:  keyspace,
		key:       key,
		values:    values,
		query:     nil,
		returning: returning,
	}
}

func NewUpsertSelect(keyspace *KeyspaceRef, key expression.Expression,
	query *Select, returning *Projection) *Upsert {
	return &Upsert{
		keyspace:  keyspace,
		key:       key,
		values:    nil,
		query:     query,
		returning: returning,
	}
}

func (this *Upsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpsert(this)
}

func (this *Upsert) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

func (this *Upsert) MapExpressions(mapper expression.Mapper) (err error) {
	if this.key != nil {
		this.key, err = mapper.Map(this.key)
		if err != nil {
			return
		}
	}

	if this.values != nil {
		this.values, err = mapper.Map(this.values)
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

func (this *Upsert) Formalize() (err error) {
	if this.values != nil {
		this.values, err = NewFormalizer().Map(this.values)
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

func (this *Upsert) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Upsert) Key() expression.Expression {
	return this.key
}

func (this *Upsert) Values() expression.Expression {
	return this.values
}

func (this *Upsert) Select() *Select {
	return this.query
}

func (this *Upsert) Returning() *Projection {
	return this.returning
}
