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
	namespace string
	keyspace  string
	as        string
	keys      expression.Expression
	indexes   IndexRefs
}

func NewKeyspaceTerm(namespace, keyspace string, as string,
	keys expression.Expression, indexes IndexRefs) *KeyspaceTerm {
	return &KeyspaceTerm{namespace, keyspace, as, keys, indexes}
}

func (this *KeyspaceTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitKeyspaceTerm(this)
}

/*
This method maps all the constituent terms, namely keys in the FROM
clause.
*/
func (this *KeyspaceTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.keys != nil {
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
	if this.keys != nil {
		exprs = append(exprs, this.keys)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *KeyspaceTerm) Privileges() (datastore.Privileges, errors.Error) {
	privs := datastore.NewPrivileges()
	if this.namespace == "#system" {
		if this.keyspace == "user_info" {
			privs[this.namespace+":"+this.keyspace] = datastore.PRIV_SECURITY_READ
		} else {
			privs[this.namespace+":"+this.keyspace] = datastore.PRIV_SYSTEM_READ
		}
	} else {
		privs[this.namespace+":"+this.keyspace] = datastore.PRIV_READ
	}
	return privs, nil
}

/*
   Representation as a N1QL string.
*/
func (this *KeyspaceTerm) String() string {
	return this.toString(false)
}

/*
   Representation as a N1QL string.
*/
func (this *KeyspaceTerm) toString(join bool) string {
	s := ""

	if this.namespace != "" {
		s += "`" + this.namespace + "`:"
	}

	s += "`" + this.keyspace + "`"

	if this.as != "" {
		s += " as `" + this.as + "`"
	}

	if this.keys != nil {
		if join {
			s += " on keys " + this.keys.String()
		} else {
			s += " use keys " + this.keys.String()
		}
	}

	return s
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *KeyspaceTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewNoTermNameError("FROM", "plan.keyspace.requires_name_or_alias")
		return
	}

	f = expression.NewFormalizer("", parent)
	if this.keys != nil {
		_, err = this.keys.Accept(f)
		if err != nil {
			return
		}
	}

	_, ok := parent.Allowed().Field(keyspace)
	if ok {
		err = errors.NewDuplicateAliasError("subquery", keyspace, "plan.keyspace.duplicate_alias")
		return nil, err
	}

	f.SetAlias(this.As())
	f.SetKeyspace(keyspace)
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
		return this.keyspace
	}
}

/*
Returns the namespace string.
*/
func (this *KeyspaceTerm) Namespace() string {
	return this.namespace
}

/*
Set the namespace string when it is empty.
*/
func (this *KeyspaceTerm) SetDefaultNamespace(namespace string) {
	if this.namespace == "" {
		this.namespace = namespace
	}
}

/*
Returns the keyspace string (buckets).
*/
func (this *KeyspaceTerm) Keyspace() string {
	return this.keyspace
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
Marshals the input keyspace into a byte array.
*/
func (this *KeyspaceTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "keyspaceTerm"}
	r["as"] = this.as
	if this.keys != nil {
		r["keys"] = expression.NewStringer().Visit(this.keys)
	}
	r["namespace"] = this.namespace
	r["keyspace"] = this.keyspace
	return json.Marshal(r)
}
