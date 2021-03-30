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

// ParentScan is used for UNNEST subqueries.
type ParentScan struct {
	legacy
}

func NewParentScan() *ParentScan {
	return &ParentScan{}
}

func (this *ParentScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParentScan(this)
}

func (this *ParentScan) New() Operator {
	return &ParentScan{}
}

func (this *ParentScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *ParentScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "ParentScan"}
	if f != nil {
		f(r)
	}
	return r
}

func (this *ParentScan) UnmarshalJSON([]byte) error {
	// NOP: ParentScan has no data structure
	return nil
}
