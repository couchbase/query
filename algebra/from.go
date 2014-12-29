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
	"fmt"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

/*
Represents the from clause in a select statement.
*/
type FromTerm interface {
	/*
	   Represents the Node interface.
	*/
	Node

	/*
	   Apply a Mapper to all the expressions in this statement
	*/
	MapExpressions(mapper expression.Mapper) error

	/*
	   Qualify all identifiers for the parent expression.
	*/
	Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error)

	/*
	   Represents the primary from term.
	*/
	PrimaryTerm() FromTerm

	/*
	   Represents alias string.
	*/
	Alias() string
}

/*
Represents the Keyspace (bucket) term in the from clause.
The keyspace can be prefixed with an optional namespace
(pool).

Nested paths can be specified. For each document
in the keyspace the path is evaluated and its value becomes
an input to the query. If any element of the path is NULL
or missing, the document is skipped and does not contribute
to the query.

The Alias for the from clause is specified using the AS
keyword.

Specific primary keys within a keyspace can be specified.
Only values having those primary keys will be included as
inputs to the query.

Type KeyspaceTerm is a struct that contains namespace,
keyspace as strings, projection as the path, the alias
string as, and the keys expression.
*/
type KeyspaceTerm struct {
	namespace  string
	keyspace   string
	projection expression.Path
	as         string
	keys       expression.Expression
}

/*
The function NewKeyspaceTerm returns a pointer to the KeyspaceTerm
struct by assigning the input attributes to the fields of the struct.
*/
func NewKeyspaceTerm(namespace, keyspace string, projection expression.Path, as string, keys expression.Expression) *KeyspaceTerm {
	return &KeyspaceTerm{namespace, keyspace, projection, as, keys}
}

/*
It calls the VisitKeyspaceTerm method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *KeyspaceTerm) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitKeyspaceTerm(this)
}

/*
This method maps all the constituent terms, namely projection and keys
in the from clause.
*/
func (this *KeyspaceTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.projection != nil {
		expr, err := mapper.Map(this.projection)
		if err != nil {
			return err
		}

		this.projection = expr.(expression.Path)
	}

	if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return err
		}
	}

	return
}

/*
Qualify all identifiers for the parent expression. Checks for
duplicate aliases.
*/
func (this *KeyspaceTerm) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewError(nil, "FROM term must have a name or alias.")
		return
	}

	if this.keys != nil {
		_, err = this.keys.Accept(parent)
		if err != nil {
			return
		}
	}

	_, ok := parent.Allowed.Field(keyspace)
	if ok {
		err = errors.NewError(nil, fmt.Sprintf("Duplicate subquery alias %s.", keyspace))
		return nil, err
	}

	allowed := value.NewScopeValue(make(map[string]interface{}), parent.Allowed)
	allowed.SetField(keyspace, keyspace)

	f = expression.NewFormalizer()
	f.Keyspace = keyspace
	f.Allowed = allowed
	return
}

/*
Return the primary term in the from clause.
*/
func (this *KeyspaceTerm) PrimaryTerm() FromTerm {
	return this
}

/*
Returns the Alias string. If as is not empty then return it.
If it is not set, then check the path (projection) and return
its alias, otherwise return the keyspace string.
*/
func (this *KeyspaceTerm) Alias() string {
	if this.as != "" {
		return this.as
	} else if this.projection != nil {
		return this.projection.Alias()
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
Returns the path (projection expression).
*/
func (this *KeyspaceTerm) Projection() expression.Path {
	return this.projection
}

/*
Returns the alias string.
*/
func (this *KeyspaceTerm) As() string {
	return this.as
}

/*
Returns the keys expression.
*/
func (this *KeyspaceTerm) Keys() expression.Expression {
	return this.keys
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
	if this.projection != nil {
		r["projection"] = expression.NewStringer().Visit(this.projection)
	}
	return json.Marshal(r)
}

/*
Represents the join clause. Joins create new input
objects by combining two or more source objects.
They can be chained. Type Join is a struct containing
fields left and right that represent the two source
objects being joined (one is a from term and the
other a keyspace term), and outer which is a bool
value representing if the join is an outer or inner
join.
*/
type Join struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

/*
The function NewJoin returns a pointer to the Join struct
by assigning the input attributes to the fields of the struct.
*/
func NewJoin(left FromTerm, outer bool, right *KeyspaceTerm) *Join {
	return &Join{left, right, outer}
}

/*
It calls the VisitJoin method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Join) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

/*
Maps left and right source objects of the join.
*/
func (this *Join) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
Qualify all identifiers for the parent expression. Checks is
a join alias exists and if it is a duplicate alias.
*/
func (this *Join) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	f.Keyspace = ""
	this.right.keys, err = f.Map(this.right.keys)
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewError(nil, "JOIN term must have a name or alias.")
		return nil, err
	}

	_, ok := f.Allowed.Field(alias)
	if ok {
		err = errors.NewError(nil, fmt.Sprintf("Duplicate JOIN alias %s.", alias))
		return nil, err
	}

	f.Allowed.SetField(alias, alias)
	return
}

/*
Returns the primary term in the left source of
the join.
*/
func (this *Join) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the alias of the right source.
*/
func (this *Join) Alias() string {
	return this.right.Alias()
}

/*
Returns the left source object of the join.
*/
func (this *Join) Left() FromTerm {
	return this.left
}

