//  Copyright 2016-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"fmt"
)

type NumberValue interface {
	Value

	Add(n NumberValue) NumberValue
	IDiv(n NumberValue) Value
	IMod(n NumberValue) Value
	Mult(n NumberValue) NumberValue
	Neg() NumberValue
	Sub(n NumberValue) NumberValue
	Int64() int64
	Float64() float64
}

func AsNumberValue(v Value) NumberValue {
	switch v := v.(type) {
	case NumberValue:
		return v
	case AnnotatedValue:
		return AsNumberValue(v.GetValue())
	default:
		panic(fmt.Sprintf("Invalid NumberValue %v", v))
	}
}
