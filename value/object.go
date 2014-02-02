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

type objectValue map[string]interface{}

func (this objectValue) Type() int {
	return OBJECT
}

func (this objectValue) Actual() interface{} {
	return (map[string]interface{})(this)
}

func (this objectValue) Equals(other Value) bool {
	switch other := other.(type) {
	case objectValue:
		return objectEquals(this, other)
	case *correlatedValue:
		return objectEquals(this, other.entries)
	case *parsedValue:
		return this.Equals(other.parse())
	case *annotatedValue:
		return this.Equals(other.Value)
	default:
		return false
	}
}

func (this objectValue) Copy() Value {
	return objectValue(copyMap(this, self))
}

func (this objectValue) CopyForUpdate() Value {
	return objectValue(copyMap(this, copyForUpdate))
}

func (this objectValue) Bytes() []byte {
	bytes, err := json.Marshal(this.Actual())
	if err != nil {
		panic(_MARSHAL_ERROR)
	}
	return bytes
}

func (this objectValue) Field(field string) (Value, error) {
	result, ok := this[field]
	if ok {
		return NewValue(result), nil
	}

	// consistent with parsedValue
	return nil, Undefined(field)
}

func (this objectValue) SetField(field string, val interface{}) error {
	this[field] = val
	return nil
}

func (this objectValue) Index(index int) (Value, error) {
	return nil, Undefined(index)
}

func (this objectValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func objectEquals(first, second map[string]interface{}) bool {
	if len(first) != len(second) {
		return false
	}

	for fk, fv := range first {
		sv, ok := second[fk]
		if !ok || !NewValue(fv).Equals(NewValue(sv)) {
			return false
		}
	}

	return true
}