/*
Returns the right source object of the join.
*/
func (this *Join) Right() *KeyspaceTerm {
	return this.right
}

/*
Returns boolean value based on if it is
an outer or inner join.
*/
func (this *Join) Outer() bool {
	return this.outer
}

/*
Marshals input join terms.
*/
func (this *Join) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "join"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	return json.Marshal(r)
}

/*
Nesting is conceptually the inverse of unnesting. Nesting
performs a join across two keyspaces (or a keyspace with
itself). But instead of producing a cross-product of the
left and right hand inputs, a single result is produced
for each left hand input, while the corresponding right
hand inputs are collected into an array and nested as a
single array-valued field in the result object. Type
Nest is a struct containing the left hand input (from term)
the right hand keyspace and a boolean outer representing
if outer or inner nest.
*/
type Nest struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

/*
The function NewNest returns a pointer to the Nest struct
by assigning the input attributes to the fields of the struct.
*/
func NewNest(left FromTerm, outer bool, right *KeyspaceTerm) *Nest {
	return &Nest{left, right, outer}
}

/*
It calls the VisitNest method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Nest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

/*
Maps the right input of the nest if the left is mapped
successfully.
*/
func (this *Nest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

/*
Qualify all identifiers for the parent expression. Checks is
a nest alias exists and if it is a duplicate alias.
*/
func (this *Nest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	f.Keyspace = ""
	this.right.keys, err = f.Map(this.right.keys)
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewError(nil, "NEST term must have a name or alias.")
		return nil, err
	}

	_, ok := f.Allowed.Field(alias)
	if ok {
		err = errors.NewError(nil, fmt.Sprintf("Duplicate NEST alias %s.", alias))
		return nil, err
	}

	f.Allowed.SetField(alias, alias)
	return
}

/*
Return the primary term in the left term of the nest clause.
*/
func (this *Nest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the nest alias of the right source.
*/
func (this *Nest) Alias() string {
	return this.right.Alias()
}

/*
Returns the left term in the nest clause.
*/
func (this *Nest) Left() FromTerm {
	return this.left
}

/*
Returns the right term in the nest clause.
*/
func (this *Nest) Right() *KeyspaceTerm {
	return this.right
}

/*
Returns a boolean value depending on if it is
an outer or inner nest.
*/
func (this *Nest) Outer() bool {
	return this.outer
}

/*
Marshals input nest terms into byte array.
*/
func (this *Nest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "nest"}
	r["left"] = this.left
	r["right"] = this.right
	r["outer"] = this.outer
	return json.Marshal(r)
}

/*
If a document or object contains a nested array, UNNEST
conceptually performs a join of the nested array with
its parent object. Each resulting joined object becomes
an input to the query.

Type Unnest is a struct containing fields left representing
the from term (parent object), the expr that is the source
nested array, the alias (as) and outer which represents if
it is an outer or inner unnest.
*/
type Unnest struct {
	left  FromTerm
	outer bool
	expr  expression.Expression
	as    string
}

/*
The function NewUnnest returns a pointer to the Unnest struct
by assigning the input attributes to the fields of the struct.
*/
func NewUnnest(left FromTerm, outer bool, expr expression.Expression, as string) *Unnest {
	return &Unnest{left, outer, expr, as}
}

/*
It calls the VisitUnnest method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Unnest) Accept(visitor NodeVisitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

/*
Maps the source array of the unnest if the parent object(left)
is mapped successfully.
*/
func (this *Unnest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.expr, err = mapper.Map(this.expr)
	return
}

/*
Qualify all identifiers for the parent expression. Checks is
a unnest alias exists and if it is a duplicate alias.
*/
func (this *Unnest) Formalize(parent *expression.Formalizer) (f *expression.Formalizer, err error) {
	f, err = this.left.Formalize(parent)
	if err != nil {
		return
	}

	this.expr, err = f.Map(this.expr)
	if err != nil {
		return
	}

	alias := this.Alias()
	if alias == "" {
		err = errors.NewError(nil, "UNNEST term must have a name or alias.")
		return nil, err
	}

	_, ok := f.Allowed.Field(alias)
	if ok {
		err = errors.NewError(nil, fmt.Sprintf("Duplicate UNNEST alias %s.", alias))
		return nil, err
	}

	f.Keyspace = ""
	f.Allowed.SetField(alias, alias)
	return
}

/*
Return the primary term in the parent object
(left term) of the unnest clause.
*/
func (this *Unnest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

/*
Returns the unnest alias if set. Else returns the alias of
the input nested array.
*/
func (this *Unnest) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.expr.Alias()
	}
}

/*
Returns the left term (parent object) in the unnest
clause.
*/
func (this *Unnest) Left() FromTerm {
	return this.left
}

/*
Returns a boolean value depending on if it is
an outer or inner unnest.
*/
func (this *Unnest) Outer() bool {
	return this.outer
}

/*
Returns the source array object path expression for
the unnest clause.
*/
func (this *Unnest) Expression() expression.Expression {
	return this.expr
}

/*
Returns the alias string in an unnest clause.
*/
func (this *Unnest) As() string {
	return this.as
}

/*
Marshals input unnest terms into byte array.
*/
func (this *Unnest) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "unnest"}
	r["left"] = this.left
	r["as"] = this.as
	r["outer"] = this.outer
	if this.expr != nil {
		r["expr"] = expression.NewStringer().Visit(this.expr)
	}
	return json.Marshal(r)
}
