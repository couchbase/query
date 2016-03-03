//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
)

// CountScan is used for SELECT COUNT(*) with no WHERE clause.
type CountScan struct {
	readonly
	keyspace datastore.Keyspace
	term     *algebra.KeyspaceTerm
}

func NewCountScan(keyspace datastore.Keyspace, term *algebra.KeyspaceTerm) *CountScan {
	return &CountScan{
		keyspace: keyspace,
		term:     term,
	}
}

func (this *CountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCountScan(this)
}

func (this *CountScan) New() Operator {
	return &CountScan{}
}

func (this *CountScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CountScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *CountScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "CountScan"}
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	if this.duration != 0 {
		r["#time"] = this.duration.String()
	}
	return json.Marshal(r)
}

func (this *CountScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_     string `json:"#operator"`
		Names string `json:"namespace"`
		Keys  string `json:"keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)

	return err
}
