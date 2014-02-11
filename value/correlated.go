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
	json "github.com/dustin/gojson"
)

// CorrelatedValue enables subqueries.
func NewCorrelatedValue(parent Value) Value {
	return &correlatedValue{
		entries: make(map[string]interface{}),
		parent:  parent,
	}
}

// CorrelatedValue enables subqueries.
type correlatedValue struct {
	entries map[string]interface{}
	parent  Value
}

func (this *correlatedValue) Type() int {
	return OBJECT
}

func (this *correlatedValue) Actual() interface{} {
	return this.entries
}

func (this *correlatedValue) Equals(other Value) bool {
	switch other := other.(type) {
	case *correlatedValue:
		return objectEquals(this.entries, other.entries)
	case objectValue:
		return objectEquals(this.entries, other)
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this *correlatedValue) Collate(other Value) int {
	switch other := other.(type) {
	case *correlatedValue:
		return objectCollate(this.entries, other.entries)
	case objectValue:
		return objectCollate(this.entries, other)
	case *parsedValue:
		return this.Collate(other.parse())
	case *annotatedValue:
		return this.Collate(other.Value)
	default:
		return 1
	}
}

func (this *correlatedValue) Truth() bool {
	return len(this.entries) > 0
}

func (this *correlatedValue) Copy() Value {
	return &correlatedValue{
		entries: copyMap(this.entries, self),
		parent:  this.parent,
	}
}

func (this *correlatedValue) CopyForUpdate() Value {
	return &correlatedValue{
		entries: copyMap(this.entries, copyForUpdate),
		parent:  this.parent,
	}
}

func (this *correlatedValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

// Search self and ancestors. Enables subqueries.
func (this *correlatedValue) Field(field string) Value {
	result, ok := this.entries[field]
	if ok {
		return NewValue(result)
	}

	if this.parent != nil {
		return this.parent.Field(field)
	}

	return missingField(field)
}

func (this *correlatedValue) SetField(field string, val interface{}) error {
	this.entries[field] = val
	return nil
}

func (this *correlatedValue) Index(index int) Value {
	return missingIndex(index)
}

func (this *correlatedValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}
