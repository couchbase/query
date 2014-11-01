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

	"github.com/couchbaselabs/query/expression"
	"github.com/couchbaselabs/query/value"
)

type NamedParameter struct {
	expression.ExpressionBase
	name string
}

func NewNamedParameter(name string) expression.Expression {
	return &NamedParameter{
		name: name,
	}
}

func (this *NamedParameter) Accept(visitor expression.Visitor) (interface{}, error) {
	switch v := visitor.(type) {
	case ExpressionVisitor:
		return v.VisitNamedParameter(this)
	case expression.Mapper:
		return this, nil
	default:
		panic(fmt.Sprintf("NamedParameter visited by %T,", visitor))
	}
}

func (this *NamedParameter) Type() value.Type { return value.JSON }

func (this *NamedParameter) Evaluate(item value.Value, context expression.Context) (
	value.Value, error) {
	val, ok := context.(Context).NamedArg(this.name)

	if ok {
		return val, nil
	} else {
		return nil, fmt.Errorf("No value for named parameter $%s.", this.name)
	}
}

func (this *NamedParameter) Indexable() bool {
	return false
}

func (this *NamedParameter) EquivalentTo(other expression.Expression) bool {
	switch other := other.(type) {
	case *NamedParameter:
		return this.name == other.name
	default:
		return false
	}
}

func (this *NamedParameter) SubsetOf(other expression.Expression) bool {
	return this.EquivalentTo(other)
}

func (this *NamedParameter) Children() expression.Expressions {
	return nil
}

func (this *NamedParameter) MapChildren(mapper expression.Mapper) error {
	return nil
}
