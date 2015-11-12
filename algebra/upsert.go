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
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the upsert DML statement. Type Upsert is a
struct that contains fields mapping to each clause in
an upsert statement. Keyspace is the keyspace-ref for
the upsert stmt. Upserts can be performed using
the insert select clause or the insert-values clause.
key and value represent expressions and query represents
the select statement in an insert-select clause. values
represents pairs for the insert values. Returning
represents the returning clause. (Update and insert).
*/
type Upsert struct {
	statementBase

	keyspace  *KeyspaceRef          `json:"keyspace"`
	key       expression.Expression `json:"key"`
	value     expression.Expression `json:"value"`
	values    Pairs                 `json:"values"`
	query     *Select               `json:"select"`
	returning *Projection           `json:"returning"`
}

/*
The function NewUpsertValues returns a pointer to the Upsert
struct by assigning the input attributes to the fields of the
struct, and setting key, value and query to nil. This
represents the insert values clause in the upsert statement.
*/
func NewUpsertValues(keyspace *KeyspaceRef, values Pairs, returning *Projection) *Upsert {
	rv := &Upsert{
		keyspace:  keyspace,
		key:       nil,
		value:     nil,
		values:    values,
		query:     nil,
		returning: returning,
	}

	rv.stmt = rv
	return rv
}

/*
The function NewUpsertSelect returns a pointer to the Upsert
struct by assigning the input attributes to the fields of the
struct, and setting values to nil. This represents the insert
select clause in the upsert statement.
*/
func NewUpsertSelect(keyspace *KeyspaceRef, key, value expression.Expression,
	query *Select, returning *Projection) *Upsert {
	rv := &Upsert{
		keyspace:  keyspace,
		key:       key,
		value:     value,
		values:    nil,
		query:     query,
		returning: returning,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitUpsert method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Upsert) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpsert(this)
}

/*
The shape of the upsert statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Upsert) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
Applies mapper to all the expressions in the upsert statement.
*/
func (this *Upsert) MapExpressions(mapper expression.Mapper) (err error) {
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
func (this *Upsert) Expressions() expression.Expressions {
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
Returns all required privileges.
*/
func (this *Upsert) Privileges() (datastore.Privileges, errors.Error) {
	privs := datastore.NewPrivileges()
	privs[this.keyspace.Namespace()+":"+this.keyspace.Keyspace()] = datastore.PRIV_WRITE

	if this.query != nil {
		qp, err := this.query.Privileges()
		if err != nil {
			return nil, err
		}

		privs.Add(qp)
	}

	subprivs, err := subqueryPrivileges(this.Expressions())
	if err != nil {
		return nil, err
	}

	privs.Add(subprivs)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the upsert statement.
*/
func (this *Upsert) Formalize() (err error) {
	if this.values != nil {
		f := expression.NewFormalizer(nil)
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
Returns the keyspace-ref for the upsert statement.
*/
func (this *Upsert) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the key expression for the insert select
clause in the upsert statement.
*/
func (this *Upsert) Key() expression.Expression {
	return this.key
}

/*
Returns the value expression for the insert select
clause in the upsert statement.
*/
func (this *Upsert) Value() expression.Expression {
	return this.value
}

/*
Returns the value pairs for the insert values
clause in the upsert statement.
*/
func (this *Upsert) Values() Pairs {
	return this.values
}

/*
Returns the select query for the insert select
clause in the upsert statement.
*/
func (this *Upsert) Select() *Select {
	return this.query
}

/*
Returns the returning clause projection for the
upsert statement.
*/
func (this *Upsert) Returning() *Projection {
	return this.returning
}
