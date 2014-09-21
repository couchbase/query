//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

type Bindings []*Binding

type Binding struct {
	variable string
	expr     Expression
	descend  bool
}

func NewBinding(variable string, expr Expression) *Binding {
	return &Binding{variable, expr, false}
}

func NewDescendantBinding(variable string, expr Expression) *Binding {
	return &Binding{variable, expr, true}
}

func (this *Binding) Variable() string {
	return this.variable
}

func (this *Binding) Expression() Expression {
	return this.expr
}

func (this *Binding) SetExpression(expr Expression) {
	this.expr = expr
}

func (this *Binding) Descend() bool {
	return this.descend
}

func (this Bindings) MapExpressions(mapper Mapper) (err error) {
	for _, b := range this {
		expr, err := mapper.Map(b.expr)
		if err != nil {
			return err
		}

		b.expr = expr
	}

	return
}
