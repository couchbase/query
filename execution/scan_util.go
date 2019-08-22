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

func evalOne(expr expression.Expression, context *Context, parent value.Value) (v value.Value, empty bool, e error) {
	if expr != nil {
		v, e = expr.Evaluate(parent, context)
	}

	if e != nil {
		return nil, false, e
	}

	if v != nil && (v.Type() == value.NULL || v.Type() == value.MISSING) && expr.Value() == nil {
		return nil, true, e
	}

	return
}

func eval(cx expression.Expressions, context *Context, parent value.Value) (value.Values, bool, error) {
	if cx == nil {
		return nil, false, nil
	}

	var e error
	var empty bool
	cv := make(value.Values, len(cx))

	for i, expr := range cx {
		cv[i], empty, e = evalOne(expr, context, parent)
		if e != nil || empty {
			return nil, empty, e
		}
	}

	return cv, false, nil
}

func notifyConn(stopchannel datastore.StopChannel) {
	// TODO we should accrue channel or service time here
	select {
	case stopchannel <- false:
	default:
	}
}

func evalLimitOffset(expr expression.Expression, parent value.Value, defval int64, covering bool, context *Context) (val int64) {
	if expr != nil {
		val, e := expr.Evaluate(parent, context)
		if e == nil && val.Type() == value.NUMBER {
			return val.(value.NumberValue).Int64()
		}
	}

	return defval
}

var _INDEX_SCAN_POOL = NewOperatorPool(16)
var _INDEX_VALUE_POOL = value.NewStringAnnotatedPool(1024)
var _INDEX_BIT_POOL = util.NewStringInt64Pool(1024)
