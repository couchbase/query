//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
