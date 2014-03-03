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
	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type NowMillis struct {
	expression.ExpressionBase
}

func NewNowMillis() expression.Function {
	return &NowMillis{}
}

func (this *NowMillis) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	nanos := context.(Context).Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *NowMillis) MinArgs() int { return 0 }

func (this *NowMillis) MaxArgs() int { return 0 }

func (this *NowMillis) Constructor() expression.FunctionConstructor {
	return func(expression.Expressions) expression.Function { return this }
}

type NowStr struct {
	expression.ExpressionBase
}

func NewNowStr() expression.Function {
	return &NowStr{}
}

func (this *NowStr) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	str := context.(Context).Now().String()
	return value.NewValue(str), nil
}

func (this *NowStr) MinArgs() int { return 0 }

func (this *NowStr) MaxArgs() int { return 0 }

func (this *NowStr) Constructor() expression.FunctionConstructor {
	return func(expression.Expressions) expression.Function { return this }
}
