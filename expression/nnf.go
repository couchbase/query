//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

type NNF struct {
	MapperBase
}

func NewNNF() *NNF {
	rv := &NNF{}
	rv.mapper = rv
	return rv
}

func (this *NNF) MapBindings() bool { return false }

func (this *NNF) VisitIn(expr *In) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	a, ok := expr.Second().(*ArrayConstruct)
	if !ok {
		return expr, nil
	}

	first := expr.First()
	operands := make(Expressions, len(a.Operands()))
	for i, op := range a.Operands() {
		operands[i] = NewEq(first, op)
	}

	return NewOr(operands...), nil
}

func (this *NNF) VisitBetween(expr *Between) (interface{}, error) {
	err := expr.MapChildren(this)
	if err != nil {
		return nil, err
	}

	return NewAnd(NewGE(expr.First(), expr.Second()),
		NewLE(expr.First(), expr.Third())), nil
}

func (this *NNF) VisitNot(expr *Not) (interface{}, error) {
	var exp Expression = expr

	switch operand := expr.Operand().(type) {
	case *Not:
		exp = operand.Operand()
	case *And:
		operands := make(Expressions, len(operand.Operands()))
		for i, op := range operand.Operands() {
			operands[i] = NewNot(op)
		}

		exp = NewOr(operands...)
	case *Or:
		operands := make(Expressions, len(operand.Operands()))
		for i, op := range operand.Operands() {
			operands[i] = NewNot(op)
		}

		exp = NewAnd(operands...)
	}

	return exp, exp.MapChildren(this)
}
