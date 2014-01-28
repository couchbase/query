//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package main

import (
	"fmt"

	"github.com/couchbaselabs/query/value"
)

func main() {
	// read some JSON
	bytes := []byte(`{"type":"test"}`)

	// create a Value object
	doc := value.NewValueFromBytes(bytes)

	// attempt to access a nested Value
	docType, err := doc.Field("type")
	if err != nil {
		panic("No property type exists.")
	}

	// convert docType to a native go value
	docTypeValue := docType.Actual()

	// display the value
	fmt.Printf("document type is %v\n", docTypeValue)
}
