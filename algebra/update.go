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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the UPDATE statement.
*/
type Update struct {
	statementBase

	keyspace     *KeyspaceRef          `json:"keyspace"`
	keys         expression.Expression `json:"keys"`
	indexes      IndexRefs             `json:"indexes"`
	set          *Set                  `json:"set"`
	unset        *Unset                `json:"unset"`
	where        expression.Expression `json:"where"`
	limit        expression.Expression `json:"limit"`
	returning    *Projection           `json:"returning"`
	optimHints   *OptimHints           `json:"optimizer_hints"`
	validateKeys bool                  `json:"validate_keys"`
}

func NewUpdate(keyspace *KeyspaceRef, keys expression.Expression, indexes IndexRefs,
	set *Set, unset *Unset, where, limit expression.Expression, returning *Projection,
	optimHints *OptimHints, validateKeys bool) *Update {

	rv := &Update{
		keyspace:     keyspace,
		keys:         keys,
		indexes:      indexes,
		set:          set,
		unset:        unset,
		where:        where,
		limit:        limit,
		returning:    returning,
		optimHints:   optimHints,
		validateKeys: validateKeys,
	}

	rv.stmt = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}

/*
The shape of the UPDATE statement is the signature of its
RETURNING clause. If not present return value is nil.
*/
func (this *Update) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
It's an Update
*/
func (this *Update) Type() string {
	return "UPDATE"
}

func (this *Update) String() string {
	s := "update "
	if this.optimHints != nil {
		s += this.optimHints.String() + " "
	}
	s += this.keyspace.Path().ProtectedString()
	alias := this.keyspace.Alias()
	if len(alias) > 0 {
		s += " as `" + alias + "`"
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
	if this.set != nil {
		s += " set"
		for _, v := range this.set.Terms() {
			if v.meta != nil {
				s += " " + v.meta.String()
			} else {
				s += " " + strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "(")
			}
			s += " = " + v.value.String()
			if v.updateFor != nil {
				s += " for "
				for _, b := range v.updateFor.Bindings() {
					s += b.String() + ","
				}
				s = s[:len(s)-1]
				if v.updateFor.When() != nil {
					s += " when " + v.updateFor.When().String()
				}
				s += " end"
			}
			s += ","
		}
		s = s[:len(s)-1]
	}
	if this.unset != nil {
		s += " unset"
		for _, v := range this.unset.Terms() {
			s += " " + strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "(")
			if v.updateFor != nil {
				s += " for "
				for _, b := range v.updateFor.Bindings() {
					s += b.String() + ","
				}
				s = s[:len(s)-1]
				if v.updateFor.When() != nil {
					s += " when " + v.updateFor.When().String()
				}
				s += " end"
			}
			s += ","
		}
		s = s[:len(s)-1]
	}

	if this.where != nil {
		s += " where " + this.where.String()
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
Applies mapper to all the expressions in the UPDATE statement.
*/
func (this *Update) MapExpressions(mapper expression.Mapper) (err error) {
	if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return
		}
	}

	if this.set != nil {
		err = this.set.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.unset != nil {
		err = this.unset.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		this.limit, err = mapper.Map(this.limit)
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
func (this *Update) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.keys != nil {
		exprs = append(exprs, this.keys)
	}

	if this.set != nil {
		exprs = append(exprs, this.set.Expressions()...)
	}

	if this.unset != nil {
		exprs = append(exprs, this.unset.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.returning != nil {
		exprs = append(exprs, this.returning.Expressions()...)
	}

	return exprs
}

func (this *Update) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.keys != nil {
		exprs = append(exprs, this.keys)
	}

	if this.set != nil {
		exprs = append(exprs, this.set.NonMutatedExpressions()...)
	}

	if this.unset != nil {
		exprs = append(exprs, this.unset.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *Update) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullKeyspace := this.keyspace.FullName()
	props := this.keyspace.PrivilegeProps()
	privs.Add(fullKeyspace, auth.PRIV_QUERY_UPDATE, props)
	if this.returning != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT, props)
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
in the UPDATE statement.
*/
func (this *Update) Formalize() (err error) {
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

	if this.set != nil {
		err = this.set.Formalize(f)
		if err != nil {
			return
		}
	}

	if this.unset != nil {
		err = this.unset.Formalize(f)
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

	if this.returning != nil {
		_, err = this.returning.Formalize(f)
	}

	return
}

/*
Returns the keyspace-ref for the UPDATE statement.
*/
func (this *Update) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the keys expression defined by the USE KEYS
clause.
*/
func (this *Update) Keys() expression.Expression {
	return this.keys
}

func (this *Update) ValidateKeys() bool {
	return this.validateKeys
}

/*
Returns the indexes defined by the USE INDEX clause.
*/
func (this *Update) Indexes() IndexRefs {
	return this.indexes
}

/*
Returns the terms from the SET clause in an UPDATE
statement.
*/
func (this *Update) Set() *Set {
	return this.set
}

/*
Returns the terms from the UNSET clause in an UPDATE
statement.
*/
func (this *Update) Unset() *Unset {
	return this.unset
}

/*
Returns the WHERE clause expression in an UPDATE
statement.
*/
func (this *Update) Where() expression.Expression {
	return this.where
}

/*
Returns the limit expression for the LIMIT
clause in an UPDATE statement.
*/
func (this *Update) Limit() expression.Expression {
	return this.limit
}

/*
Returns the RETURNING clause projection for the
UPDATE statement.
*/
func (this *Update) Returning() *Projection {
	return this.returning
}

/*
Optimier hints
*/
func (this *Update) OptimHints() *OptimHints {
	return this.optimHints
}

func (this *Update) SetOptimHints(optimHints *OptimHints) {
	this.optimHints = optimHints
}
