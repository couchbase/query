//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the delete DML statement. Type Delete is a
struct that contains fields mapping to each clause in
the delete stmt.  Keyspace is the keyspace-ref, keys
expression represents the use keys clause, the where
,limit, offset expression map to the where, limit, offset clause
and returning represents the returning clause.
*/
type Delete struct {
	statementBase

	keyspace     *KeyspaceRef          `json:"keyspace"`
	keys         expression.Expression `json:"keys"`
	indexes      IndexRefs             `json:"indexes"`
	where        expression.Expression `json:"where"`
	limit        expression.Expression `json:"limit"`
	offset       expression.Expression `json:"offset"`
	returning    *Projection           `json:"returning"`
	optimHints   *OptimHints           `json:"optimizer_hints"`
	validateKeys bool                  `json:"validate_keys"`
}

/*
The function NewDelete returns a pointer to the Delete
struct by assigning the input attributes to the fields
of the struct
*/
func NewDelete(keyspace *KeyspaceRef, keys expression.Expression, indexes IndexRefs,
	where, limit, offset expression.Expression, returning *Projection, optimHints *OptimHints, validateKeys bool) *Delete {
	rv := &Delete{
		keyspace:     keyspace,
		keys:         keys,
		indexes:      indexes,
		where:        where,
		limit:        limit,
		offset:       offset,
		returning:    returning,
		optimHints:   optimHints,
		validateKeys: validateKeys,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitDelete method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Delete) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDelete(this)
}

/*
The shape of the insert statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Delete) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
It's a delete
*/
func (this *Delete) Type() string {
	return "DELETE"
}

func (this *Delete) String() string {
	s := "delete "
	if this.optimHints != nil {
		s += this.optimHints.String() + " "
	}
	s += "from "
	s += this.keyspace.Path().ProtectedString()
	alias := this.keyspace.Alias()
	if len(alias) > 0 {
		s += " as " + alias
	}
	if this.keys != nil {
		if this.validateKeys {
			s += " use keys validate " + this.keys.String()
		} else {
			s += " use keys " + this.keys.String()
		}
	}
	if this.indexes != nil {
		s += " use index(" + this.indexes.String() + ")"
	}

	if this.where != nil {
		s += " where " + this.where.String()
	}
	if this.offset != nil {
		s += " offset " + this.offset.String()
	}
	if this.limit != nil {
		s += " limit " + this.limit.String()
	}
	if this.returning != nil {
		s += " returning " + this.returning.String()
	}
	return s
}

/*
Applies mapper to all the expressions in the delete statement.
*/
func (this *Delete) MapExpressions(mapper expression.Mapper) (err error) {
	if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return err
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return err
		}
	}

	if this.limit != nil {
		this.limit, err = mapper.Map(this.limit)
		if err != nil {
			return err
		}
	}

	if this.offset != nil {
		this.offset, err = mapper.Map(this.offset)
		if err != nil {
			return err
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
func (this *Delete) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)

	if this.keys != nil {
		exprs = append(exprs, this.keys)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.offset != nil {
		exprs = append(exprs, this.offset)
	}

	if this.returning != nil {
		exprs = append(exprs, this.returning.Expressions()...)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *Delete) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	props := this.keyspace.PrivilegeProps()
	fullKeyspace := this.keyspace.FullName()
	isSystem := this.keyspace.IsSystem()
	if isSystem {
		datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_DELETE, privs)
	} else {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_DELETE, props)
	}
	if this.returning != nil {
		if isSystem {
			datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_DELETE, privs)
		} else {
			privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT, props)
		}
	}

	exprs := this.Expressions()
	for _, expr := range exprs {
		privs.AddAll(expr.Privileges())
	}

	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}
	privs.AddAll(subprivs)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the delete statement.
*/
func (this *Delete) Formalize() (err error) {
	f, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	empty := expression.NewFormalizer("", nil)
	if this.keys != nil {
		_, err = this.keys.Accept(empty)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = f.Map(this.where)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		_, err = this.limit.Accept(empty)
		if err != nil {
			return
		}
	}

	if this.offset != nil {
		_, err = this.offset.Accept(empty)
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		_, err = this.returning.Formalize(f)
	}

	return
}

/*
Returns the keyspace-ref for the delete statement.
*/
func (this *Delete) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the use keys expression for the delete
statement.
*/
func (this *Delete) Keys() expression.Expression {
	return this.keys
}

func (this *Delete) ValidateKeys() bool {
	return this.validateKeys
}

/*
Returns the indexes defined by the use index clause.
*/
func (this *Delete) Indexes() IndexRefs {
	return this.indexes
}

/*
Returns the expression for the where clause in the
delete statement.
*/
func (this *Delete) Where() expression.Expression {
	return this.where
}

/*
Returns the expression for the limit clause in the
delete statement.
*/
func (this *Delete) Limit() expression.Expression {
	return this.limit
}

/*
Returns the expression for the Offset clause in the
delete statement.
*/
func (this *Delete) Offset() expression.Expression {
	return this.offset
}

/*
Returns the returning clause projection for the
delete statement.
*/
func (this *Delete) Returning() *Projection {
	return this.returning
}

/*
Optimier hints
*/
func (this *Delete) OptimHints() *OptimHints {
	return this.optimHints
}

func (this *Delete) SetOptimHints(optimHints *OptimHints) {
	this.optimHints = optimHints
}
