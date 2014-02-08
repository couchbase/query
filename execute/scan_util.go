//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execute

import (
	"github.com/couchbaselabs/query/algebra"
	"github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

func eval(cx algebra.CompositeExpression, context *Context, parent value.Value) (value.CompositeValue, bool) {
	if cx == nil {
		return nil, true
	}

	cv := make(value.CompositeValue, len(cx))
	var e error
	for i, expr := range cx {
		cv[i], e = expr.Evaluate(parent, context)
		if e != nil {
			context.ErrorChannel() <- err.NewError(e, "Error evaluating filter term.")
			return nil, false
		}
	}

	return cv, true
}
