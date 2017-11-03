//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type AnsiNest struct {
	readonly
	outer    bool
	alias    string
	onclause expression.Expression
	child    Operator
}

func NewAnsiNest(join *algebra.AnsiNest, child Operator) *AnsiNest {
	rv := &AnsiNest{
		outer:    join.Outer(),
		alias:    join.Alias(),
		onclause: join.Onclause(),
		child:    child,
	}

	return rv
}

func (this *AnsiNest) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAnsiNest(this)
}

func (this *AnsiNest) New() Operator {
	return &AnsiNest{}
}

func (this *AnsiNest) Outer() bool {
	return this.outer
}

func (this *AnsiNest) Alias() string {
	return this.alias
}

func (this *AnsiNest) Onclause() expression.Expression {
	return this.onclause
}

func (this *AnsiNest) Child() Operator {
	return this.child
}

func (this *AnsiNest) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AnsiNest) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AnsiNest"}
	r["alias"] = this.alias
	r["onclause"] = expression.NewStringer().Visit(this.onclause)

	if this.outer {
		r["outer"] = this.outer
	}

	r["~child"] = this.child

	if f != nil {
		f(r)
	}
	return r
}

func (this *AnsiNest) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Onclause string          `json:"onclause"`
		Outer    bool            `json:"outer"`
		Alias    string          `json:"alias"`
		Child    json.RawMessage `json:"~child"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Onclause != "" {
		this.onclause, err = parser.Parse(_unmarshalled.Onclause)
		if err != nil {
			return err
		}
	}

	this.outer = _unmarshalled.Outer
	this.alias = _unmarshalled.Alias

	raw_child := _unmarshalled.Child
	var child_type struct {
		Op_name string `json:"#operator"`
	}

	err = json.Unmarshal(raw_child, &child_type)
	if err != nil {
		return err
	}

	this.child, err = MakeOperator(child_type.Op_name, raw_child)
	if err != nil {
		return err
	}

	return nil
}
