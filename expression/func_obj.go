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

///////////////////////////////////////////////////
//
// ObjectKeys
//
///////////////////////////////////////////////////

type ObjectKeys struct {
	UnaryFunctionBase
}

func NewObjectKeys(operand Expression) Function {
	return &ObjectKeys{
		*NewUnaryFunctionBase("object_keys", operand),
	}
}

func (this *ObjectKeys) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectKeys) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ObjectKeys) Apply(context Context, arg value.Value) (value.Value, error) {
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
	return func(operands ...Expression) Function {
		return NewObjectKeys(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectLength
//
///////////////////////////////////////////////////

type ObjectLength struct {
	UnaryFunctionBase
}

func NewObjectLength(operand Expression) Function {
	return &ObjectLength{
		*NewUnaryFunctionBase("object_length", operand),
	}
}

func (this *ObjectLength) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectLength) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ObjectLength) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if arg.Type() != value.OBJECT {
		return value.NULL_VALUE, nil
	}

	oa := arg.Actual().(map[string]interface{})
	return value.NewValue(float64(len(oa))), nil
}

func (this *ObjectLength) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewObjectLength(operands[0])
	}
}

///////////////////////////////////////////////////
//
// ObjectValues
//
///////////////////////////////////////////////////

type ObjectValues struct {
	UnaryFunctionBase
}

func NewObjectValues(operand Expression) Function {
	return &ObjectValues{
		*NewUnaryFunctionBase("object_values", operand),
	}
}

func (this *ObjectValues) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ObjectValues) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *ObjectValues) Apply(context Context, arg value.Value) (value.Value, error) {
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
	return func(operands ...Expression) Function {
		return NewObjectValues(operands[0])
	}
}
