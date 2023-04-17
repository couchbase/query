//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
)

/*
Represents the keyspace_ref used in DML statements
*/
type KeyspaceRef struct {
	path *Path                 `json:"path"`
	expr expression.Expression `json:"expr"`
	as   string                `json:"as"`
}

func NewKeyspaceRefFromPath(path *Path, as string) *KeyspaceRef {
	return &KeyspaceRef{path, nil, as}
}

func NewKeyspaceRefFromExpression(expr expression.Expression, as string) *KeyspaceRef {
	return &KeyspaceRef{nil, expr, as}
}

func NewKeyspaceRefWithContext(keyspace, as, namespace, queryContext string) *KeyspaceRef {
	return &KeyspaceRef{NewPathWithContext(keyspace, namespace, queryContext), nil, as}
}

/*
Qualify identifiers for the keyspace. It also makes sure that the
keyspace term contains a name or alias.
*/
func (this *KeyspaceRef) Formalize() (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("Keyspace", "", "semantics.keyspace.reference_requires_name_or_alias")
		return
	}

	f = expression.NewFormalizer(keyspace, nil)
	return
}

func (this *KeyspaceRef) Path() *Path {
	return this.path
}

func (this *KeyspaceRef) ExpressionTerm() expression.Expression {
	return this.expr
}

/*
Returns the namespace string.
*/
func (this *KeyspaceRef) Namespace() string {
	if this.path != nil {
		return this.path.Namespace()
	}
	return ""
}

/*
Is this pointing to the system store?
*/
func (this *KeyspaceRef) IsSystem() bool {
	return this.path != nil && this.path.IsSystem()
}

/*
Set the default namespace.
FIXME ideally this should go
*/
func (this *KeyspaceRef) SetDefaultNamespace(namespace string) {
	if this.path != nil {
		this.path.SetDefaultNamespace(namespace)
	}
}

/*
Returns the keyspace string.
*/
func (this *KeyspaceRef) Keyspace() string {
	if this.path != nil {
		return this.path.Keyspace()
	}
	return this.expr.String()
}

/*
Returns the AS alias string.
*/
func (this *KeyspaceRef) As() string {
	return this.as
}

/*
Returns the alias as the keyspace or the as string
based on if as is empty.
*/
func (this *KeyspaceRef) Alias() string {
	if this.as != "" {
		return this.as
	} else if this.path != nil {
		return this.path.Alias()
	} else {
		return ""
	}
}

/*
Marshals input into byte array.
*/
func (this *KeyspaceRef) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	if this.path != nil {
		r["path"] = this.path
	} else {
		r["expr"] = this.expr
	}
	if this.as != "" {
		r["as"] = this.as
	}

	return json.Marshal(r)
}

func (this *KeyspaceRef) MarshalKeyspace(m map[string]interface{}) {
	if this.path != nil {
		this.path.marshalKeyspace(m)
	} else {
		m["expr"] = this.expr
	}
}

/*
Returns the full keyspace name, including the namespace.
*/
func (this *KeyspaceRef) FullName() string {
	if this.path != nil {
		return this.path.SimpleString()
	}
	return this.expr.String()
}

func (this *KeyspaceRef) PrivilegeProps() int {
	if this.path != nil {
		return auth.PRIV_PROPS_NONE
	} else {
		return auth.PRIV_PROPS_DYNAMIC_TARGET
	}
}
