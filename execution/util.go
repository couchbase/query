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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func notifyChildren(children ...Operator) {
	for _, child := range children {
		if child != nil {
			child.SendAction(_ACTION_STOP)
		}
	}
}

func pauseChildren(children ...Operator) {
	for _, child := range children {
		if child != nil {
			child.SendAction(_ACTION_PAUSE)
		}
	}
}

func copyOperator(op Operator) Operator {
	if op == nil {
		return nil
	} else {
		return op.Copy()
	}
}

var _STRING_POOL = util.NewStringPool(_BATCH_SIZE)
var _STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(_BATCH_SIZE)

func getCachedValue(item value.AnnotatedValue, expr expression.Expression, s string, context *Context) (rv value.Value, err error) {
	sv1 := item.GetAttachment(s)
	switch sv1 := sv1.(type) {
	case value.Value:
		rv = sv1
	default:
		rv, err = expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "getCachedValue()"))
			return
		}

		item.SetAttachment(s, rv)
	}
	return
}
