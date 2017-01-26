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
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// HasToken
//
///////////////////////////////////////////////////

type HasToken struct {
	FunctionBase
}

func NewHasToken(operands ...Expression) Function {
	rv := &HasToken{
		*NewFunctionBase("has_token", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *HasToken) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *HasToken) Type() value.Type { return value.BOOLEAN }

func (this *HasToken) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *HasToken) Apply(context Context, args ...value.Value) (value.Value, error) {
	source := args[0]
	token := args[1]

	if source.Type() == value.MISSING || token.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if source.Type() == value.NULL || token.Type() == value.NULL {
		return value.NULL_VALUE, nil
	}

	options := _EMPTY_OPTIONS
	if len(args) >= 3 {
		switch args[2].Type() {
		case value.OBJECT:
			options = args[2]
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			return value.NULL_VALUE, nil
		}
	}

	set := _SET_POOL.Get()
	defer _SET_POOL.Put(set)
	set = source.Tokens(set, options)
	return value.NewValue(set.Has(token)), nil
}

func (this *HasToken) MinArgs() int { return 2 }

func (this *HasToken) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *HasToken) Constructor() FunctionConstructor {
	return NewHasToken
}

///////////////////////////////////////////////////
//
// Tokens
//
///////////////////////////////////////////////////

/*
MB-20850. Enumerate list of all tokens within the operand. For
strings, this is the list of discrete words within the string. For all
other atomic JSON values, it is the operand itself. For arrays, all
the individual array elements are tokenized. And for objects, the
names are included verbatim, while the values are tokenized.
*/
type Tokens struct {
	FunctionBase
}

func NewTokens(operands ...Expression) Function {
	rv := &Tokens{
		*NewFunctionBase("tokens", operands...),
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Tokens) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Tokens) Type() value.Type { return value.ARRAY }

func (this *Tokens) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Tokens) Apply(context Context, args ...value.Value) (value.Value, error) {
	arg := args[0]
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	options := _EMPTY_OPTIONS
	if len(args) >= 2 {
		switch args[1].Type() {
		case value.OBJECT:
			options = args[1]
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			return value.NULL_VALUE, nil
		}
	}

	set := _SET_POOL.Get()
	defer _SET_POOL.Put(set)
	set = arg.Tokens(set, options)
	items := set.Items()
	return value.NewValue(items), nil
}

func (this *Tokens) MinArgs() int { return 1 }

func (this *Tokens) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Tokens) Constructor() FunctionConstructor {
	return NewTokens
}

var _EMPTY_OPTIONS = value.NewValue(map[string]interface{}{})
