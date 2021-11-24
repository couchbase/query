//  Copyright 2021-Present Couchbase, Inc.
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

type Receive struct {
	execution
}

func NewReceive() *Receive {
	return &Receive{}
}

func (this *Receive) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitReceive(this)
}

func (this *Receive) New() Operator {
	return &Receive{}
}

func (this *Receive) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Receive) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Receive"}
	if f != nil {
		f(r)
	}
	return r
}

func (this *Receive) UnmarshalJSON([]byte) error {
	// NOP: Receive has no data structure
	return nil
}
