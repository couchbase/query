//  Copieright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

/*
List implements a slice of Values as []value.Value.
*/
type List struct {
	list Values
}

func NewList(size int) *List {
	return &List{
		list: make(Values, 0, size),
	}
}

func (this *List) Add(item Value) {
	this.list = append(this.list, item)
}

func (this *List) AddAll(items Values) {
	for _, item := range items {
		this.Add(item)
	}
}

func (this *List) Len() int {
	return len(this.list)
}

func (this *List) Values() []Value {
	return this.list
}

func (this *List) Clear() {
	this.list = nil
}

func (this *List) Copy() *List {
	rv := make(Values, len(this.list))
	for k, v := range this.list {
		rv[k] = v
	}
	return &List{rv}
}

func (this *List) Union(other *List) {
	this.AddAll(other.Values())
}
