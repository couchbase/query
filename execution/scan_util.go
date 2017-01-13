//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func eval(cx expression.Expressions, context *Context, parent value.Value) (value.Values, bool, error) {
	if cx == nil {
		return nil, false, nil
	}

	cv := make(value.Values, len(cx))
	var e error
	for i, expr := range cx {
		if expr == nil {
			continue
		}

		cv[i], e = expr.Evaluate(parent, context)
		if e != nil {
			return nil, false, e
		}

		if cv[i] != nil && (cv[i].Type() == value.NULL || cv[i].Type() == value.MISSING) &&
			expr.Value() == nil {
			return nil, true, nil
		}
	}

	return cv, false, nil
}

func notifyConn(stopchannel datastore.StopChannel) {
	select {
	case stopchannel <- false:
	default:
	}
}

func getLimit(limit expression.Expression, covering bool, context *Context) int64 {
	rv := int64(-1)
	if limit != nil {
		if context.ScanConsistency() == datastore.UNBOUNDED || covering {
			lv, err := limit.Evaluate(nil, context)
			if err == nil && lv.Type() == value.NUMBER {
				rv = lv.(value.NumberValue).Int64()
			}
		}
	}

	return rv
}

var _INDEX_SCAN_POOL = NewOperatorPool(16)
var _INDEX_COUNT_POOL = util.NewStringIntPool(1024)
var _INDEX_VALUE_POOL = value.NewStringAnnotatedPool(1024)
var _INDEX_BIT_POOL = util.NewStringInt64Pool(1024)
