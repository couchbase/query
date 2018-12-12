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

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/* property flags shared among SimpleFromTerm, not all terms use all flags */
const (
	TERM_ANSI_JOIN       = 1 << iota // right-hand side of ANSI JOIN
	TERM_ANSI_NEST                   // right-hand side of ANSI NEST
	TERM_PRIMARY_JOIN                // join on primary key (meta().id)
	TERM_UNDER_NL                    // inner side of nested-loop join
	TERM_UNDER_HASH                  // right-hand side of Hash Join
	TERM_INDEX_JOIN_NEST             // right-hand side of index join/nest
)

/*
Represents the Keyspace (bucket) term in the FROM clause.  The
keyspace can be prefixed with an optional namespace (pool).

Nested paths can be specified. For each document in the keyspace the
path is evaluated and its value becomes an input to the query. If any
element of the path is NULL or missing, the document is skipped and
does not contribute to the query.

The alias for the FROM clause is specified using the AS keyword.

Specific primary keys within a keyspace can be specified.  Only values
having those primary keys will be included as inputs to the query.
*/
type KeyspaceTerm struct {
	path     *Path
	as       string
	keys     expression.Expression
	indexes  IndexRefs
	joinKeys expression.Expression
	joinHint JoinHint
	property uint32
}

func NewKeyspaceTerm(namespace, keyspace string, as string,
	keys expression.Expression, indexes IndexRefs) *KeyspaceTerm {
	return &KeyspaceTerm{NewPathShort(namespace, keyspace), as, keys, indexes, nil, JOIN_HINT_NONE, 0}
}

func NewKeyspaceTermFromPath(path *Path, as string,
	keys expression.Expression, indexes IndexRefs) *KeyspaceTerm {
	return &KeyspaceTerm{path, as, keys, indexes, nil, JOIN_HINT_NONE, 0}
}

func (this *KeyspaceTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitKeyspaceTerm(this)
}

/*
This method maps all the constituent terms, namely keys in the FROM
clause.
*/
func (this *KeyspaceTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.joinKeys != nil {
		this.joinKeys, err = mapper.Map(this.joinKeys)
		if err != nil {
			return err
		}
	} else if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return err
		}
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this *KeyspaceTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 1)
	if this.joinKeys != nil {
		exprs = append(exprs, this.joinKeys)
	} else if this.keys != nil {
		exprs = append(exprs, this.keys)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *KeyspaceTerm) Privileges() (*auth.Privileges, errors.Error) {
	privs, err := privilegesFromPath(this.path)
	if err != nil {
		return nil, err
	}
	if this.joinKeys != nil {
		privs.AddAll(this.joinKeys.Privileges())
	} else if this.keys != nil {
		privs.AddAll(this.keys.Privileges())
	}
	return privs, nil
}

func privilegesFromPath(path *Path) (*auth.Privileges, errors.Error) {
	namespace := path.Namespace()
	var bucket string
	if path.IsCollection() {
		// Use permissions of bucket for collection.
		// JTODO: This should actually allow collection-level permissions.
		bucket = path.Bucket()
	} else {
		bucket = path.Keyspace()
	}
	privs := auth.NewPrivileges()
	fullKeyspace := namespace + ":" + bucket
	if namespace == "#system" {
		switch bucket {
		case "user_info", "applicable_roles":
			privs.Add(fullKeyspace, auth.PRIV_SECURITY_READ)
		case "keyspaces", "indexes", "my_user_info":
			// Do nothing. These tables handle security internally, by
			// filtering the results.
		case "datastores", "namespaces", "dual":
			// Do nothing. These three tables are open to all.
		default:
			privs.Add(fullKeyspace, auth.PRIV_SYSTEM_READ)
		}
	} else {
		privs.Add(fullKeyspace, auth.PRIV_QUERY_SELECT)
	}
	return privs, nil
}

