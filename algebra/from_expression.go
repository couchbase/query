//  Copyright (c) 2016 Couchbase, Inc.
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

type ExpressionTerm struct {
	fromExpr     expression.Expression
	as           string
	keyspaceTerm *KeyspaceTerm
	isKeyspace   bool
}

/*
Constructor.
*/
func NewExpressionTerm(fromExpr expression.Expression, as string,
	keyspaceTerm *KeyspaceTerm) *ExpressionTerm {
	return &ExpressionTerm{fromExpr: fromExpr, as: as, keyspaceTerm: keyspaceTerm}
}

/*
Visitor pattern.
*/
func (this *ExpressionTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitExpressionTerm(this)
}

/*
Apply mapping to all contained Expressions.
*/
func (this *ExpressionTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.isKeyspace {
		return this.keyspaceTerm.MapExpressions(mapper)
	} else {
		this.fromExpr, _ = mapper.Map(this.fromExpr)
	}
	return nil
}

/*
   Returns all contained Expressions.
*/
func (this *ExpressionTerm) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, 0, 2)

	if this.isKeyspace {
		exprs = append(exprs, this.keyspaceTerm.Expressions()...)
	} else {
		exprs = append(exprs, this.fromExpr)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *ExpressionTerm) Privileges() (datastore.Privileges, errors.Error) {
	if this.isKeyspace {
		return this.keyspaceTerm.Privileges()
	}
	return datastore.NewPrivileges(), nil
}

/*
   Representation as a N1QL string.
*/
func (this *ExpressionTerm) String() string {
	if this.isKeyspace {
		return this.keyspaceTerm.String()
	} else if this.as != "" {
		return this.fromExpr.String() + " as " + this.as
	} else {
		return this.fromExpr.String()
	}
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *ExpressionTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	if this.keyspaceTerm != nil {
		_, ok := parent.Aliases().Field(this.keyspaceTerm.Keyspace())
		this.isKeyspace = !ok
	}

	if this.isKeyspace {
		return this.keyspaceTerm.Formalize(parent)
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewNoTermNameError("FROM expression", "plan.fromExpr.requires_name_or_alias")
		return nil, err
	}

	_, ok := parent.Allowed().Field(alias)
	if ok {
		err = errors.NewDuplicateAliasError("FROM expression", alias, "plan.fromExpr.duplicate_alias")
		return nil, err
	}

	if this.keyspaceTerm != nil && this.KeyspaceTerm().Keys() != nil {
		err = errors.NewUseKeysError("FROM expression", "plan.fromExpr.usekeys_are_not_alllowed")
		return nil, err
	}

	f = expression.NewFormalizer("", parent)
	this.fromExpr, err = f.Map(this.fromExpr)
	if err != nil {
		return
	}

	f.Allowed().SetField(alias, alias)
	f.SetAlias(this.as)
	return
}

/*
Return the primary term in the from clause.
*/
func (this *ExpressionTerm) PrimaryTerm() FromTerm {
	return this
}

/*
Returns the Alias string.
*/
func (this *ExpressionTerm) Alias() string {
	if this.isKeyspace {
		return this.keyspaceTerm.Alias()
	} else if this.as != "" {
		return this.as
	} else {
		return this.fromExpr.Alias()
	}
}

/*
Returns the from Expression
*/
func (this *ExpressionTerm) ExpressionTerm() expression.Expression {
	return this.fromExpr
}

/*
Returns the Keyspace Term
*/
func (this *ExpressionTerm) KeyspaceTerm() *KeyspaceTerm {
	return this.keyspaceTerm
}

/*
Returns the if Expression is Keyspace
*/
func (this *ExpressionTerm) IsKeyspace() bool {
	return this.isKeyspace
}

/*
Marshals input ExpressionTerm.
*/
func (this *ExpressionTerm) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "ExpressionTerm"}
	r["as"] = this.as
	r["fromexpr"] = this.fromExpr
	return json.Marshal(r)
}
