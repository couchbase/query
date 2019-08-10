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
	"fmt"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
This represents the Window NTILE() function.
It divides an ordered data set into a number of buckets indicated by expr
and assigns the appropriate number to each row. The buckets are numbered 1 through expr.
    The buckets represents the number of buckets indicated by expr
    The nrows represents the total number of rows in the partition
    The cbucket represents the current bucket number inuse.
    The cMaxRow represents the current bucket number can be used until cMaxRow.
*/

type Ntile struct {
	AggregateBase
	buckets int64
	nrows   int64
	cbucket int64
	cMaxRow int64
}

/*
The function NewNtile calls NewAggregateBase to
create an aggregate function named Ntile
*/

func NewNtile(operands expression.Expressions, flags uint32, wTerm *WindowTerm) Aggregate {
	rv := &Ntile{
		*NewAggregateBase("ntile", operands, flags, wTerm), 0, 0, 0, 0,
	}

	rv.SetExpr(rv)
	return rv
}

/*
It calls the VisitFunction method by passing in the receiver to
and returns the interface. It is a visitor pattern.
*/
func (this *Ntile) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

/*
It returns a value of type NUMBER.
*/
func (this *Ntile) Type() value.Type { return value.NUMBER }

/*
Calls the evaluate method for aggregate functions and passes in the
receiver, current item and current context.
*/
func (this *Ntile) Evaluate(item value.Value, context expression.Context) (result value.Value, e error) {
	return this.evaluate(this, item, context)
}

/*
The constructor returns a NewNtile with the input operand
cast to a Function as the FunctionConstructor.
*/
func (this *Ntile) Constructor() expression.FunctionConstructor {
	return func(operands ...expression.Expression) expression.Function {
		return NewNtile(operands, uint32(0), nil)
	}
}

/*
Copy of the aggregate function
*/

func (this *Ntile) Copy() expression.Expression {
	rv := &Ntile{
		*NewAggregateBase(this.Name(), expression.CopyExpressions(this.Operands()),
			this.Flags(), CopyWindowTerm(this.WindowTerm())),
		this.buckets, this.nrows, this.cbucket, this.cMaxRow,
	}

	rv.BaseCopy(this)
	rv.SetExpr(rv)
	return rv
}

/*
If no input to the Ntile function, then the default value
returned is a null.
*/
func (this *Ntile) Default(item value.Value, context Context) (value.Value, error) {
	return value.NULL_VALUE, nil
}

/*
The some of input data part of the window attachment in item.
*/

func (this *Ntile) CumulateInitial(item, cumulative value.Value, context Context) (value.Value, error) {
	part, err := getWindowAttachment(item, this.Name())
	if err != nil || part == nil {
		return nil, fmt.Errorf("Invalid %s %v of type %T.", this.Name(), part, part)
	}

	return this.cumulatePart(item, part, context)
}

/*
Aggregates intermediate results and return them.
*/
func (this *Ntile) CumulateIntermediate(part, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Aggregates Remove NOOP and return same input them.
*/

func (this *Ntile) CumulateRemove(item, cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
Returns input cumulative value as the Final result.
*/
func (this *Ntile) ComputeFinal(cumulative value.Value, context Context) (value.Value, error) {
	return cumulative, nil
}

/*
The part field in the items attachment contains the current row in the partition.
The nrows field in the items attachment contains the totla number of rows in the partition.
When current row is 0 then
       calculate number of buckets. And
       NOTE: expr depends on document it uses from the first row of partition for evaluation
             and uses for that value for all the rows in the partition.
When current row reaches cMaxRow calculate next cMaxRow and increament current cbucket
Returns cbucket as NTILE value.
Example: PARTITION has 26 rows and NTILE(4)
         Returns: Rows  0-6   1   7rows
                  Rows  7-13  2   7rows
                  Rows 14-19  3   6rows
                  Rows 20-25  4   6rows
*/

func (this *Ntile) cumulatePart(item, part value.Value, context Context) (value.Value, error) {
	cv, _ := part.Field("part")

	if cv.Type() != value.NUMBER {
		return nil, fmt.Errorf("%s internal Missing or invalid values: %v.", this.Name(), cv.Actual())
	}

	c := value.AsNumberValue(cv).Int64()
	if c == 0 {
		bv, e := this.Operands()[0].Evaluate(item, context)
		if e != nil || bv.Type() != value.NUMBER {
			return value.NULL_VALUE, e
		}
		this.buckets = value.AsNumberValue(bv).Int64()
		if this.buckets <= 0 {
			return nil, fmt.Errorf("%s Invalid argument: %v.", this.Name(), bv.Actual())
		}

		nrowsv, _ := part.Field("nrows")
		if nrowsv.Type() != value.NUMBER {
			return nil, fmt.Errorf("%s internal Missing or invalid values: %v.", this.Name(), nrowsv.Actual())
		}
		this.nrows = value.AsNumberValue(nrowsv).Int64()
		this.cbucket = 0
		this.cMaxRow = 0
	}

	if c == this.cMaxRow {
		this.cMaxRow += (this.nrows / this.buckets)
		if this.cbucket < (this.nrows % this.buckets) {
			this.cMaxRow++
		}
		this.cbucket++
	}

	return value.NewValue(this.cbucket), nil
}
