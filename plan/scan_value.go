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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// ValueScan is used for VALUES clauses, e.g. in INSERTs.
type ValueScan struct {
	readonly
	values algebra.Pairs
}

func NewValueScan(values algebra.Pairs) *ValueScan {
	return &ValueScan{
		values: values,
	}
}

func (this *ValueScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitValueScan(this)
}

func (this *ValueScan) New() Operator {
	return &ValueScan{}
}

func (this *ValueScan) Values() algebra.Pairs {
	return this.values
}

func (this *ValueScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "ValueScan"}
	r["values"] = this.values.Expression().String()
	return json.Marshal(r)
}

func (this *ValueScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_      string `json:"#operator"`
		Values string `json:"values"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Values == "" {
		return nil
	}

	expr, err := parser.Parse(_unmarshalled.Values)
	if err != nil {
		return err
	}

	array, ok := expr.(*expression.ArrayConstruct)
	if !ok {
		return fmt.Errorf("Invalid VALUES expression %s", _unmarshalled.Values)
	}

	this.values, err = algebra.NewValuesPairs(array)
	return err
}
