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
	"fmt"

	"github.com/couchbase/query/expression"
)

/*
This represents the value pairs used in an insert
values clause in an insert or upsert statement. It
contains multiple key value Pairs.
*/
type Pairs []*Pair

/*
Type Pair is a struct that contains key and value
expressions.
*/
type Pair struct {
	Key   expression.Expression
	Value expression.Expression
}

/*
Applies mapper to the key and value expressions.
*/
func (this *Pair) MapExpressions(mapper expression.Mapper) (err error) {
	this.Key, err = mapper.Map(this.Key)
	if err != nil {
		return
	}

	this.Value, err = mapper.Map(this.Value)
	return
}

/*
Returns all contained Expressions.
*/
func (this *Pair) Expressions() expression.Expressions {
	return expression.Expressions{this.Key, this.Value}
}

/*
Creates and returns a new array construct containing
the key value pair.
*/
func (this *Pair) Expression() expression.Expression {
	return expression.NewArrayConstruct(this.Key, this.Value)
}

/*
Calls MarshalJSON on the expression returned by Expression().
*/
func (this *Pair) MarshalJSON() ([]byte, error) {
	return this.Expression().MarshalJSON()
}

/*
Applies mapper to multiple key-value pairs.
*/
func (this Pairs) MapExpressions(mapper expression.Mapper) (err error) {
	for _, pair := range this {
		err = pair.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this Pairs) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, len(this)*2)

	for i, pair := range this {
		exprs[i*2] = pair.Key
		exprs[i*2+1] = pair.Value
	}

	return exprs
}

/*
Creates and returns a new array construct containing
the all the key value rxpression pair's in Pairs.
*/
func (this Pairs) Expression() expression.Expression {
	exprs := make(expression.Expressions, len(this))

	for i, pair := range this {
		exprs[i] = pair.Expression()
	}

	return expression.NewArrayConstruct(exprs...)
}

/*
Calls MarshalJSON on the expression returned by Expression().
*/
func (this Pairs) MarshalJSON() ([]byte, error) {
	return this.Expression().MarshalJSON()
}

/*
Range over the operands of the input array construct
and create new key value pair's using the NewPair()
method and add it to Pairs. Return.
*/
func NewPairs(array *expression.ArrayConstruct) (pairs Pairs, err error) {
	operands := array.Operands()
	pairs = make(Pairs, len(operands))
	for i, op := range operands {
		pairs[i], err = NewPair(op)
		if err != nil {
			return nil, err
		}
	}

	return
}

/*
Create a key value pair using the operands of the input
expression Array construct and return.
*/
func NewPair(expr expression.Expression) (*Pair, error) {
	array, ok := expr.(*expression.ArrayConstruct)
	if !ok {
		return nil, fmt.Errorf("Invalid VALUES expression %s", expr.String())
	}

	operands := array.Operands()
	if len(operands) != 2 {
		return nil, fmt.Errorf("Invalid VALUES expression %s", expr.String())
	}

	pair := &Pair{
		Key:   operands[0],
		Value: operands[1],
	}

	return pair, nil
}
