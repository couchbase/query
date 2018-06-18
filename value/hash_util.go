//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

/*
 * A couple of helper functions for using hash table for values
 */
func MarshalValue(val interface{}) ([]byte, error) {
	hashVal := val.(Value)
	return hashVal.MarshalJSON()
}

func EqualValue(val1, val2 interface{}) bool {
	value1 := val1.(Value)
	value2 := val2.(Value)
	return value1.Equals(value2).Truth()
}
