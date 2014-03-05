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

func (this *IfMissing) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfMissingOrNull) evaluate(args value.Values) (value.Value, error) {
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

func (this *IfNull) evaluate(args value.Values) (value.Value, error) {
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

func (this *MissingIf) evaluate(first, second value.Value) (value.Value, error) {
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

func (this *NullIf) evaluate(first, second value.Value) (value.Value, error) {
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
