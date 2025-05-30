//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// Grouping of input data. Parallelizable.
type InitialGroup struct {
	readonly
	optEstimate
	keys          expression.Expressions
	aggregates    algebra.Aggregates
	flags         uint32
	groupAs       string   // the alias of the Group As clause
	groupAsFields []string // the allowed fields in the Group As output
}

func NewInitialGroup(keys expression.Expressions, aggregates algebra.Aggregates,
	cost, cardinality float64, size int64, frCost float64, canSpill bool, groupAs string, groupAsFields []string) *InitialGroup {
	rv := &InitialGroup{
		keys:          keys,
		aggregates:    aggregates,
		groupAs:       groupAs,
		groupAsFields: groupAsFields,
	}
	if canSpill {
		rv.flags |= _CAN_SPILL
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *InitialGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitInitialGroup(this)
}

func (this *InitialGroup) New() Operator {
	return &InitialGroup{}
}

func (this *InitialGroup) Keys() expression.Expressions {
	return this.keys
}

func (this *InitialGroup) SetKeys(keys expression.Expressions) {
	this.keys = keys
}

func (this *InitialGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}

func (this *InitialGroup) CanSpill() bool {
	return (this.flags & _CAN_SPILL) != 0
}

func (this *InitialGroup) GroupAs() string {
	return this.groupAs
}

func (this *InitialGroup) GroupAsFields() []string {
	return this.groupAsFields
}

func (this *InitialGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *InitialGroup) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "InitialGroup"}
	keylist := make([]string, 0, len(this.keys))
	for _, key := range this.keys {
		keylist = append(keylist, key.String())
	}
	r["group_keys"] = keylist
	s := make([]interface{}, 0, len(this.aggregates))
	for _, agg := range this.aggregates {
		s = append(s, agg.String())
	}
	r["aggregates"] = s
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if this.flags > 0 {
		r["flags"] = this.flags
	}
	if this.groupAs != "" {
		r["group_as"] = this.groupAs
	}
	if len(this.groupAsFields) > 0 {
		r["group_as_fields"] = this.groupAsFields
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *InitialGroup) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_             string                 `json:"#operator"`
		Keys          []string               `json:"group_keys"`
		Aggs          []string               `json:"aggregates"`
		OptEstimate   map[string]interface{} `json:"optimizer_estimates"`
		Flags         uint32                 `json:"flags"`
		GroupAs       string                 `json:"group_as"`
		GroupAsFields []string               `json:"group_as_fields"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keys = make(expression.Expressions, len(_unmarshalled.Keys))
	for i, key := range _unmarshalled.Keys {
		key_expr, err := parser.Parse(key)
		if err != nil {
			return err
		}
		this.keys[i] = key_expr
	}

	this.aggregates = make(algebra.Aggregates, len(_unmarshalled.Aggs))
	for i, agg := range _unmarshalled.Aggs {
		agg_expr, err := parser.Parse(agg)
		if err != nil {
			return err
		}
		this.aggregates[i], _ = agg_expr.(algebra.Aggregate)
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)
	this.flags = _unmarshalled.Flags
	this.groupAs = _unmarshalled.GroupAs
	this.groupAsFields = _unmarshalled.GroupAsFields

	planContext := this.PlanContext()
	if planContext != nil {
		err = this.keys.MapExpressions(planContext)
		if err != nil {
			return err
		}
		for _, agg := range this.aggregates {
			err = agg.Children().MapExpressions(planContext)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Grouping of groups. Recursable and parallelizable.
type IntermediateGroup struct {
	readonly
	optEstimate
	keys       expression.Expressions
	aggregates algebra.Aggregates
	flags      uint32
	groupAs    string
}

func NewIntermediateGroup(keys expression.Expressions, aggregates algebra.Aggregates,
	cost, cardinality float64, size int64, frCost float64, canSpill bool, groupAs string) *IntermediateGroup {
	rv := &IntermediateGroup{
		keys:       keys,
		aggregates: aggregates,
		groupAs:    groupAs,
	}
	if canSpill {
		rv.flags |= _CAN_SPILL
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *IntermediateGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIntermediateGroup(this)
}

func (this *IntermediateGroup) New() Operator {
	return &IntermediateGroup{}
}

func (this *IntermediateGroup) Keys() expression.Expressions {
	return this.keys
}

func (this *IntermediateGroup) SetKeys(keys expression.Expressions) {
	this.keys = keys
}

func (this *IntermediateGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}

func (this *IntermediateGroup) CanSpill() bool {
	return (this.flags & _CAN_SPILL) != 0
}

func (this *IntermediateGroup) GroupAs() string {
	return this.groupAs
}

func (this *IntermediateGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IntermediateGroup) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "IntermediateGroup"}
	keylist := make([]string, 0, len(this.keys))
	for _, key := range this.keys {
		keylist = append(keylist, key.String())
	}
	r["group_keys"] = keylist
	s := make([]interface{}, 0, len(this.aggregates))
	for _, agg := range this.aggregates {
		s = append(s, agg.String())
	}
	r["aggregates"] = s
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if this.flags > 0 {
		r["flags"] = this.flags
	}
	if this.groupAs != "" {
		r["group_as"] = this.groupAs
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *IntermediateGroup) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Keys        []string               `json:"group_keys"`
		Aggs        []string               `json:"aggregates"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		Flags       uint32                 `json:"flags"`
		GroupAs     string                 `json:"group_as"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keys = make(expression.Expressions, len(_unmarshalled.Keys))
	for i, key := range _unmarshalled.Keys {
		key_expr, err := parser.Parse(key)
		if err != nil {
			return err
		}
		this.keys[i] = key_expr
	}

	this.aggregates = make(algebra.Aggregates, len(_unmarshalled.Aggs))
	for i, agg := range _unmarshalled.Aggs {
		agg_expr, err := parser.Parse(agg)
		if err != nil {
			return err
		}
		this.aggregates[i], _ = agg_expr.(algebra.Aggregate)
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)
	this.flags = _unmarshalled.Flags
	this.groupAs = _unmarshalled.GroupAs

	planContext := this.PlanContext()
	if planContext != nil {
		err = this.keys.MapExpressions(planContext)
		if err != nil {
			return err
		}
		for _, agg := range this.aggregates {
			err = agg.Children().MapExpressions(planContext)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Final grouping and aggregation.
type FinalGroup struct {
	readonly
	optEstimate
	keys       expression.Expressions
	aggregates algebra.Aggregates
	flags      uint32
}

func NewFinalGroup(keys expression.Expressions, aggregates algebra.Aggregates,
	cost, cardinality float64, size int64, frCost float64, canSpill bool) *FinalGroup {
	rv := &FinalGroup{
		keys:       keys,
		aggregates: aggregates,
	}
	if canSpill {
		rv.flags |= _CAN_SPILL
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *FinalGroup) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFinalGroup(this)
}

func (this *FinalGroup) New() Operator {
	return &FinalGroup{}
}

func (this *FinalGroup) Keys() expression.Expressions {
	return this.keys
}

func (this *FinalGroup) SetKeys(keys expression.Expressions) {
	this.keys = keys
}

func (this *FinalGroup) Aggregates() algebra.Aggregates {
	return this.aggregates
}

func (this *FinalGroup) CanSpill() bool {
	return (this.flags & _CAN_SPILL) != 0
}

func (this *FinalGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *FinalGroup) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "FinalGroup"}
	keylist := make([]string, 0, len(this.keys))
	for _, key := range this.keys {
		keylist = append(keylist, key.String())
	}
	r["group_keys"] = keylist
	s := make([]interface{}, 0, len(this.aggregates))
	for _, agg := range this.aggregates {
		s = append(s, agg.String())
	}
	r["aggregates"] = s
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if this.flags > 0 {
		r["flags"] = this.flags
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *FinalGroup) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string                 `json:"#operator"`
		Keys        []string               `json:"group_keys"`
		Aggs        []string               `json:"aggregates"`
		OptEstimate map[string]interface{} `json:"optimizer_estimates"`
		Flags       uint32                 `json:"flags"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keys = make(expression.Expressions, len(_unmarshalled.Keys))
	for i, key := range _unmarshalled.Keys {
		key_expr, err := parser.Parse(key)
		if err != nil {
			return err
		}
		this.keys[i] = key_expr
	}

	this.aggregates = make(algebra.Aggregates, len(_unmarshalled.Aggs))
	for i, agg := range _unmarshalled.Aggs {
		agg_expr, err := parser.Parse(agg)
		if err != nil {
			return err
		}
		this.aggregates[i], _ = agg_expr.(algebra.Aggregate)
	}

	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)
	this.flags = _unmarshalled.Flags

	planContext := this.PlanContext()
	if planContext != nil {
		err = this.keys.MapExpressions(planContext)
		if err != nil {
			return err
		}
		for _, agg := range this.aggregates {
			err = agg.Children().MapExpressions(planContext)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
