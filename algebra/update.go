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
	let          expression.Bindings   `json:"let"`
}

func NewUpdate(keyspace *KeyspaceRef, keys expression.Expression, indexes IndexRefs,
	set *Set, unset *Unset, where, limit expression.Expression, returning *Projection,
	optimHints *OptimHints, validateKeys bool, let expression.Bindings) *Update {

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
		let:          let,
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
	var buf strings.Builder
	buf.WriteString("update ")
	if this.optimHints != nil {
		buf.WriteString(this.optimHints.String())
		buf.WriteString(" ")
	}
	buf.WriteString(this.keyspace.Path().ProtectedString())
	alias := this.keyspace.Alias()
	if len(alias) > 0 {
		buf.WriteString(" as `")
		buf.WriteString(alias)
		buf.WriteString("`")
	}
	if this.keys != nil {
		if this.validateKeys {
			buf.WriteString(" use keys validate ")
			buf.WriteString(this.keys.String())
		} else {
			buf.WriteString(" use keys ")
			buf.WriteString(this.keys.String())
		}
	}
	if this.indexes != nil {
		buf.WriteString(" use index(")
		buf.WriteString(this.indexes.String())
		buf.WriteString(")")
	}
	if this.let != nil {
		buf.WriteString(" let ")
		buf.WriteString(stringBindings(this.let))
	}
	if this.set != nil {
		buf.WriteString(" set")
		var lastSetTerm bool
		setTermsLen := len(this.set.Terms())
		for setTermIdx, v := range this.set.Terms() {
			lastSetTerm = setTermIdx == setTermsLen-1
			if v.meta != nil {
				buf.WriteString(" ")
				buf.WriteString(v.meta.String())
				buf.WriteString(".")
				buf.WriteString(strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "("))
			} else {
				buf.WriteString(" ")
				buf.WriteString(strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "("))
			}
			buf.WriteString(" = ")
			buf.WriteString(v.value.String())
			if v.updateFor != nil {
				buf.WriteString(" for ")
				var lastUpdForBindTerm bool
				updForBindingsLen := len(v.updateFor.Bindings())
				for updForBindIdx, b := range v.updateFor.Bindings() {
					lastUpdForBindTerm = updForBindIdx == updForBindingsLen-1
					buf.WriteString(b.String())
					if !lastUpdForBindTerm {
						buf.WriteString(",")
					}
				}
				if v.updateFor.When() != nil {
					buf.WriteString(" when ")
					buf.WriteString(v.updateFor.When().String())
				}
				buf.WriteString(" end")
			}
			if !lastSetTerm {
				buf.WriteString(",")
			}
		}
	}
	if this.unset != nil {
		buf.WriteString(" unset")
		var lastUnsetTerm bool
		unsetTermsLen := len(this.unset.Terms())
		for unsetTermIdx, v := range this.unset.Terms() {
			lastUnsetTerm = unsetTermIdx == unsetTermsLen-1
			if v.meta != nil {
				buf.WriteString(" ")
				buf.WriteString(v.meta.String())
				buf.WriteString(".")
				buf.WriteString(strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "("))
			} else {
				buf.WriteString(" ")
				buf.WriteString(strings.TrimPrefix(strings.TrimSuffix(v.path.String(), ")"), "("))
			}
			if v.updateFor != nil {
				buf.WriteString(" for ")
				var lastUpdForTerm bool
				updateForBindingsLen := len(v.updateFor.Bindings())
				for updForBindIdx, b := range v.updateFor.Bindings() {
					lastUpdForTerm = updForBindIdx == updateForBindingsLen-1
					buf.WriteString(b.String())
					if !lastUpdForTerm {
						buf.WriteString(",")
					}
				}
				if v.updateFor.When() != nil {
					buf.WriteString(" when ")
					buf.WriteString(v.updateFor.When().String())
				}
				buf.WriteString(" end")
			}
			if !lastUnsetTerm {
				buf.WriteString(",")
			}
		}
	}
	if this.where != nil {
		buf.WriteString(" where ")
		buf.WriteString(this.where.String())
	}
	if this.limit != nil {
		buf.WriteString(" limit ")
		buf.WriteString(this.limit.String())
	}
	if this.returning != nil {
		buf.WriteString(" returning ")
		buf.WriteString(this.returning.String())
	}
	return buf.String()
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

	if this.let != nil {
		err = this.let.MapExpressions(mapper)
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

	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
	}

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

	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
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
	isSystem := this.keyspace.IsSystem()

	if isSystem {
		datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_UPDATE, privs)
	} else {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_UPDATE, props)
	}

	if this.returning != nil {
		if isSystem {
			datastore.GetSystemstore().PrivilegesFromPath(fullKeyspace, this.keyspace.Keyspace(), auth.PRIV_QUERY_SELECT, privs)
		} else {
			privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT, props)
		}

		if this.returning.HasSystemXattrs() {
			privs.Add(fullKeyspace, auth.PRIV_XATTRS, props)
		}
	}

	exprs := this.Expressions()
	subprivs, err := subqueryPrivileges(exprs)
	if err != nil {
		return nil, err
	}
	privs.AddAll(subprivs)

	if (this.set != nil && this.set.HasSystemXattrs()) || (this.unset != nil && this.unset.HasSystemXattrs()) {
		privs.Add(fullKeyspace, auth.PRIV_XATTRS_WRITE, props)
	}

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

	if this.let != nil {
		err = f.PushBindings(this.let, false)
		if err != nil {
			return
		}
	}

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

func (this *Update) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Update) Keys() expression.Expression {
	return this.keys
}

func (this *Update) ValidateKeys() bool {
	return this.validateKeys
}

func (this *Update) Indexes() IndexRefs {
	return this.indexes
}

func (this *Update) Let() expression.Bindings {
	return this.let
}

func (this *Update) Set() *Set {
	return this.set
}

func (this *Update) Unset() *Unset {
	return this.unset
}

func (this *Update) Where() expression.Expression {
	return this.where
}

func (this *Update) Limit() expression.Expression {
	return this.limit
}

func (this *Update) Returning() *Projection {
	return this.returning
}

func (this *Update) OptimHints() *OptimHints {
	return this.optimHints
}

func (this *Update) SetOptimHints(optimHints *OptimHints) {
	this.optimHints = optimHints
}
