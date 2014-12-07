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

type Discard struct {
	readonly
}

func NewDiscard() *Discard {
	return &Discard{}
}

func (this *Discard) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDiscard(this)
}

func (this *Discard) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "Discard"}
	return json.Marshal(r)
}

func (this *Discard) New() Operator {
	return &Discard{}
}

func (this *Discard) UnmarshalJSON([]byte) error {
	// NOP: Discard has no data structure
	return nil
}