/*
   Representation as a N1QL string.
*/
func (this *KeyspaceTerm) String() string {
	s := this.path.ProtectedString()

	if this.as != "" {
		s += " as `" + this.as + "`"
	}

	if this.joinKeys != nil {
		if this.IsIndexJoinNest() {
			s += " on key " + this.joinKeys.String()
		} else {
			s += " on keys " + this.joinKeys.String()
		}
	} else if this.keys != nil {
		s += " use keys " + this.keys.String()
	}

	// since use keys cannot be mixed with join hints, we can safely add the "use" keyword
	switch this.joinHint {
	case USE_HASH_BUILD:
		s += " use hash(build)"
	case USE_HASH_PROBE:
		s += " use hash(probe)"
	case USE_NL:
		s += " use nl"
	}

	return s
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *KeyspaceTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	var errString string
	keyspace := this.Alias()
	if keyspace == "" {
		if this.IsAnsiJoin() {
			errString = "JOIN"
		} else if this.IsAnsiNest() {
			errString = "NEST"
		} else {
			errString = "FROM"
		}
		err = errors.NewNoTermNameError(errString, "semantics.keyspace.requires_name_or_alias")
		return
	}

	if this.IsAnsiJoinOp() {
		f = parent
	} else {
		f = expression.NewFormalizer("", parent)
	}

	if this.joinKeys != nil {
		_, err = this.joinKeys.Accept(f)
		if err != nil {
			return
		}
	} else if this.keys != nil {
		_, err = this.keys.Accept(f)
		if err != nil {
			return
		}
	}

	_, ok := parent.Allowed().Field(keyspace)
	if ok {
		if this.IsAnsiJoin() {
			errString = "JOIN"
		} else if this.IsAnsiNest() {
			errString = "NEST"
		} else {
			errString = "subquery"
		}
		err = errors.NewDuplicateAliasError(errString, keyspace, "semantics.keyspace.duplicate_alias")
		return nil, err
	}

	if this.IsAnsiJoinOp() {
		f.SetKeyspace("")
		f.SetAllowedAlias(keyspace, true)
		f.SetAlias(this.As())
	} else {
		f.SetAlias(this.As())
		f.SetKeyspace(keyspace)
	}
	return
}

/*
Return the primary term in the FROM clause.
*/
func (this *KeyspaceTerm) PrimaryTerm() FromTerm {
	return this
}

/*
Returns the alias string.
*/
func (this *KeyspaceTerm) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.path.Alias()
	}
}

/*
Returns the namespace string.
*/
func (this *KeyspaceTerm) Namespace() string {
	return this.path.Namespace()
}

/*
Set the namespace string when it is empty.
*/
func (this *KeyspaceTerm) SetDefaultNamespace(namespace string) {
	this.path.SetDefaultNamespace(namespace)
}

/*
Returns the keyspace string (buckets).
*/
func (this *KeyspaceTerm) Keyspace() string {
	return this.path.Keyspace()
}

/*
Returns the explicit alias.
*/
func (this *KeyspaceTerm) As() string {
	return this.as
}

/*
Returns the keys expression defined by the USE KEYS
clause.
*/
func (this *KeyspaceTerm) Keys() expression.Expression {
	return this.keys
}

/*
Returns the indexes defined by the USE INDEX clause.
*/
func (this *KeyspaceTerm) Indexes() IndexRefs {
	return this.indexes
}

/*
Returns the join keys expression defined by the ON KEYS
or ON KEY ... FOR ... clause.
*/
func (this *KeyspaceTerm) JoinKeys() expression.Expression {
	return this.joinKeys
}

/*
Returns the join hint (USE HASH or USE NL).
*/
func (this *KeyspaceTerm) JoinHint() JoinHint {
	return this.joinHint
}

/*
Join hint prefers hash join
*/
func (this *KeyspaceTerm) PreferHash() bool {
	return this.joinHint == USE_HASH_BUILD || this.joinHint == USE_HASH_PROBE
}

/*
Join hint prefers nested loop join
*/
func (this *KeyspaceTerm) PreferNL() bool {
	return this.joinHint == USE_NL
}

