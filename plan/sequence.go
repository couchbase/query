//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import "encoding/json"

type Sequence struct {
	planContext *planContext
	children    []Operator `json:"~children"`
}

func NewSequence(children ...Operator) *Sequence {
	return &Sequence{nil, children}
}

func (this *Sequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitSequence(this)
}

func (this *Sequence) Readonly() bool {
	for _, child := range this.children {
		if !child.Readonly() {
			return false
		}
	}

	return true
}

func (this *Sequence) New() Operator {
	return &Sequence{}
}

func (this *Sequence) Children() []Operator {
	return this.children
}

func (this *Sequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Sequence) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Sequence"}
	if f != nil {
		f(r)
	} else {
		r["~children"] = this.children
	}
	return r
}

func (this *Sequence) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string            `json:"#operator"`
		Children []json.RawMessage `json:"~children"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.children = make([]Operator, 0, len(_unmarshalled.Children))

	for _, raw_child := range _unmarshalled.Children {
		var child_type struct {
			Op_name string `json:"#operator"`
		}

		err = json.Unmarshal(raw_child, &child_type)
		if err != nil {
			return err
		}

		child_op, err := MakeOperator(child_type.Op_name, raw_child, this.planContext)
		if err != nil {
			return err
		}

		this.children = append(this.children, child_op)
	}

	return err
}

func (this *Sequence) verify(prepared *Prepared) bool {
	for _, child := range this.children {
		if !child.verify(prepared) {
			return false
		}
	}

	return true
}

func (this *Sequence) Cost() float64 {
	last_child := len(this.children) - 1
	return this.children[last_child].Cost()
}

func (this *Sequence) Cardinality() float64 {
	last_child := len(this.children) - 1
	return this.children[last_child].Cardinality()
}

func (this *Sequence) Size() int64 {
	last_child := len(this.children) - 1
	return this.children[last_child].Size()
}

func (this *Sequence) FrCost() float64 {
	last_child := len(this.children) - 1
	return this.children[last_child].FrCost()
}

func (this *Sequence) PlanContext() *planContext {
	return this.planContext
}

func (this *Sequence) SetPlanContext(planContext *planContext) {
	this.planContext = planContext
}
