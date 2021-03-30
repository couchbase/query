//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package plan

import (
	"encoding/json"
)

type Collect struct {
	execution
}

func NewCollect() *Collect {
	return &Collect{}
}

func (this *Collect) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCollect(this)
}

func (this *Collect) New() Operator {
	return &Collect{}
}

func (this *Collect) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Collect) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Collect"}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Collect) UnmarshalJSON([]byte) error {
	// NOP: Collect has no data structure
	return nil
}
