//  Copyright 2020-Present Couchbase, Inc.
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

type All struct {
	execution
}

func NewAll() *All {
	return &All{}
}

func (this *All) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAll(this)
}

func (this *All) New() Operator {
	return &All{}
}

func (this *All) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *All) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "All"}
	if f != nil {
		f(r)
	}
	return r
}

func (this *All) UnmarshalJSON([]byte) error {
	// NOP: All has no data structure
	return nil
}
