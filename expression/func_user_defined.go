//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

func GetUserDefinedFunction(name functions.FunctionName, check bool) Function {

	// we fail to create a UDF expression if the UDF does not exist
	// this is for backwards compatibility with missing functions
	// and to avoid succesfully preparing statements with missing
	// functions
	// if the UDF gets dropped later, we fail on execution
	if check && !functions.PreLoad(name) {
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
		name: name,
	}
	rv.Init(name.Key(), operands...)

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

	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}

	parkableContext, ok := context.(ParkableContext)
	if !ok {
		return nil, errors.NewEvaluationError(fmt.Errorf("Casting context of type %T to ParkableContext failed.", context),
			this.name.Key())
	}

	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)

	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}

	return functions.ExecuteFunction(this.name, functions.READONLY, args, parkableContext)
}

func (this *UserDefinedFunction) EvaluateForIndex(item value.Value, context Context) (value.Value, value.Values, error) {
	args := _ARGS_POOL.GetSized(len(this.operands))
	defer _ARGS_POOL.Put(args)

	for i, op := range this.operands {
		var err error
		args[i], err = op.Evaluate(item, context)
		if err != nil {
			return nil, nil, err
		}
	}

	val, err := functions.ExecuteFunction(this.name, functions.READONLY+functions.INVARIANT, args, context.(ParkableContext))
	if err == nil {
		return val, nil, nil
	} else {
		return val, nil, fmt.Errorf("%s", err.Error())
	}
}

func (this *UserDefinedFunction) Indexable() bool {
	return functions.Indexable(this.name) != value.FALSE
}

// Full name of the function with appropriate backticks
func (this *UserDefinedFunction) ProtectedName() string {
	return this.name.ProtectedKey()
}

/*
Factory method pattern.
*/
func (this *UserDefinedFunction) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewUserDefinedFunction(this.name, operands...)
	}
}
