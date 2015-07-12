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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func eval(cx expression.Expressions, context *Context, parent value.Value) (value.Values, bool) {
	if cx == nil {
		return nil, true
	}

	cv := make(value.Values, len(cx))
	var e error
	for i, expr := range cx {
		cv[i], e = expr.Evaluate(parent, context)
		if e != nil {
			context.Error(errors.NewEvaluationError(e, "filter term"))
			return nil, false
		}
	}

	return cv, true
}

func notifyConn(conn *datastore.IndexConnection) {
	select {
	case conn.StopChannel() <- false:
	default:
	}
}

var _SCAN_POOL = NewOperatorPool(16)
var _COUNT_POOL = util.NewStringIntPool(1024)
var _VALUE_POOL = value.NewStringAnnotatedPool(1024)
