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
	"reflect"

	"github.com/couchbaselabs/query/value"
)

// Commutative and associative operators.
type caNAry interface {
	nAry
	construct(constant value.Value, others Expressions) Expression
}

type caNAryBase struct {
	nAryBase
}

// Commutative.
func (this *caNAryBase) equivalentTo(expr, other Expression) bool {
	if reflect.TypeOf(expr) != reflect.TypeOf(other) {
		return false
	}

	that := interface{}(other).(*caNAryBase)
	if len(this.operands) != len(that.operands) {
		return false
	}

	found := make([]bool, len(this.operands))

	for _, first := range this.operands {
		for j, second := range that.operands {
			if !found[j] && first.EquivalentTo(second) {
				found[j] = true
				break
			}
		}
	}

	for _, f := range found {
		if !f {
			return false
		}
	}

	return true
}

// Associative.
func (this *caNAryBase) fold(expr caNAry) (Expression, error) {
	t, e := expr.VisitChildren(&Folder{})
	if e != nil {
		return t, e
	}

	operands := make(Expressions, 0, len(this.operands))
	for _, o := range this.operands {
		if reflect.TypeOf(expr) == reflect.TypeOf(o) {
			// Associative, so promote subexpressions.
			for _, oo := range interface{}(o).(*caNAryBase).operands {
				operands = append(operands, oo)
			}
		} else {
			operands = append(operands, o)
		}
	}

	this.operands = operands

	constants := make(value.Values, 0, len(operands))
	others := make(Expressions, 0, len(operands))
	for i, o := range operands {
		switch o := o.(type) {
		case *Constant:
			constants[i] = o.Value()
		default:
			others[i] = o
		}
	}

	if len(constants) == 0 {
		return expr, nil
	}

	c, e := expr.eval(constants)
	if e != nil {
		return nil, e
	}

	if len(others) == 0 {
		return NewConstant(c), nil
	}

	return expr.construct(c, others), nil
}
