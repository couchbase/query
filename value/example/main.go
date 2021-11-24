//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"fmt"

	"github.com/couchbase/query/value"
)

func main() {
	// read some JSON
	bytes := []byte(`{"type":"test"}`)

	// create a Value object
	doc := value.NewValue(bytes)

	// attempt to access a nested Value
	docType, ok := doc.Field("type")
	if !ok {
		panic("No property type exists.")
	}

	// convert docType to a native go value
	docTypeValue := docType.Actual()

	// display the value
	fmt.Printf("document type is %v\n", docTypeValue)
}
