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
)

type Alias struct {
	readonly
	optEstimate
	BuildBitFilterBase
	alias   string
	primary bool // alias for subquery as primary term
}

func NewAlias(alias string, primary bool, cost, cardinality float64, size int64, frCost float64) *Alias {
	rv := &Alias{
		alias:   alias,
		primary: primary,
	}
	setOptEstimate(&rv.optEstimate, cost, cardinality, size, frCost)
	return rv
}

func (this *Alias) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlias(this)
}

func (this *Alias) New() Operator {
	return &Alias{}
}

func (this *Alias) Alias() string {
	return this.alias
}

func (this *Alias) Primary() bool {
	return this.primary
}

func (this *Alias) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Alias) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Alias"}
	r["as"] = this.alias
	if !this.primary {
		r["secondary_term"] = !this.primary
	}
	if this.hasBuildBitFilter() {
		this.marshalBuildBitFilters(r)
	}
	if optEstimate := marshalOptEstimate(&this.optEstimate); optEstimate != nil {
		r["optimizer_estimates"] = optEstimate
	}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Alias) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_               string                 `json:"#operator"`
		As              string                 `json:"as"`
		Secondary       bool                   `json:"secondary_term"`
		OptEstimate     map[string]interface{} `json:"optimizer_estimates"`
		BuildBitFilters []json.RawMessage      `json:"build_bit_filters"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	this.alias = _unmarshalled.As
	this.primary = !_unmarshalled.Secondary // if not set assume to be primary to be safe
	if len(_unmarshalled.BuildBitFilters) > 0 {
		err = this.unmarshalBuildBitFilters(_unmarshalled.BuildBitFilters)
		if err != nil {
			return err
		}
	}
	unmarshalOptEstimate(&this.optEstimate, _unmarshalled.OptEstimate)
	return err
}
