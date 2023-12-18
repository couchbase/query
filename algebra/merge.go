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

	keyspace   *KeyspaceRef          `json:"keyspace"`
	indexes    IndexRefs             `json:"indexes"`
	source     *MergeSource          `json:"source"`
	on         expression.Expression `json:"on"`
	isOnKey    bool                  `json:"is_on_key"`
	actions    *MergeActions         `json:"actions"`
	limit      expression.Expression `json:"limit"`
	returning  *Projection           `json:"returning"`
	optimHints *OptimHints           `json:"optimizer_hints"`
	let        expression.Bindings   `json:"let"`
	extraPrivs *auth.Privileges      `json:"extra_privs"`
}

/*
The function NewMerge returns a pointer to the Merge
struct by assigning the input attributes to the fields
of the struct.
*/
func NewMerge(keyspace *KeyspaceRef, indexes IndexRefs, source *MergeSource,
	isOnKey bool, on expression.Expression, actions *MergeActions,
	limit expression.Expression, returning *Projection, optimHints *OptimHints, let expression.Bindings) *Merge {
	rv := &Merge{
		keyspace:   keyspace,
		indexes:    indexes,
		source:     source,
		on:         on,
		isOnKey:    isOnKey,
		actions:    actions,
		limit:      limit,
		returning:  returning,
		optimHints: optimHints,
		let:        let,
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

	if this.let != nil {
		err = this.let.MapExpressions(mapper)
		if err != nil {
			return
		}
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
	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
	}
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
	if this.let != nil {
		exprs = append(exprs, this.let.Expressions()...)
	}
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
		privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT, auth.PRIV_PROPS_NONE)
	}
	if this.extraPrivs != nil {
		privs.AddAll(this.extraPrivs)
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

func (this *Merge) SetExtraPrivs(ep *auth.Privileges) {
	this.extraPrivs = ep
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
		return errors.NewDuplicateAliasError("MERGE", kf.Keyspace(), this.source.ErrorContext(), "semantics.merge.duplicate_alias")
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

	if this.let != nil {
		err = f.PushBindings(this.let, false)
		if err != nil {
			return
		}
		for _, v := range this.let {
			kf.SetAllowedAlias(v.Variable(), false)
		}
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

func (this *Merge) KeyspaceRef() *KeyspaceRef {
	return this.keyspace
}

func (this *Merge) Indexes() IndexRefs {
	return this.indexes
}

func (this *Merge) Source() *MergeSource {
	return this.source
}

func (this *Merge) On() expression.Expression {
	return this.on
}

func (this *Merge) IsOnKey() bool {
	return this.isOnKey
}

func (this *Merge) Let() expression.Bindings {
	return this.let
}

func (this *Merge) Actions() *MergeActions {
	return this.actions
}

func (this *Merge) Limit() expression.Expression {
	return this.limit
}

func (this *Merge) Returning() *Projection {
	return this.returning
}

func (this *Merge) Type() string {
	return "MERGE"
}

func (this *Merge) String() string {
	s := "merge "
	if this.optimHints != nil {
		s += this.optimHints.String() + " "
	}
	s += "into "
	s += this.keyspace.Path().ProtectedString()
	alias := this.keyspace.Alias()
	if alias != "" {
		s += " as `" + alias + "`"
	}
	if this.indexes != nil {
		s += " use index(" + this.indexes.String() + ")"
	}
	s += " using " + this.source.String()
	if this.on != nil {
		if this.isOnKey {
			s += " on key " + this.on.String()
		} else {
			s += " on " + this.on.String()
		}
	}
	if this.let != nil {
		s += " let " + stringBindings(this.let)
	}
	if this.actions != nil {
		if this.actions.update != nil {
			s += " when matched then update"
			if this.actions.update.set != nil {
				s += " set"
				for _, v := range this.actions.update.set.Terms() {
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
			if this.actions.update.unset != nil {
				s += " unset"
				for _, v := range this.actions.update.unset.Terms() {
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
			if this.actions.update.where != nil {
				s += " where " + this.actions.update.where.String()
			}
		}
		if this.actions.delete != nil {
			s += " when matched then delete"
			if this.actions.delete.where != nil {
				s += " where " + this.actions.delete.where.String()
			}
		}
		if this.actions.insert != nil {
			s += " when not matched then insert"
			if this.isOnKey {
				s += this.actions.insert.value.String()
			} else {
				s += "(key " + this.actions.insert.key.String()
				if this.actions.insert.value != nil {
					s += ", value " + this.actions.insert.value.String()
				}
				if this.actions.insert.options != nil {
					s += ", options " + this.actions.insert.options.String()
				}
				s += ")"
			}
			if this.actions.insert.where != nil {
				s += " where " + this.actions.insert.where.String()
			}
		}
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
Optimier hints
*/
func (this *Merge) OptimHints() *OptimHints {
	return this.optimHints
}

func (this *Merge) SetOptimHints(optimHints *OptimHints) {
	this.optimHints = optimHints
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

func (this *MergeSource) ErrorContext() string {
	if this.from != nil {
		return this.from.ErrorContext()
	} else if this.query != nil {
		return this.from.ErrorContext()
	} else if this.expr != nil {
		return this.expr.ErrorContext()
	}
	return ""
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
		err = errors.NewNoTermNameError("MergeSource", this.ErrorContext(), "semantics.mergesource.requires_name_or_alias")
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

func (this *MergeSource) String() string {
	if this.query != nil {
		return this.query.String()
	} else if this.from != nil {
		return this.from.Path().ProtectedString()
	} else if this.expr != nil {
		return this.expr.String()
	}
	return ""
}

// MergeSource.Keyspace() no longer needed, as we use paths instead

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
func (this *MergeActions) AddPrivilegesFor(privs *auth.Privileges, fullKeyspace string) {
	if this.update != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_UPDATE, auth.PRIV_PROPS_NONE)
	}

	if this.delete != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_DELETE, auth.PRIV_PROPS_NONE)
	}

	if this.insert != nil {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_INSERT, auth.PRIV_PROPS_NONE)
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
