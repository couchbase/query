//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"github.com/couchbaselabs/query/value"
)

type IsArray struct {
	unaryBase
}

func NewIsArray(arg Expression) Function {
	return &IsArray{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsArray) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsArray) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsArray) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsArray) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsArray) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsArray) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsArray) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.ARRAY), nil
}

func (this *IsArray) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsArray(args[0])
	}
}

type IsAtom struct {
	unaryBase
}

func NewIsAtom(arg Expression) Function {
	return &IsAtom{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsAtom) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsAtom) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsAtom) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsAtom) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsAtom) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsAtom) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsAtom) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	switch arg.Type() {
	case value.BOOLEAN, value.NUMBER, value.STRING:
		return value.NewValue(true), nil
	default:
		return value.NewValue(false), nil
	}
}

func (this *IsAtom) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsAtom(args[0])
	}
}

type IsBool struct {
	unaryBase
}

func NewIsBool(arg Expression) Function {
	return &IsBool{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsBool) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsBool) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsBool) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsBool) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsBool) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsBool) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsBool) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.BOOLEAN), nil
}

func (this *IsBool) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsBool(args[0])
	}
}

type IsNum struct {
	unaryBase
}

func NewIsNum(arg Expression) Function {
	return &IsNum{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsNum) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsNum) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsNum) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsNum) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsNum) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsNum) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsNum) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.NUMBER), nil
}

func (this *IsNum) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsNum(args[0])
	}
}

type IsObj struct {
	unaryBase
}

func NewIsObj(arg Expression) Function {
	return &IsObj{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsObj) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsObj) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsObj) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsObj) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsObj) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsObj) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsObj) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.OBJECT), nil
}

func (this *IsObj) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsObj(args[0])
	}
}

type IsStr struct {
	unaryBase
}

func NewIsStr(arg Expression) Function {
	return &IsStr{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *IsStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IsStr) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IsStr) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IsStr) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IsStr) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IsStr) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IsStr) eval(arg value.Value) (value.Value, error) {
	if arg.Type() <= value.NULL {
		return arg, nil
	}

	return value.NewValue(arg.Type() == value.STRING), nil
}

func (this *IsStr) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewIsStr(args[0])
	}
}

type TypeName struct {
	unaryBase
}

func NewTypeName(arg Expression) Function {
	return &TypeName{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *TypeName) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *TypeName) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *TypeName) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *TypeName) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *TypeName) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *TypeName) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *TypeName) eval(arg value.Value) (value.Value, error) {
	tn, _ := value.TypeName(arg.Type())
	return value.NewValue(tn), nil
}

func (this *TypeName) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewTypeName(args[0])
	}
}
