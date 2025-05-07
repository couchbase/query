//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"strings"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
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

func (this *Insert) String() string {
	var buf strings.Builder
	buf.WriteString("insert into ")
	buf.WriteString(this.keyspace.Path().ProtectedString())
	alias := this.keyspace.Alias()
	if len(alias) > 0 {
		buf.WriteString(" as `")
		buf.WriteString(alias)
		buf.WriteString("`")
	}
	if this.key != nil && this.value != nil {
		buf.WriteString(" (key ")
		buf.WriteString(this.key.String())
		buf.WriteString(", value ")
		buf.WriteString(this.value.String())
		if this.options != nil {
			buf.WriteString(", options ")
			buf.WriteString(this.options.String())
		}
		buf.WriteString(")")
	}
	if this.values != nil && len(this.values) > 0 {
		buf.WriteString(" values")
		var lastValuesElem bool
		valuesLen := len(this.values)
		for valuesIdx, v := range this.values {
			lastValuesElem = valuesIdx == valuesLen-1
			buf.WriteString("(")
			buf.WriteString(v.key.String())
			buf.WriteString(",")
			buf.WriteString(v.value.String())
			if v.options != nil {
				buf.WriteString(",")
				buf.WriteString(v.options.String())
			}
			buf.WriteString(")")
			if !lastValuesElem {
				buf.WriteString(",")
			}
		}
	} else if this.query != nil {
		buf.WriteString(this.query.String())
	}
	if this.returning != nil {
		buf.WriteString(" returning ")
		buf.WriteString(this.returning.String())
	}
	return buf.String()
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
	props := this.keyspace.PrivilegeProps()
	fullKeyspace := this.keyspace.FullName()
	isSystem := this.keyspace.IsSystem()

	if isSystem {
		datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_INSERT, privs)
	} else {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_INSERT, props)
	}

	if this.returning != nil {
		if isSystem {
			datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_SELECT, privs)
		} else {
			privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT, props)
		}
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
