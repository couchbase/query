//  Copyright 2017-Present Couchbase, Inc.
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

type IndexKeyOrders []*IndexKeyOrder

type IndexKeyOrder struct {
	KeyPos int
	Desc   bool
}

func NewIndexKeyOrders(keyPos int, desc bool) *IndexKeyOrder {
	return &IndexKeyOrder{
		KeyPos: keyPos,
		Desc:   desc,
	}
}

func (this *IndexKeyOrder) Copy() *IndexKeyOrder {
	return &IndexKeyOrder{
		KeyPos: this.KeyPos,
		Desc:   this.Desc,
	}
}

func (this *IndexKeyOrder) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *IndexKeyOrder) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 2)

	r["keypos"] = this.KeyPos
	if this.Desc {
		r["desc"] = this.Desc
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *IndexKeyOrder) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		KeyPos int  `json:"keypos"`
		Desc   bool `json:"desc"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.KeyPos = _unmarshalled.KeyPos
	this.Desc = _unmarshalled.Desc

	return nil
}

func (this *IndexKeyOrder) String() string {
	bytes, _ := this.MarshalJSON()
	return string(bytes)
}

func (this IndexKeyOrders) Copy() IndexKeyOrders {
	orderkeys := make(IndexKeyOrders, len(this))
	for i, r := range this {
		if r != nil {
			orderkeys[i] = r.Copy()
		}
	}

	return orderkeys
}
