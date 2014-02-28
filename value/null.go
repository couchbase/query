//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

type nullValue struct {
}

var NULL_VALUE = &nullValue{}

func NewNullValue() Value {
	return NULL_VALUE
}

func (this *nullValue) Type() int {
	return NULL
}

func (this *nullValue) Actual() interface{} {
	return nil
}

func (this *nullValue) Equals(other Value) bool {
	return other.Type() == NULL
}

func (this *nullValue) Collate(other Value) int {
	return NULL - other.Type()
}

func (this *nullValue) Truth() bool {
	return false
}

func (this *nullValue) Copy() Value {
	return this
}

func (this *nullValue) CopyForUpdate() Value {
	return this
}

var _NULL_BYTES = []byte("null")

func (this *nullValue) Bytes() []byte {
	return _NULL_BYTES
}

func (this *nullValue) Field(field string) (Value, bool) {
	return NULL_VALUE, false
}

func (this *nullValue) SetField(field string, val interface{}) error {
	return Unsettable(field)
}

func (this *nullValue) Index(index int) (Value, bool) {
	return NULL_VALUE, false
}

func (this *nullValue) SetIndex(index int, val interface{}) error {
	return Unsettable(index)
}

func (this *nullValue) Slice(start, end int) (Value, bool) {
	return NULL_VALUE, false
}
