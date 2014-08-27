//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"encoding/json"
)

// ScopeValue provides alias scoping for subqueries, FORs, LETs,
// projections, etc.
type ScopeValue struct {
	Value
	parent Value
}

// ScopeValue provides alias scoping for subqueries, FORs, LETs,
// projections, etc.
func NewScopeValue(value interface{}, parent Value) Value {
	return &ScopeValue{
		Value:  NewValue(value),
		parent: parent,
	}
}

func (this *ScopeValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Actual())
}

func (this *ScopeValue) Copy() Value {
	return &ScopeValue{
		Value:  this.Value.Copy(),
		parent: this.parent,
	}
}

func (this *ScopeValue) CopyForUpdate() Value {
	return &ScopeValue{
		Value:  this.Value.CopyForUpdate(),
		parent: this.parent,
	}
}

// Search self, the parent. Implements scoping.
func (this *ScopeValue) Field(field string) (Value, bool) {
	result, ok := this.Value.Field(field)
	if ok {
		return result, true
	}

	if this.parent != nil {
		return this.parent.Field(field)
	}

	return missingField(field), false
}

func (this *ScopeValue) Fields() map[string]interface{} {
	if this.parent == nil {
		return this.Value.Fields()
	}

	rv := make(map[string]interface{})

	v := this.Value.Fields()
	for vf, _ := range v {
		rv[vf] = vf
	}

	p := this.parent.Fields()
	for pf, _ := range p {
		rv[pf] = pf
	}

	return rv
}
