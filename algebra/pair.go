//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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
	key     expression.Expression
	value   expression.Expression
	options expression.Expression
}

func NewPair(key, value, options expression.Expression) *Pair {
	return &Pair{
		key:     key,
		value:   value,
		options: options,
	}
}

/* This function is Object map (key:value)
 * options are not valid. ignore it
 */

func MapPairs(pairs Pairs) map[expression.Expression]expression.Expression {
	mapping := make(map[expression.Expression]expression.Expression, len(pairs))

	for _, pair := range pairs {
		mapping[pair.key] = pair.value
	}

	return mapping
}

/*
Applies mapper to the key and value expressions.
*/
func (this *Pair) MapExpressions(mapper expression.Mapper) (err error) {
	this.key, err = mapper.Map(this.key)

	if err == nil && this.value != nil {
		this.value, err = mapper.Map(this.value)
	}

	if err == nil && this.options != nil {
		this.options, err = mapper.Map(this.options)
	}

	return
}

/*
Returns all contained Expressions.
*/
func (this *Pair) Expressions() (exprs expression.Expressions) {
	exprs = make(expression.Expressions, 0, 3)
	exprs = append(exprs, this.key)

	if this.value != nil {
		exprs = append(exprs, this.value)
	}

	if this.options != nil {
		exprs = append(exprs, this.options)
	}

	return
}

func (this *Pair) Key() expression.Expression {
	return this.key
}

func (this *Pair) Value() expression.Expression {
	return this.value
}

func (this *Pair) Options() expression.Expression {
	return this.options
}

/*
Creates and returns a new array construct containing
the key value pair.
*/
func (this *Pair) Expression() expression.Expression {

	return expression.NewArrayConstruct(this.Expressions()...)
}

/*
Calls MarshalJSON on the expression returned by Expression().
*/
func (this *Pair) MarshalJSON() ([]byte, error) {
	return this.Expression().MarshalJSON()
}

/*
Applies mapper to multiple key-value-options pairs.
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
	exprs := make(expression.Expressions, 0, len(this)*3)

	for _, pair := range this {
		exprs = append(exprs, pair.key)
		if pair.value != nil {
			exprs = append(exprs, pair.value)
		}

		if pair.options != nil {
			exprs = append(exprs, pair.options)
		}
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
func NewValuesPairs(array *expression.ArrayConstruct) (pairs Pairs, err error) {
	operands := array.Operands()
	pairs = make(Pairs, len(operands))
	for i, op := range operands {
		pairs[i], err = NewValuesPair(op)
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
func NewValuesPair(expr expression.Expression) (*Pair, error) {
	if array, ok := expr.(*expression.ArrayConstruct); ok {
		operands := array.Operands()
		switch len(operands) {
		case 1:
			return &Pair{key: operands[0]}, nil
		case 2:
			return &Pair{key: operands[0], value: operands[1]}, nil
		case 3:
			return &Pair{key: operands[0], value: operands[1], options: operands[2]}, nil
		}
	}

	return nil, fmt.Errorf("Invalid VALUES expression %s", expr.String())
}
