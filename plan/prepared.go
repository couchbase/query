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

	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/value"
)

func BuildPrepared(stmt algebra.Statement, datastore, systemstore datastore.Datastore,
	creds datastore.Credentials, namespace string, subquery bool) (*Prepared, error) {
	operator, err := Build(stmt, datastore, systemstore, creds, namespace, subquery)
	if err != nil {
		return nil, err
	}

	signature := stmt.Signature()
	return newPrepared(operator, signature), nil
}

type Prepared struct {
	Operator
	signature value.Value
}

func newPrepared(operator Operator, signature value.Value) *Prepared {
	return &Prepared{
		Operator:  operator,
		signature: signature,
	}
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 2)
	r["operator"] = this.Operator
	r["signature"] = this.signature

	return json.Marshal(r)
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &op_type)
	if err != nil {
		return err
	}

	this.Operator, err = MakeOperator(op_type.Operator, body)

	return err
}

func (this *Prepared) Signature() value.Value {
	return this.signature
}
