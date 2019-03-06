//  Copyright (c) 2019 Couchbase, Inc.
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
	"math"

	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

func GetUserDefinedFunction(name functions.FunctionName) Function {

	// we fail to create a UDF expression if the UDF does not exist
	// this is for backwards compatibility with missing functions
	// and to avoid succesfully preparing statements with missing
	// functions
	// if the UDF gets dropped later, we fail on execution
	if !functions.PreLoad(name) {
		return nil
	}
	return &UserDefinedFunction{name: name}
}

///////////////////////////////////////////////////
//
// UDF
//
///////////////////////////////////////////////////

/*
This represents the execution of UDFs
*/
type UserDefinedFunction struct {
	UserDefinedFunctionBase
	name functions.FunctionName
}

func NewUserDefinedFunction(name functions.FunctionName, operands ...Expression) Function {
	rv := &UserDefinedFunction{
		*NewUserDefinedFunctionBase(name.Key(), operands...),
		name,
	}

	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *UserDefinedFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *UserDefinedFunction) Type() value.Type {
	return value.JSON
}

func (this *UserDefinedFunction) MinArgs() int { return 0 }

func (this *UserDefinedFunction) MaxArgs() int { return math.MaxInt16 }

func (this *UserDefinedFunction) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *UserDefinedFunction) EvaluateForIndex(item value.Value, context Context) (value.Value, value.Values, error) {
	val, err := this.EvalForIndex(this, item, context)
	return val, nil, err
}

func (this *UserDefinedFunction) Apply(context Context, args ...value.Value) (value.Value, error) {
	val, err := functions.ExecuteFunction(this.name, functions.READONLY, args, context)
	if err == nil {
		return val, nil
	} else {
		return val, fmt.Errorf(err.Error())
	}
}

func (this *UserDefinedFunction) IdxApply(context Context, args ...value.Value) (value.Value, error) {
	val, err := functions.ExecuteFunction(this.name, functions.READONLY+functions.INVARIANT, args, context)
	if err == nil {
		return val, nil
	} else {
		return val, fmt.Errorf(err.Error())
	}
}

func (this *UserDefinedFunction) Indexable() bool {
	return functions.Indexable(this.name) != value.FALSE
}

/*
Factory method pattern.
*/
func (this *UserDefinedFunction) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUserDefinedFunction(this.name, operands...)
	}
}