/*
Returns the property.
*/
func (this *KeyspaceTerm) Property() uint32 {
	return this.property
}

/*
Returns whether this keyspace is for an ANSI JOIN
*/
func (this *KeyspaceTerm) IsAnsiJoin() bool {
	return (this.property & TERM_ANSI_JOIN) != 0
}

/*
Returns whether this keyspace is for an ANSI NEST
*/
func (this *KeyspaceTerm) IsAnsiNest() bool {
	return (this.property & TERM_ANSI_NEST) != 0
}

/*
Returns whether this keyspace is for an ANSI JOIN or ANSI NEST
*/
func (this *KeyspaceTerm) IsAnsiJoinOp() bool {
	return (this.property & (TERM_ANSI_JOIN | TERM_ANSI_NEST)) != 0
}

/*
Returns whether joining on primary key (meta().id)
*/
func (this *KeyspaceTerm) IsPrimaryJoin() bool {
	if this.IsAnsiJoinOp() {
		return (this.property & TERM_PRIMARY_JOIN) != 0
	} else {
		return false
	}
}

/*
Returns whether under inner of nested-loop join
*/
func (this *KeyspaceTerm) IsUnderNL() bool {
	return (this.property & TERM_UNDER_NL) != 0
}

/*
Returns whether this keyspace is being considered for Hash Join
*/
func (this *KeyspaceTerm) IsUnderHash() bool {
	return (this.property & TERM_UNDER_HASH) != 0
}

/*
Returns whether it's right-hand side of index join/nest
*/
func (this *KeyspaceTerm) IsIndexJoinNest() bool {
	return (this.property & TERM_INDEX_JOIN_NEST) != 0
}

/*
Set join keys
*/
func (this *KeyspaceTerm) SetJoinKeys(keys expression.Expression) {
	this.joinKeys = keys
}

/*
Set join hint
*/
func (this *KeyspaceTerm) SetJoinHint(joinHint JoinHint) {
	this.joinHint = joinHint
}

/*
Set property
*/
func (this *KeyspaceTerm) SetProperty(property uint32) {
	this.property = property
}

/*
Set ANSI JOIN property
*/
func (this *KeyspaceTerm) SetAnsiJoin() {
	this.property |= TERM_ANSI_JOIN
}

/*
Set ANSI NEST property
*/
func (this *KeyspaceTerm) SetAnsiNest() {
	this.property |= TERM_ANSI_NEST
}

/*
Set PRIMARY JOIN property
*/
func (this *KeyspaceTerm) SetPrimaryJoin() {
	if this.IsAnsiJoinOp() {
		this.property |= TERM_PRIMARY_JOIN
	}
}

/*
Set UNDER NL property
*/
func (this *KeyspaceTerm) SetUnderNL() {
	this.property |= TERM_UNDER_NL
}

/*
Unset UNDER NL property
*/
func (this *KeyspaceTerm) UnsetUnderNL() {
	this.property &^= TERM_UNDER_NL
}

/*
Set UNDER HASH property
*/
func (this *KeyspaceTerm) SetUnderHash() {
	this.property |= TERM_UNDER_HASH
}

/*
Unset UNDER HASH property
*/
func (this *KeyspaceTerm) UnsetUnderHash() {
	this.property &^= TERM_UNDER_HASH
}

/*
Set INDEX JOIN/NEST property
*/
func (this *KeyspaceTerm) SetIndexJoinNest() {
	this.property |= TERM_INDEX_JOIN_NEST
}

/*
Marshals the input keyspace into a byte array.
*/
func (this *KeyspaceTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "keyspaceTerm"}
	r["as"] = this.as
	if this.joinKeys != nil {
		r["keys"] = expression.NewStringer().Visit(this.joinKeys)
	} else if this.keys != nil {
		r["keys"] = expression.NewStringer().Visit(this.keys)
	}
	r["path"] = this.path
	return json.Marshal(r)
}

func (this *KeyspaceTerm) Path() *Path {
	return this.path
}
