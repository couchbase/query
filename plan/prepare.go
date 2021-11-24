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

	"github.com/couchbase/query/value"
)

type Prepare struct {
	execution
	prepared value.Value
	plan     *Prepared
	force    bool
}

func NewPrepare(prepared value.Value, plan *Prepared, force bool) *Prepare {
	return &Prepare{
		prepared: prepared,
		plan:     plan,
		force:    force,
	}
}

func (this *Prepare) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrepare(this)
}

func (this *Prepare) New() Operator {
	return &Prepare{}
}

func (this *Prepare) Prepared() value.Value {
	return this.prepared
}

func (this *Prepare) Plan() *Prepared {
	return this.plan
}

func (this *Prepare) Force() bool {
	return this.force
}

func (this *Prepare) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Prepare) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Prepare"}
	r["prepared"] = this.prepared
	if f != nil {
		f(r)
	}
	return r
}

func (this *Prepare) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_        string          `json:"#operator"`
		Prepared json.RawMessage `json:"prepared"`
	}
	var plan *Prepared

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.prepared = value.NewValue(_unmarshalled.Prepared)

	// I cannot foresee any use of the below, but for completeness
	err = json.Unmarshal(_unmarshalled.Prepared, &plan)
	if err != nil {
		return err
	}
	this.plan = plan
	this.force = true
	return nil
}
