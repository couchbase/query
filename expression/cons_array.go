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

type ArrayConstruct struct {
	FunctionBase
}

func NewArrayConstruct(operands ...Expression) Function {
	return &ArrayConstruct{
		*NewFunctionBase("array", operands...),
	}
}

func (this *ArrayConstruct) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitArrayConstruct(this)
}

func (this *ArrayConstruct) Type() value.Type { return value.ARRAY }

func (this *ArrayConstruct) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *ArrayConstruct) Apply(context Context, args ...value.Value) (value.Value, error) {
	aa := make([]interface{}, len(args))

	for i, arg := range args {
		aa[i] = arg.Actual()
	}

	return value.NewValue(aa), nil
}

func (this *ArrayConstruct) Constructor() FunctionConstructor { return NewArrayConstruct }
