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

func NewObjectKeys(operand Expression) Function {
	return &ObjectKeys{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ObjectKeys) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().(map[string]interface{})
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

func NewObjectLength(operand Expression) Function {
	return &ObjectLength{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ObjectLength) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().(map[string]interface{})
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

func NewObjectValues(operand Expression) Function {
	return &ObjectValues{
		unaryBase{
			operand: operand,
		},
	}
}

func (this *ObjectValues) evaluate(operand value.Value) (value.Value, error) {
	if operand.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if operand.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := operand.Actual().(map[string]interface{})
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
