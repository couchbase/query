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
Represents the merge DML statement. Type Merge is a
struct that contains fields mapping to each clause in
the statement.  Keyspace is the keyspace-ref for
the merge stmt. Merge source represents the path or a
select statement with an alias, the key expression
represents the keys clause. Merge actions can have
three possible statements, the merge update, merge
delete or the merge insert statement. Limit represents
the limit clause and Returning represents the returning
clause.
*/
type Merge struct {
	statementBase

	keyspace  *KeyspaceRef          `json:"keyspace"`
	indexes   IndexRefs             `json:"indexes"`
	source    *MergeSource          `json:"source"`
	on        expression.Expression `json:"on"`
	isOnKey   bool                  `json:"is_on_key"`
	actions   *MergeActions         `json:"actions"`
	limit     expression.Expression `json:"limit"`
	returning *Projection           `json:"returning"`
}

/*
The function NewMerge returns a pointer to the Merge
struct by assigning the input attributes to the fields
of the struct.
*/
func NewMerge(keyspace *KeyspaceRef, indexes IndexRefs, source *MergeSource,
	isOnKey bool, on expression.Expression, actions *MergeActions,
	limit expression.Expression, returning *Projection) *Merge {
	rv := &Merge{
		keyspace:  keyspace,
		indexes:   indexes,
		source:    source,
		on:        on,
		isOnKey:   isOnKey,
		actions:   actions,
		limit:     limit,
		returning: returning,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitMerge method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

/*
The shape of the merge statements is the signature of its
returning clause. If not present return value is nil.
*/
func (this *Merge) Signature() value.Value {
	if this.returning != nil {
		return this.returning.Signature()
	} else {
		return nil
	}
}

/*
Applies mapper to all the expressions in the merge statement.
*/
func (this *Merge) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.source.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.on, err = mapper.Map(this.on)
	if err != nil {
		return
	}

	err = this.actions.MapExpressions(mapper)
	if err != nil {
		return
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
func (this *Merge) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 64)

	exprs = append(exprs, this.source.Expressions()...)
	exprs = append(exprs, this.on)
	exprs = append(exprs, this.actions.Expressions()...)

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.returning != nil {
		exprs = append(exprs, this.returning.Expressions()...)
	}

	return exprs
}

func (this *Merge) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 64)

	exprs = append(exprs, this.source.Expressions()...)
	exprs = append(exprs, this.on)
	exprs = append(exprs, this.actions.NonMutatedExpressions()...)

	if this.limit != nil {
		exprs = append(exprs, this.limit)
	}

	if this.returning != nil && this.Actions().Delete() != nil {
		exprs = append(exprs, this.returning.Expressions()...)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *Merge) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullKeyspace := this.keyspace.FullName()
	if this.returning != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT)
	}

	sp, err := this.source.Privileges()
	if err != nil {
		return nil, err
	}
	privs.AddAll(sp)

	if this.actions != nil {
		this.actions.AddPrivilegesFor(privs, fullKeyspace)
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
in the merge statement.
*/
func (this *Merge) Formalize() (err error) {
	kf, err := this.keyspace.Formalize()
	if err != nil {
		return err
	}

	sf, err := this.source.Formalize()
	if err != nil {
		return err
	}

	if kf.Keyspace() != "" && kf.Keyspace() == sf.Keyspace() {
		return errors.NewDuplicateAliasError("MERGE", kf.Keyspace(), "semantics.merge.duplicate_alias")
	}

	f := expression.NewFormalizer("", nil)

	if kf.Keyspace() != "" {
		f.SetAllowedAlias(kf.Keyspace(), true)
	}

	if sf.Keyspace() != "" {
		f.SetAllowedAlias(sf.Keyspace(), true)
	}

	if this.isOnKey {
		this.on, err = sf.Map(this.on)
	} else {
		this.on, err = f.Map(this.on)
	}
	if err != nil {
		return err
	}

	// need to formalize separately for INSERT and
	// UPDATE/DELETE since INSERT can only reference source
	if this.actions.insert != nil {
		err = this.actions.insert.Formalize(sf)
		if err != nil {
			return
		}
	}

	if this.actions.update != nil {
		err = this.actions.update.Formalize(f)
		if err != nil {
			return
		}
	}

	if this.actions.delete != nil {
		err = this.actions.delete.Formalize(f)
		if err != nil {
			return
		}
	}

	if this.limit != nil {
		_, err = this.limit.Accept(expression.NewFormalizer("", nil))
		if err != nil {
			return
		}
	}

	if this.returning != nil {
		_, err = this.returning.Formalize(kf)
	}

	return
}

/*
Returns the keyspace-ref for the merge statement.
*/
func (this *Merge) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

/*
Returns the index hints for the merge statement.
*/
func (this *Merge) Indexes() IndexRefs {
	return this.indexes
}

/*
Returns the merge source for the merge statement.
*/
func (this *Merge) Source() *MergeSource {
	return this.source
}

/*
Returns the on expression for the on clause in
the merge statement.
*/
func (this *Merge) On() expression.Expression {
	return this.on
}

/*
Returns whether the on clause is using 'on keys' syntax
*/
func (this *Merge) IsOnKey() bool {
	return this.isOnKey
}

/*
Returns the merge actions for the merge statement.
*/
func (this *Merge) Actions() *MergeActions {
	return this.actions
}

/*
Returns the limit expression for the limit clause
in the merge statement.
*/
func (this *Merge) Limit() expression.Expression {
	return this.limit
}

/*
Returns the returning clause projection for the
merge statement.
*/
func (this *Merge) Returning() *Projection {
	return this.returning
}

func (this *Merge) Type() string {
	return "MERGE"
}

/*
Represents the merge source. Type MergeSource is a
struct containing three fields, the from keyspace
term, the select query and the alias as string.
*/
type MergeSource struct {
	from  *KeyspaceTerm   `json:"from"`
	query *SubqueryTerm   `json:"select"`
	expr  *ExpressionTerm `json:"expr"`
}

/*
The function NewMergeSourceFrom returns a pointer
to the MergeSource struct by assigning the input
attributes to the fields of the struct, setting
the from keyspace term and the alias.
*/
func NewMergeSourceFrom(from *KeyspaceTerm) *MergeSource {
	return &MergeSource{
		from: from,
	}
}

/*
The function NewMergeSourceSelect returns a pointer
to the MergeSource struct by assigning the input
attributes to the fields of the struct, setting
the query and the alias.
*/
func NewMergeSourceSubquery(query *SubqueryTerm) *MergeSource {
	return &MergeSource{
		query: query,
	}
}

/*
The function NewMergeSourceExpression returns a pointer
to the MergeSource struct by assigning the input
attributes to the fields of the struct, setting
the expr and the alias.
*/
func NewMergeSourceExpression(expr *ExpressionTerm) *MergeSource {
	return &MergeSource{
		expr: expr,
	}
}

/*
Applies mapper to the query expressions.
*/
func (this *MergeSource) MapExpressions(mapper expression.Mapper) (err error) {
	if this.query != nil {
		err = this.query.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.expr != nil {
		err = this.expr.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *MergeSource) Expressions() expression.Expressions {
	if this.query != nil {
		return this.query.Expressions()
	}

	if this.expr != nil {
		return this.expr.Expressions()
	}

	return nil
}

/*
Returns all required privileges.
*/
func (this *MergeSource) Privileges() (*auth.Privileges, errors.Error) {
	if this.from != nil {
		return this.from.Privileges()
	}

	if this.query != nil {
		return this.query.Privileges()
	}

	return this.expr.Privileges()
}

/*
Fully qualify identifiers for each of the constituent fields
in the merge source statement.
*/
func (this *MergeSource) Formalize() (f *expression.Formalizer, err error) {
	if this.from != nil {
		_, err = this.from.Formalize(expression.NewFormalizer("", nil))
		if err != nil {
			return
		}
	}

	if this.query != nil {
		_, err = this.query.Formalize(expression.NewFormalizer("", nil))
		if err != nil {
			return
		}
	}

	if this.expr != nil {
		_, err = this.expr.Formalize(expression.NewFormalizer("", nil))
		if err != nil {
			return
		}
		if this.expr.IsKeyspace() {
			this.from = this.expr.KeyspaceTerm()
			this.expr = nil
		}
	}

	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("MergeSource", "semantics.mergesource.requires_name_or_alias")
		return nil, err
	}

	f = expression.NewFormalizer(keyspace, nil)
	return
}

/*
Return the from keyspace term for the merge source.
*/
func (this *MergeSource) From() *KeyspaceTerm {
	return this.from
}

/*
Return the select query for the merge source.
*/
func (this *MergeSource) SubqueryTerm() *SubqueryTerm {
	return this.query
}

/*
Return the ExpressionTerm for the merge source.
*/
func (this *MergeSource) ExpressionTerm() *ExpressionTerm {
	return this.expr
}

/*
Return the alias for the merge source. If AS
is not specified return the from clause alias.
*/
func (this *MergeSource) Alias() string {
	if this.query != nil {
		return this.query.Alias()
	} else if this.from != nil {
		return this.from.Alias()
	} else if this.expr != nil {
		return this.expr.Alias()
	} else {
		return ""
	}
}

/*
Return the actual keyspace name for the merge source.
If not KeyspaceTerm return empty string
*/
func (this *MergeSource) Keyspace() string {
	if this.from != nil {
		return this.from.Keyspace()
	} else {
		return ""
	}
}

/*
Represents the merge actions in a merge statement. They
can be merge update, merge delete and merge insert.
*/
type MergeActions struct {
	update *MergeUpdate `json:"update"`
	delete *MergeDelete `json:"delete"`
	insert *MergeInsert `json:"insert"`
}

/*
The function NewMergeActions returns a pointer to the
MergeActions struct by assigning the input attributes
to the fields of the struct.
*/
func NewMergeActions(update *MergeUpdate, delete *MergeDelete, insert *MergeInsert) *MergeActions {
	return &MergeActions{
		update: update,
		delete: delete,
		insert: insert,
	}
}

/*
Returns all contained Expressions.
*/
func (this *MergeActions) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.update != nil {
		exprs = append(exprs, this.update.Expressions()...)
	}

	if this.delete != nil {
		exprs = append(exprs, this.delete.Expressions()...)
	}

	if this.insert != nil {
		exprs = append(exprs, this.insert.Expressions()...)
	}

	return exprs
}

func (this *MergeActions) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 16)

	if this.update != nil {
		exprs = append(exprs, this.update.NonMutatedExpressions()...)
	}

	if this.delete != nil {
		exprs = append(exprs, this.delete.Expressions()...)
	}

	if this.insert != nil {
		exprs = append(exprs, this.insert.Expressions()...)
	}

	return exprs
}

