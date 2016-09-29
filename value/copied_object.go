//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// A copiedObjectValue is like an objectValue, except it shares its elements with
// at least one other object. Accordingly, when it is recycled, to prevent double recycling of
// maps, the recycling algorithm does not recurse down into the elements.

package value

type copiedObjectValue struct {
	objectValue
}

func (this copiedObjectValue) Recycle() {
	// Do nothing.
}
