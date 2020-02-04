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
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
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
	statementBase

	keyspace  *KeyspaceRef          `json:"keyspace"`
	key       expression.Expression `json:"key"`
	value     expression.Expression `json:"value"`
	options   expression.Expression `json:"options"`
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
	rv := &Insert{
		keyspace:  keyspace,
		key:       nil,
		value:     nil,
		options:   nil,
		values:    values,
		query:     nil,
		returning: returning,
	}

	rv.stmt = rv
	return rv
}

/*
The function NewInsertSelect returns a pointer to the Insert
struct by assigning the input attributes to the fields of the
struct, and setting values to nil. This represents the insert
select clause.
*/
func NewInsertSelect(keyspace *KeyspaceRef, key, value, options expression.Expression,
	query *Select, returning *Projection) *Insert {
	rv := &Insert{
		keyspace:  keyspace,
		key:       key,
		value:     value,
		options:   options,
		values:    nil,
		query:     query,
		returning: returning,
	}

	rv.stmt = rv
	return rv
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
It's an insert
*/
func (this *Insert) Type() string {
	return "INSERT"
}

/*
Applies mapper to all the expressions in the insert statement.
*/
func (this *Insert) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.MapExpressionsNoSelect(mapper)
	if err == nil && this.query != nil {
		err = this.query.MapExpressions(mapper)
	}

	return err
}

func (this *Insert) MapExpressionsNoSelect(mapper expression.Mapper) (err error) {
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

	if this.options != nil {
		this.options, err = mapper.Map(this.options)
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

	if this.options != nil {
		exprs = append(exprs, this.options)
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
func (this *Insert) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullKeyspace := this.keyspace.FullName()
	privs.Add(fullKeyspace, auth.PRIV_QUERY_INSERT)
	if this.returning != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT)
	}

	if this.query != nil {
		qp, err := this.query.Privileges()
		if err != nil {
			return nil, err
		}
		privs.AddAll(qp)
	}

	exprs := this.Expressions()
	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}
	privs.AddAll(subprivs)

	for _, expr := range exprs {
		privs.AddAll(expr.Privileges())
	}

	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the insert statement.
*/
func (this *Insert) Formalize() (err error) {
	if this.values != nil {
		f := expression.NewFormalizer("", nil)
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

func (this *Insert) Options() expression.Expression {
	return this.options
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
