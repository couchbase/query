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
Represents the UPDATE statement.
*/
type Update struct {
	statementBase

	keyspace  *KeyspaceRef          `json:"keyspace"`
	keys      expression.Expression `json:"keys"`
	indexes   IndexRefs             `json:"indexes"`
	set       *Set                  `json:"set"`
	unset     *Unset                `json:"unset"`
	where     expression.Expression `json:"where"`
	limit     expression.Expression `json:"limit"`
	returning *Projection           `json:"returning"`
}

func NewUpdate(keyspace *KeyspaceRef, keys expression.Expression, indexes IndexRefs,
	set *Set, unset *Unset, where, limit expression.Expression, returning *Projection) *Update {
	rv := &Update{
		keyspace:  keyspace,
		keys:      keys,
		indexes:   indexes,
		set:       set,
		unset:     unset,
		where:     where,
		limit:     limit,
		returning: returning,
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

/*
Returns all required privileges.
*/
func (this *Update) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullKeyspace := this.keyspace.FullName()
	privs.Add(fullKeyspace, auth.PRIV_WRITE)
	privs.Add(fullKeyspace, auth.PRIV_QUERY_UPDATE)

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
