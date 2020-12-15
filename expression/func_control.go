//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"fmt"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Abort
//
///////////////////////////////////////////////////

/*
this represents programmatically cancelling a request
*/
type Abort struct {
	UnaryFunctionBase
}

func NewAbort(operand Expression) Function {
	rv := &Abort{
		*NewUnaryFunctionBase("abort", operand),
	}
	rv.setVolatile()

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Abort) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Abort) Type() value.Type {
	return value.JSON
}

func (this *Abort) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Abort) Apply(context Context, arg value.Value) (value.Value, error) {
	return value.NULL_VALUE, errors.NewAbortError(fmt.Sprintf("%v", arg))
}

/*
Factory method pattern.
*/
func (this *Abort) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAbort(operands[0])
	}
}
