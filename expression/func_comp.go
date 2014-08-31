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

///////////////////////////////////////////////////
//
// Greatest
//
///////////////////////////////////////////////////

type Greatest struct {
	FunctionBase
}

func NewGreatest(operands ...Expression) Function {
	return &Greatest{
		*NewFunctionBase("greatest", operands...),
	}
}

func (this *Greatest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Greatest) Type() value.Type { return value.JSON }

func (this *Greatest) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Greatest) Apply(context Context, args ...value.Value) (value.Value, error) {
	rv := value.NULL_VALUE
	for _, a := range args {
		if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if a.Collate(rv) > 0 {
			rv = a
		}
	}

	return rv, nil
}

func (this *Greatest) Constructor() FunctionConstructor { return NewGreatest }

///////////////////////////////////////////////////
//
// Least
//
///////////////////////////////////////////////////

type Least struct {
	FunctionBase
}

func NewLeast(operands ...Expression) Function {
	return &Least{
		*NewFunctionBase("least", operands...),
	}
}

func (this *Least) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Least) Type() value.Type { return value.JSON }

func (this *Least) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Least) Apply(context Context, args ...value.Value) (value.Value, error) {
	rv := value.NULL_VALUE

	for _, a := range args {
		if a.Type() <= value.NULL {
			continue
		} else if rv == value.NULL_VALUE {
			rv = a
		} else if a.Collate(rv) < 0 {
			rv = a
		}
	}

	return rv, nil
}

func (this *Least) Constructor() FunctionConstructor { return NewLeast }
