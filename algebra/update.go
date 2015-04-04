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
	"encoding/json"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Update DML statement. Type Update is a
struct that contains fields mapping to each clause in
an update statement. Keyspace is the keyspace-ref for
the update stmt. The keys expression represents the
keys clause, set and unset represent the set and
unset clause, the limit expression represents the
limit clause and returning is the returning clause.
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

/*
The function NewUpdate returns a pointer to the Update
struct by assigning the input attributes to the fields
of the struct.
*/
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
It calls the VisitUpdate method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}

/*
The shape of the update statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Update) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
Applies mapper to all the expressions in the update statement.
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
func (this *Update) Privileges() (datastore.Privileges, errors.Error) {
	privs := datastore.NewPrivileges()
	privs[this.keyspace.Namespace()+":"+this.keyspace.Keyspace()] = datastore.PRIV_WRITE

	subprivs, err := subqueryPrivileges(this.Expressions())
	if err != nil {
		return nil, err
	}

	privs.Add(subprivs)
	return privs, nil
}

/*
Fully qualify identifiers for each of the constituent clauses
in the Update statement.
*/
func (this *Update) Formalize() (err error) {
	f, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	empty := expression.NewFormalizer()

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
Returns the keyspace-ref for the update statement.
*/
func (this *Update) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the keys expression defined by the use keys
clause.
*/
func (this *Update) Keys() expression.Expression {
	return this.keys
}

/*
Returns the indexes defined by the use index clause.
*/
func (this *Update) Indexes() IndexRefs {
	return this.indexes
}

/*
Returns the terms from the set clause in an update
statement.
*/
func (this *Update) Set() *Set {
	return this.set
}

/*
Returns the terms from the unset clause in an update
statement.
*/
func (this *Update) Unset() *Unset {
	return this.unset
}

/*
Returns the where clause expression in an update
statement.
*/
func (this *Update) Where() expression.Expression {
	return this.where
}

/*
Return the limit expression for the limit
clause in an update statement.
*/
func (this *Update) Limit() expression.Expression {
	return this.limit
}

/*
Returns the returning clause projection for the
update statement.
*/
func (this *Update) Returning() *Projection {
	return this.returning
}

/*
Represents the set clause in the update statement.
Type Set is a struct that contains the terms field
that represents setTerms.
*/
type Set struct {
	terms SetTerms
}

/*
The function NewSet returns a pointer to the Set
struct by assigning the input attributes to the
fields of the struct.
*/
func NewSet(terms SetTerms) *Set {
	return &Set{terms}
}

/*
Applies mapper to all the terms in the setTerms.
*/
func (this *Set) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *Set) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)
	for _, term := range this.terms {
		exprs = append(exprs, term.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for each term in the set terms.
*/
func (this *Set) Formalize(f *expression.Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns rhe terms in the set clause defined by
setTerms.
*/
func (this *Set) Terms() SetTerms {
	return this.terms
}

/*
Represents the Unset clause in the update statement.
Type Unset is a struct that contains the terms field
that represents UnsetTerms.
*/
type Unset struct {
	terms UnsetTerms
}

/*
The function NewUnset returns a pointer to the Unset
struct by assigning the input attributes to the
fields of the struct.
*/
func NewUnset(terms UnsetTerms) *Unset {
	return &Unset{terms}
}

/*
Applies mapper to all the terms in the UnsetTerms.
*/
func (this *Unset) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this.terms {
		err = term.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *Unset) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)
	for _, term := range this.terms {
		exprs = append(exprs, term.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for each term in the Unset terms.
*/
func (this *Unset) Formalize(f *expression.Formalizer) (err error) {
	for _, term := range this.terms {
		err = term.Formalize(f)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns rhe terms in the set clause defined by
UnsetTerms.
*/
func (this *Unset) Terms() UnsetTerms {
	return this.terms
}

/*
Represents Set terms from the set clause. Type SetTerms
is a slice containing SetTerm.
*/
type SetTerms []*SetTerm

/*
Represents the set clause. Type SetTerm is a struct
that contains fields mapping to each expression and
sub clause in the set clause. The path and value
represent the expression path and value and updateFor
is the update-for statement.
*/
type SetTerm struct {
	path      expression.Path       `json:"path"`
	value     expression.Expression `json:"value"`
	updateFor *UpdateFor            `json:"path-for"`
}

/*
The function NewSetTerm returns a pointer to the SetTerm
struct by assigning the input attributes to the fields
of the struct.
*/
func NewSetTerm(path expression.Path, value expression.Expression, updateFor *UpdateFor) *SetTerm {
	return &SetTerm{path, value, updateFor}
}

/*
Applies mapper to the path and value expressions, and updatefor
in the set Term.
*/
func (this *SetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	path, err := mapper.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)

	this.value, err = mapper.Map(this.value)
	if err != nil {
		return
	}

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *SetTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)
	exprs = append(exprs, this.path, this.value)

	if this.updateFor != nil {
		exprs = append(exprs, this.updateFor.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for the update for stmt, the path
and value expressions in the set clause.
*/
func (this *SetTerm) Formalize(f *expression.Formalizer) (err error) {
	if this.updateFor != nil {
		sv, err := f.PushBindings(this.updateFor.bindings)
		if err != nil {
			return err
		}

		defer f.PopBindings(sv)
	}

	path, err := f.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)
	this.value, err = f.Map(this.value)
	return
}

/*
Return the path expression in the set clause.
*/
func (this *SetTerm) Path() expression.Path {
	return this.path
}

/*
Return the value expression in the set clause.
*/
func (this *SetTerm) Value() expression.Expression {
	return this.value
}

/*
Return the update for statement in the set clause.
*/
func (this *SetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

/*
Marshals input into byte array.
*/
func (this *SetTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "setTerm"}
	r["path"] = expression.NewStringer().Visit(this.path)
	r["value"] = expression.NewStringer().Visit(this.value)
	r["updateFor"] = this.updateFor
	return json.Marshal(r)
}

/*
Represents Unset terms from the unset clause. Type UnsetTerms
is a slice containing UnsetTerm.
*/
type UnsetTerms []*UnsetTerm

/*
Represents the unset clause. Type UnsetTerm is a struct
that contains fields mapping to each expression and
sub clause in the unset clause. path represents the
expression path and updateFor is the update-for statement.
*/
type UnsetTerm struct {
	path      expression.Path `json:"path"`
	updateFor *UpdateFor      `json:"path-for"`
}

/*
The function NewUnsetTerm returns a pointer to the UnsetTerm
struct by assigning the input attributes to the fields
of the struct.
*/
func NewUnsetTerm(path expression.Path, updateFor *UpdateFor) *UnsetTerm {
	return &UnsetTerm{path, updateFor}
}

/*
Applies mapper to the path expressions and update for in
the unset Term.
*/
func (this *UnsetTerm) MapExpressions(mapper expression.Mapper) (err error) {
	path, err := mapper.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)

	if this.updateFor != nil {
		err = this.updateFor.MapExpressions(mapper)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *UnsetTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)
	exprs = append(exprs, this.path)

	if this.updateFor != nil {
		exprs = append(exprs, this.updateFor.Expressions()...)
	}

	return exprs
}

/*
Fully qualify identifiers for the update for stmt and the path
expression in the unset clause.
*/
func (this *UnsetTerm) Formalize(f *expression.Formalizer) (err error) {
	if this.updateFor != nil {
		sv, err := f.PushBindings(this.updateFor.bindings)
		if err != nil {
			return err
		}

		defer f.PopBindings(sv)
	}

	path, err := f.Map(this.path)
	if err != nil {
		return err
	}

	this.path = path.(expression.Path)
	return
}

/*
Return the path expression in the unset clause.
*/
func (this *UnsetTerm) Path() expression.Path {
	return this.path
}

/*
Return the update for statement in the unset clause.
*/
func (this *UnsetTerm) UpdateFor() *UpdateFor {
	return this.updateFor
}

/*
Marshals input into byte array.
*/
func (this *UnsetTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "unsetTerm"}
	r["path"] = expression.NewStringer().Visit(this.path)
	r["updateFor"] = this.updateFor
	return json.Marshal(r)
}

/*
Represents the UpdateFor statement. Type UpdateFor is a
struct that contains fields mapping to each expression
in the statement. Bindings and when map to the expressions
in the 'var in/within path' and when clause.
*/
type UpdateFor struct {
	bindings expression.Bindings
	when     expression.Expression
}

/*
The function NewUpdateFor returns a pointer to the
UpdateFor struct by assigning the input attributes
to the fields of the struct.
*/
func NewUpdateFor(bindings expression.Bindings, when expression.Expression) *UpdateFor {
	return &UpdateFor{bindings, when}
}

/*
Apply mapper to expressions in the when clause and bindings.
*/
func (this *UpdateFor) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.bindings.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.when != nil {
		this.when, err = mapper.Map(this.when)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *UpdateFor) Expressions() expression.Expressions {
	exprs := this.bindings.Expressions()

	if this.when != nil {
		exprs = append(exprs, this.when)
	}

	return exprs
}

/*
Return the expression bindings for the update for statement.
*/
func (this *UpdateFor) Bindings() expression.Bindings {
	return this.bindings
}

/*
Return the when expression for the when clause in the
update for statement.
*/
func (this *UpdateFor) When() expression.Expression {
	return this.when
}

/*
Marshals input into byte array.
*/
func (this *UpdateFor) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "updateFor"}
	r["bindings"] = this.bindings
	if this.when != nil {
		r["when"] = expression.NewStringer().Visit(this.when)
	}
	return json.Marshal(r)
}
