//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

// Sorter sorts an ARRAY Value in place.
type Sorter struct {
	value Value
}

func NewSorter(value Value) *Sorter {
	return &Sorter{value: NewValue(value)}
}

func (this *Sorter) Len() int {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		return len(actual)
	default:
		return 0
	}
}

func (this *Sorter) Less(i, j int) bool {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		return NewValue(actual[i]).Collate(NewValue(actual[j])) < 0
	default:
		return false
	}
}

func (this *Sorter) Swap(i, j int) {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		actual[i], actual[j] = actual[j], actual[i]
	}
}
