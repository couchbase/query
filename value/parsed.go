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
	"strconv"

	jsonpointer "github.com/dustin/go-jsonpointer"
	json "github.com/dustin/gojson"
)

// A structure for storing and manipulating a (possibly JSON) value.
type parsedValue struct {
	raw        []byte
	parsedType int
	parsed     Value
}

func (this *parsedValue) Type() int {
	return this.parsedType
}

func (this *parsedValue) Actual() interface{} {
	if this.parsedType == NOT_JSON {
		return nil
	}

	return this.parse().Actual()
}

func (this *parsedValue) Equals(other Value) bool {
	return this.parse().Equals(other)
}

func (this *parsedValue) Collate(other Value) int {
	return this.parse().Collate(other)
}

func (this *parsedValue) Copy() Value {
	if this.parsed != nil {
		return this.parsed.Copy()
	}

	rv := parsedValue{
		raw:        this.raw,
		parsedType: this.parsedType,
	}

	return &rv
}

func (this *parsedValue) CopyForUpdate() Value {
	if this.parsedType == NOT_JSON {
		return this.Copy()
	}

	return this.parse().CopyForUpdate()
}

func (this *parsedValue) Bytes() []byte {
	switch this.parsedType {
	case ARRAY, OBJECT:
		return this.parse().Bytes()
	default:
		return this.raw
	}
}

func (this *parsedValue) Field(field string) (Value, error) {
	if this.parsed != nil {
		return this.parsed.Field(field)
	}

	if this.parsedType != OBJECT {
		return nil, Undefined(field)
	}

	if this.raw != nil {
		res, err := jsonpointer.Find(this.raw, "/"+field)
		if err != nil {
			return nil, err
		}
		if res != nil {
			return NewValueFromBytes(res), nil
		}
	}

	return nil, Undefined(field)
}

func (this *parsedValue) SetField(field string, val interface{}) error {
	if this.parsedType != OBJECT {
		return Unsettable(field)
	}

	return this.parse().SetField(field, val)
}

func (this *parsedValue) Index(index int) (Value, error) {
	if this.parsed != nil {
		return this.parsed.Index(index)
	}

	if this.parsedType != ARRAY {
		return nil, Undefined(index)
	}

	if this.raw != nil {
		res, err := jsonpointer.Find(this.raw, "/"+strconv.Itoa(index))
		if err != nil {
			return nil, err
		}
		if res != nil {
			return NewValueFromBytes(res), nil
		}
	}

	return nil, Undefined(index)
}

func (this *parsedValue) SetIndex(index int, val interface{}) error {
	if this.parsedType != ARRAY {
		return Unsettable(index)
	}

	return this.parse().SetIndex(index, val)
}

func (this *parsedValue) parse() Value {
	if this.parsed == nil {
		if this.parsedType == NOT_JSON {
			return nil
		}

		var p interface{}
		err := json.Unmarshal(this.raw, &p)
		if err != nil {
			panic("Unexpected parse error on valid JSON.")
		}
		this.parsed = NewValue(p)
	}

	return this.parsed
}
