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
)

type Parallel struct {
	child Operator
}

func NewParallel(child Operator) *Parallel {
	return &Parallel{child}
}

func (this *Parallel) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitParallel(this)
}

func (this *Parallel) New() Operator {
	return &Parallel{}
}

func (this *Parallel) Readonly() bool {
	return this.child.Readonly()
}

func (this *Parallel) Child() Operator {
	return this.child
}

func (this *Parallel) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Parallel"}
	r["~child"] = this.child
	return json.Marshal(r)
}

func (this *Parallel) UnmarshalJSON([]byte) error {
	// TODO: Implement
	return nil
}
