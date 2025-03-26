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

type IndexProjection struct {
	EntryKeys  []int
	PrimaryKey bool
}

func NewIndexProjection(size int, primary bool) *IndexProjection {
	return &IndexProjection{
		EntryKeys:  make([]int, 0, size),
		PrimaryKey: primary,
	}
}

func (this *IndexProjection) Copy() *IndexProjection {
	rv := &IndexProjection{
		EntryKeys:  make([]int, 0, len(this.EntryKeys)),
		PrimaryKey: this.PrimaryKey,
	}
	rv.EntryKeys = append(rv.EntryKeys, this.EntryKeys...)
	return rv
}

func (this *IndexProjection) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexProjection) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)

	if len(this.EntryKeys) != 0 {
		r["entry_keys"] = this.EntryKeys
	}
	if this.PrimaryKey {
		r["primary_key"] = this.PrimaryKey
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexProjection) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		EntryKeys  []int `json:"entry_keys"`
		PrimaryKey bool  `json:"primary_key"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.EntryKeys = _unmarshalled.EntryKeys
	this.PrimaryKey = _unmarshalled.PrimaryKey

	return nil
}

func (this *IndexProjection) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}
