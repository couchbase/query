//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Aggregate function  var_samp(expr)/variance_samp(expr). It returns
the arithmetic sample variance of all the number values in the
group. Type VarSamp is a struct that inherits from AggregateBase.
*/
type VarSamp struct {
	AggregateBase
}

/*
The function NewVarSamp calls NewAggregateBase to
create an aggregate function named VarSamp with
one expression as input.
*/
func NewVarSamp(operand expression.Expression) Aggregate {
	rv := &VarSamp{
		*NewAggregateBase("var_samp", operand),
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *VarSamp) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *VarSamp) Type() value.Type {
	return value.NUMBER
}

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *VarSamp) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewVarSamp with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *VarSamp) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewVarSamp(operands[0])
	}
}

/*
If no input to the VarSamp function, then the default value
returned is a null.
*/
func (this *VarSamp) Default() value.Value {
	return value.NULL_VALUE
}

/*
Aggregates input data by evaluating operands.
For all values other than Number, return the input value itself.
Maintain two variables for sum and
list of all the values of type NUMBER.
Call addStddevVariance to compute the intermediate aggregate value and return it.
*/
func (this *VarSamp) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	item, e := this.Operand().Evaluate(item, context)
	if e != nil {
		return nil, e
	}

	if item.Type() != value.NUMBER {
		return cumulative, nil
	}

	return addStddevVariance(item, cumulative, false)
}

/*
Aggregates intermediate results and return them.
*/
func (this *VarSamp) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulateStddevVariance(part, cumulative, false)
}

/*
Compute the sample variance as the final.
Return NULL if no values of type NUMBER exist.
Return NULL if only one value exists.
calculate sample variance according to definition
and return it.
*/
func (this *VarSamp) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	if cumulative == value.NULL_VALUE {
		return cumulative, nil
	}

	variance, e := computeVariance(cumulative, false, true, 1.0)
	if e != nil {
		return nil, e
	}

	return variance, nil
}
