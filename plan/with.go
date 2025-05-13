//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
	"github.com/couchbase/query/value"
)

type With struct {
	readonly
	optEstimate
	bindings *algebra.WithClause
	child    Operator
}

func NewWith(bindings *algebra.WithClause, child Operator, cost, cardinality float64,
	size int64, frCost float64) *With {
	rv := &With{
		bindings: bindings,
		child:    child,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *With) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitWith(this)
}

func (this *With) New() Operator {
	return &With{}
}

func (this *With) Bindings() *algebra.WithClause {
	return this.bindings
}

func (this *With) Readonly() bool {
	return this.child.Readonly()
}

func (this *With) Child() Operator {
	return this.child
}

func (this *With) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *With) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "With"}
	r["recursive"] = this.bindings.IsRecursive()
	r["bindings"] = this.bindings.Bindings()
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	} else {
		r["~child"] = this.child
	}
	return r
}

func (this *With) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Recursive   bool                   `json:"recursive"`
		Bindings    json.RawMessage        `json:"bindings"`
		Child       json.RawMessage        `json:"~child"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
	}

	var child_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	withs, err := unmarshalWiths(_unmarshalled.Bindings)
	if err != nil {
		return err
	}

	this.bindings = algebra.NewWithClause(_unmarshalled.Recursive, withs)
	this.bindings.SetBindings(withs)

	err = json.Unmarshal(_unmarshalled.Child, &child_type)
	if err != nil {
		return err
	}

	planContext := this.PlanContext()
	if planContext != nil {
		planContext.addWiths(withs)
	}

	this.child, err = MakeOperator(child_type.Operator, _unmarshalled.Child, planContext)
	if err != nil {
		return err
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)

	return nil
}

func unmarshalWiths(body []byte) (expression.Withs, error) {
	var _unmarshalled []struct {
		Alias   string          `json:"alias"`
		Expr    string          `json:"expr"`
		Rexpr   string          `json:"rexpr"`
		Isunion bool            `json:"is_union"`
		Config  json.RawMessage `json:"config"`
		Cycle   json.RawMessage `json:"cycle"`
		Var     string          `json:"var"` // for plan from pre-7.6 server
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	withs := make(expression.Withs, len(_unmarshalled))
	for i, with := range _unmarshalled {
		var expr, rexpr expression.Expression
		var union bool
		var config value.Value
		var cycle *algebra.CycleCheck
		var err error

		expr, err = parser.Parse(with.Expr)
		if err != nil {
			return nil, err
		}

		if with.Rexpr != "" {
			rexpr, err = parser.Parse(with.Rexpr)
			if err != nil {
				return nil, err
			}

			if with.Isunion {
				union = true
			}
			if len(with.Config) > 0 {
				config = value.NewValue([]byte(with.Config))
			}
			if with.Cycle != nil {
				cycle, err = unmarshalCycle([]byte(with.Cycle))
				if err != nil {
					return nil, err
				}
			}
		}

		alias := with.Alias
		if alias == "" {
			if with.Var != "" {
				// if the plan was generated from pre-7.6 server, it'll have "var" instead of "alias"
				alias = with.Var
			} else {
				return nil, errors.NewPlanInternalError("Unmarshal of WITH clause is missing WITH alias")
			}
		}
		withs[i] = algebra.NewWith(alias, expr, rexpr, union, config, cycle)
	}

	return withs, nil
}

func unmarshalCycle(body []byte) (*algebra.CycleCheck, error) {
	var _unmarshalled struct {
		Cycle []string `json:"cycle"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return nil, err
	}

	cycle := expression.Expressions{}
	for _, field := range _unmarshalled.Cycle {
		expfield, err := parser.Parse(field)
		if err != nil {
			return nil, err
		}

		cycle = append(cycle, expfield)
	}

	cycleCheck := algebra.NewCycleCheck(cycle)
	return cycleCheck, nil
}

func (this *With) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
