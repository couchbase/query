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

type IfMissing struct {
	nAryBase
}

func NewIfMissing(args Expressions) Function {
	return &IfMissing{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfMissing) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IfMissing) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IfMissing) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IfMissing) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IfMissing) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IfMissing) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IfMissing) eval(args value.Values) (value.Value, error) {
	for _, a := range args {
		if a.Type() != value.MISSING {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfMissing) MinArgs() int { return 2 }

func (this *IfMissing) Constructor() FunctionConstructor { return NewIfMissing }

type IfMissingOrNull struct {
	nAryBase
}

func NewIfMissingOrNull(args Expressions) Function {
	return &IfMissingOrNull{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfMissingOrNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IfMissingOrNull) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IfMissingOrNull) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IfMissingOrNull) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IfMissingOrNull) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IfMissingOrNull) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IfMissingOrNull) eval(args value.Values) (value.Value, error) {
	for _, a := range args {
		if a.Type() > value.NULL {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfMissingOrNull) MinArgs() int { return 2 }

func (this *IfMissingOrNull) Constructor() FunctionConstructor { return NewIfMissingOrNull }

type IfNull struct {
	nAryBase
}

func NewIfNull(args Expressions) Function {
	return &IfNull{
		nAryBase{
			operands: args,
		},
	}
}

func (this *IfNull) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *IfNull) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *IfNull) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *IfNull) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *IfNull) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *IfNull) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *IfNull) eval(args value.Values) (value.Value, error) {
	for _, a := range args {
		if a.Type() != value.NULL {
			return a, nil
		}
	}

	return value.NULL_VALUE, nil
}

func (this *IfNull) MinArgs() int { return 2 }

func (this *IfNull) Constructor() FunctionConstructor { return NewIfNull }

type MissingIf struct {
	binaryBase
}

func NewMissingIf(first, second Expression) Function {
	return &MissingIf{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *MissingIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *MissingIf) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *MissingIf) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *MissingIf) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *MissingIf) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *MissingIf) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *MissingIf) eval(first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.MISSING_VALUE, nil
	} else {
		return first, nil
	}
}

func (this *MissingIf) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewMissingIf(args[0], args[1])
	}
}

type NullIf struct {
	binaryBase
}

func NewNullIf(first, second Expression) Function {
	return &NullIf{
		binaryBase{
			first:  first,
			second: second,
		},
	}
}

func (this *NullIf) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *NullIf) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *NullIf) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *NullIf) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *NullIf) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *NullIf) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *NullIf) eval(first, second value.Value) (value.Value, error) {
	if first.Equals(second) {
		return value.NULL_VALUE, nil
	} else {
		return first, nil
	}
}

func (this *NullIf) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewNullIf(args[0], args[1])
	}
}
