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
	"fmt"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type FromTerm interface {
	Node
	MapExpressions(mapper expression.Mapper) error
	Formalize() (f *Formalizer, err error)
	PrimaryTerm() FromTerm
	Alias() string
}

type KeyspaceTerm struct {
	namespace string
	keyspace  string
	project   expression.Path
	as        string
	keys      expression.Expression
}

func NewKeyspaceTerm(namespace, keyspace string, project expression.Path, as string, keys expression.Expression) *KeyspaceTerm {
	return &KeyspaceTerm{namespace, keyspace, project, as, keys}
}

func (this *KeyspaceTerm) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitKeyspaceTerm(this)
}

func (this *KeyspaceTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if this.project != nil {
		expr, err := mapper.Map(this.project)
		if err != nil {
			return err
		}

		this.project = expr.(expression.Path)
	}

	if this.keys != nil {
		this.keys, err = mapper.Map(this.keys)
		if err != nil {
			return err
		}
	}

	return
}

func (this *KeyspaceTerm) Formalize() (f *Formalizer, err error) {
	keyspace := this.Alias()
	if keyspace == "" {
		err = errors.NewError(nil, "FROM term must have a name or alias.")
		return
	}

	if this.keys != nil {
		_, err = this.keys.Accept(expression.EMPTY_FORMALIZER)
		if err != nil {
			return
		}
	}

	allowed := value.NewValue(make(map[string]interface{}))
	allowed.SetField(keyspace, keyspace)

	f = NewFormalizer()
	f.Keyspace = keyspace
	f.Allowed = allowed
	return
}

func (this *KeyspaceTerm) PrimaryTerm() FromTerm {
	return this
}

func (this *KeyspaceTerm) Alias() string {
	if this.as != "" {
		return this.as
	} else if this.project != nil {
		return this.project.Alias()
	} else {
		return this.keyspace
	}
}

func (this *KeyspaceTerm) Namespace() string {
	return this.namespace
}

func (this *KeyspaceTerm) Keyspace() string {
	return this.keyspace
}

func (this *KeyspaceTerm) Project() expression.Path {
	return this.project
}

func (this *KeyspaceTerm) As() string {
	return this.as
}

func (this *KeyspaceTerm) Keys() expression.Expression {
	return this.keys
}

type Join struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

func NewJoin(left FromTerm, outer bool, right *KeyspaceTerm) *Join {
	return &Join{left, right, outer}
}

func (this *Join) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitJoin(this)
}

func (this *Join) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

func (this *Join) Formalize() (f *Formalizer, err error) {
	f, err = this.left.Formalize()
	if err != nil {
		return
	}

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

func (this *Join) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Join) Alias() string {
	return this.right.Alias()
}

func (this *Join) Left() FromTerm {
	return this.left
}

func (this *Join) Right() *KeyspaceTerm {
	return this.right
}

func (this *Join) Outer() bool {
	return this.outer
}

type Nest struct {
	left  FromTerm
	right *KeyspaceTerm
	outer bool
}

func NewNest(left FromTerm, outer bool, right *KeyspaceTerm) *Nest {
	return &Nest{left, right, outer}
}

func (this *Nest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitNest(this)
}

func (this *Nest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	return this.right.MapExpressions(mapper)
}

func (this *Nest) Formalize() (f *Formalizer, err error) {
	f, err = this.left.Formalize()
	if err != nil {
		return
	}

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

func (this *Nest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Nest) Alias() string {
	return this.right.Alias()
}

func (this *Nest) Left() FromTerm {
	return this.left
}

func (this *Nest) Right() *KeyspaceTerm {
	return this.right
}

func (this *Nest) Outer() bool {
	return this.outer
}

type Unnest struct {
	left  FromTerm
	outer bool
	expr  expression.Expression
	as    string
}

func NewUnnest(left FromTerm, outer bool, expr expression.Expression, as string) *Unnest {
	return &Unnest{left, outer, expr, as}
}

func (this *Unnest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUnnest(this)
}

func (this *Unnest) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.left.MapExpressions(mapper)
	if err != nil {
		return
	}

	this.expr, err = mapper.Map(this.expr)
	return
}

func (this *Unnest) Formalize() (f *Formalizer, err error) {
	f, err = this.left.Formalize()
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

	f.Allowed.SetField(alias, alias)
	return
}

func (this *Unnest) PrimaryTerm() FromTerm {
	return this.left.PrimaryTerm()
}

func (this *Unnest) Alias() string {
	if this.as != "" {
		return this.as
	} else {
		return this.expr.Alias()
	}
}

func (this *Unnest) Left() FromTerm {
	return this.left
}

func (this *Unnest) Outer() bool {
	return this.outer
}

func (this *Unnest) Expression() expression.Expression {
	return this.expr
}

func (this *Unnest) As() string {
	return this.as
}
