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
	"bytes"
	"math"

	"github.com/couchbaselabs/query/value"
)

type Concat struct {
	FunctionBase
}

func NewConcat(operands ...Expression) Function {
	return &Concat{
		*NewFunctionBase("concat", operands...),
	}
}

func (this *Concat) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitConcat(this)
}

func (this *Concat) Type() value.Type { return value.STRING }

func (this *Concat) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Concat) Apply(context Context, args ...value.Value) (value.Value, error) {
	var buf bytes.Buffer
	null := false

	for _, arg := range args {
		switch arg.Type() {
		case value.STRING:
			if !null {
				buf.WriteString(arg.Actual().(string))
			}
		case value.MISSING:
			return value.MISSING_VALUE, nil
		default:
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	}

	return value.NewValue(buf.String()), nil
}

func (this *Concat) MinArgs() int { return 2 }

func (this *Concat) MaxArgs() int { return math.MaxInt16 }

func (this *Concat) Constructor() FunctionConstructor { return NewConcat }