/*
Determine the privileges requires for the merge actions,
and add them to 'privs' (which will not be nil).
The keyspace being acted on is passed down as 'keyspace'.
*/
func (this *MergeActions) AddPrivilegesFor(privs *auth.Privileges, keyspace string) {
	if this.update != nil {
		privs.Add(keyspace, auth.PRIV_QUERY_UPDATE)
	}

	if this.delete != nil {
		privs.Add(keyspace, auth.PRIV_QUERY_DELETE)
	}

	if this.insert != nil {
		privs.Add(keyspace, auth.PRIV_QUERY_INSERT)
	}
}

/*
Apply mapper to the expressions in the merge update,delete and insert
statements.
*/
func (this *MergeActions) MapExpressions(mapper expression.Mapper) (err error) {
	if this.update != nil {
		err = this.update.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.delete != nil {
		err = this.delete.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.insert != nil {
		err = this.insert.MapExpressions(mapper)
	}

	return
}

/*
Returns the merge update merge action statement.
*/
func (this *MergeActions) Update() *MergeUpdate {
	return this.update
}

/*
Returns the merge delete merge action statement.
*/
func (this *MergeActions) Delete() *MergeDelete {
	return this.delete
}

/*
Returns the merge insert merge action statement.
*/
func (this *MergeActions) Insert() *MergeInsert {
	return this.insert
}

/*
Represents the merge update merge-actions statement.
Type MergeUpdate is a struct that contains the where
condition expression along with the set and unset
clause.
*/
type MergeUpdate struct {
	set   *Set                  `json:"set"`
	unset *Unset                `json:"unset"`
	where expression.Expression `json:"where"`
}

/*
The function NewMergeUpdate returns a pointer to the
MergeUpdate struct by assigning the input attributes
to the fields of the struct.
*/
func NewMergeUpdate(set *Set, unset *Unset, where expression.Expression) *MergeUpdate {
	return &MergeUpdate{set, unset, where}
}

/*
Applies mapper to the set, unset and where expressions.
*/
func (this *MergeUpdate) MapExpressions(mapper expression.Mapper) (err error) {
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
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *MergeUpdate) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)

	if this.set != nil {
		exprs = append(exprs, this.set.Expressions()...)
	}

	if this.unset != nil {
		exprs = append(exprs, this.unset.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	return exprs
}

func (this *MergeUpdate) NonMutatedExpressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 8)

	if this.set != nil {
		exprs = append(exprs, this.set.NonMutatedExpressions()...)
	}

	if this.unset != nil {
		exprs = append(exprs, this.unset.Expressions()...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	return exprs
}

/*
Fully qualify identifiers for each of the constituent fields
in the update action of merge statement.
*/
func (this *MergeUpdate) Formalize(f *expression.Formalizer) (err error) {
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
	}

	return
}

/*
Returns the set clause in a merge update merge-action
statement.
*/
func (this *MergeUpdate) Set() *Set {
	return this.set
}

/*
Returns the unset clause in a merge update
merge-action statement.
*/
func (this *MergeUpdate) Unset() *Unset {
	return this.unset
}

/*
Return the where clause exppression condition.
*/
func (this *MergeUpdate) Where() expression.Expression {
	return this.where
}

/*
Represents the merge delete merge actions statement.
Type MergeDelete is a struct that contains the where
condition expression.
*/
type MergeDelete struct {
	where expression.Expression `json:"where"`
}

/*
The function NewMergeDelete returns a pointer to the
MergeDelete struct by assigning the input attributes
to the fields of the struct.
*/
func NewMergeDelete(where expression.Expression) *MergeDelete {
	return &MergeDelete{where}
}

/*
Apply mapper to where condition expressions.
*/
func (this *MergeDelete) MapExpressions(mapper expression.Mapper) (err error) {
	if this.where != nil {
		this.where, err = mapper.Map(this.where)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *MergeDelete) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 1)

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	return exprs
}

