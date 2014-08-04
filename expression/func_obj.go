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
	"sort"

	"github.com/couchbaselabs/query/value"
)

type ObjectKeys struct {
	unaryBase
}

func NewObjectKeys(arg Expression) Function {
	return &ObjectKeys{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ObjectKeys) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ObjectKeys) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ObjectKeys) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ObjectKeys) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ObjectKeys) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ObjectKeys) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ObjectKeys) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	keys := make(sort.StringSlice, 0, len(oa))
	for key, _ := range oa {
		keys = append(keys, key)
	}

	sort.Sort(keys)
	ra := make([]interface{}, len(keys))
	for i, k := range keys {
		ra[i] = k
	}

	return value.NewValue(ra), nil
}

func (this *ObjectKeys) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewObjectKeys(args[0])
	}
}

type ObjectLength struct {
	unaryBase
}

func NewObjectLength(arg Expression) Function {
	return &ObjectLength{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ObjectLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ObjectLength) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ObjectLength) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ObjectLength) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ObjectLength) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ObjectLength) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ObjectLength) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	return value.NewValue(float64(len(oa))), nil
}

func (this *ObjectLength) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewObjectLength(args[0])
	}
}

type ObjectValues struct {
	unaryBase
}

func NewObjectValues(arg Expression) Function {
	return &ObjectValues{
		unaryBase{
			operand: arg,
		},
	}
}

func (this *ObjectValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

func (this *ObjectValues) EquivalentTo(other Expression) bool {
	return this.equivalentTo(this, other)
}

func (this *ObjectValues) Fold() (Expression, error) {
	return this.fold(this)
}

func (this *ObjectValues) Formalize(forbidden, allowed value.Value, keyspace string) (Expression, error) {
	return this.formalize(this, forbidden, allowed, keyspace)
}

func (this *ObjectValues) SubsetOf(other Expression) bool {
	return this.subsetOf(this, other)
}

func (this *ObjectValues) VisitChildren(visitor Visitor) (Expression, error) {
	return this.visitChildren(this, visitor)
}

func (this *ObjectValues) eval(arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	keys := make(sort.StringSlice, 0, len(oa))
	for key, _ := range oa {
		keys = append(keys, key)
	}

	sort.Sort(keys)
	ra := make([]interface{}, len(keys))
	for i, k := range keys {
		ra[i] = oa[k]
	}

	return value.NewValue(ra), nil
}

func (this *ObjectValues) Constructor() FunctionConstructor {
	return func(args Expressions) Function {
		return NewObjectValues(args[0])
	}
}
