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

/*
Represents the insert DML statement. Type Insert is a
struct that contains fields mapping to each clause in
an insert statement. Keyspace is the keyspace-ref for
the insert stmt. Inserts can be performed using
the insert select clause or the insert-values clause.
key and value represent expressions and query represents
the select statement in an insert-select clause. values
represents pairs for the insert values. Returning
represents the returning clause.
*/
type Insert struct {
	keyspace  *KeyspaceRef          `json:"keyspace"`
	key       expression.Expression `json:"key"`
	value     expression.Expression `json:"value"`
	values    Pairs                 `json:"values"`
	query     *Select               `json:"select"`
	returning *Projection           `json:"returning"`
}

/*
The function NewInsertValues returns a pointer to the Insert
struct by assigning the input attributes to the fields of the
struct, and setting key, value and query to nil. This
represents the insert values clause.
*/
func NewInsertValues(keyspace *KeyspaceRef, values Pairs, returning *Projection) *Insert {
	return &Insert{
		keyspace:  keyspace,
		key:       nil,
		value:     nil,
		values:    values,
		query:     nil,
		returning: returning,
	}
}

/*
The function NewInsertSelect returns a pointer to the Insert
struct by assigning the input attributes to the fields of the
struct, and setting values to nil. This represents the insert
select clause.
*/
func NewInsertSelect(keyspace *KeyspaceRef, key, value expression.Expression,
	query *Select, returning *Projection) *Insert {
	return &Insert{
		keyspace:  keyspace,
		key:       key,
		value:     value,
		values:    nil,
		query:     query,
		returning: returning,
	}
}

/*
It calls the VisitInsert method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Insert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInsert(this)
}

/*
The shape of the insert statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Insert) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
Applies mapper to all the expressions in the insert statement.
*/
func (this *Insert) MapExpressions(mapper expression.Mapper) (err error) {
	if this.key != nil {
		this.key, err = mapper.Map(this.key)
		if err != nil {
			return
		}
	}

	if this.value != nil {
		this.value, err = mapper.Map(this.value)
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

/*
Returns all contained Expressions.
*/
func (this *Insert) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.key != nil {
		exprs = append(exprs, this.key)
	}

	if this.value != nil {
		exprs = append(exprs, this.value)
	}

	if this.values != nil {
		exprs = append(exprs, this.values.Expressions()...)
	}

	if this.query != nil {
		exprs = append(exprs, this.query.Expressions()...)
	}

	if this.returning != nil {
		exprs = append(exprs, this.returning.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for each of the constituent clauses
in the insert statement.
*/
func (this *Insert) Formalize() (err error) {
	if this.values != nil {
		f := expression.NewFormalizer()
		err = this.values.MapExpressions(f)
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
		_, err = this.returning.Formalize(f)
	}

	return
}

/*
Returns the keyspace-ref for the insert statement.
*/
func (this *Insert) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the key expression for the insert select
clause.
*/
func (this *Insert) Key() expression.Expression {
	return this.key
}

/*
Returns the value expression for the insert select
clause.
*/
func (this *Insert) Value() expression.Expression {
	return this.value
}

/*
Returns the value pairs for the insert values
clause.
*/
func (this *Insert) Values() Pairs {
	return this.values
}

/*
Returns the select query for the insert select
clause.
*/
func (this *Insert) Select() *Select {
	return this.query
}

/*
Returns the returning clause projection for the
insert statement.
*/
func (this *Insert) Returning() *Projection {
	return this.returning
}