/*
Fully qualify identifiers for each of the constituent fields
in the delete action of merge statement.
*/
func (this *MergeDelete) Formalize(f *expression.Formalizer) (err error) {
	if this.where != nil {
		this.where, err = f.Map(this.where)
	}

	return
}

/*
Return the where clause exppression condition.
*/
func (this *MergeDelete) Where() expression.Expression {
	return this.where
}

/*
Represents the merge insert merge actions statement.
Type MergeInsert is a struct that contains the value
and where condition expressions.
*/
type MergeInsert struct {
	key     expression.Expression `json:"key"`
	value   expression.Expression `json:"value"`
	options expression.Expression `json:"options"`
	where   expression.Expression `json:"where"`
}

/*
The function NewMergeInsert returns a pointer to the MergeInsert
struct by assigning the input attributes to the fields of the
struct.
*/
func NewMergeInsert(key, value, options, where expression.Expression) *MergeInsert {
	return &MergeInsert{key, value, options, where}
}

/*
Apply mapper to value and where expressions.
*/
func (this *MergeInsert) MapExpressions(mapper expression.Mapper) (err error) {
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

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *MergeInsert) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 3)

	if this.key != nil {
		exprs = append(exprs, this.key)
	}

	if this.value != nil {
		exprs = append(exprs, this.value)
	}

	if this.options != nil {
		exprs = append(exprs, this.options)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	return exprs
}

/*
Fully qualify identifiers for each of the constituent fields
in the insert action of merge statement.
*/
func (this *MergeInsert) Formalize(f *expression.Formalizer) (err error) {
	if this.key != nil {
		this.key, err = f.Map(this.key)
		if err != nil {
			return
		}
	}

	if this.value != nil {
		this.value, err = f.Map(this.value)
		if err != nil {
			return
		}
	}

	if this.options != nil {
		this.options, err = f.Map(this.options)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = f.Map(this.where)
	}

	return
}

/*
Return the merge insert key expression.
*/
func (this *MergeInsert) Key() expression.Expression {
	return this.key
}

/*
Return the merge insert value expression.
*/
func (this *MergeInsert) Value() expression.Expression {
	return this.value
}

/*
Return the merge insert options expression.
*/
func (this *MergeInsert) Options() expression.Expression {
	return this.options
}

/*
Return the where clause exppression condition.
*/
func (this *MergeInsert) Where() expression.Expression {
	return this.where
}
