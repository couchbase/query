//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func getCachedValue(item value.AnnotatedValue, expr expression.Expression, s string, context *opContext) (rv value.Value,
	err error) {

	sv1 := item.GetAttachment(s)
	switch sv1 := sv1.(type) {
	case value.Value:
		rv = sv1
	default:
		rv, err = expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "cached value"))
			return
		}

		item.SetAttachment(s, rv)
	}
	return
}

func getOriginalCachedValue(item value.AnnotatedValue, expr expression.Expression, s string, context *opContext) (rv value.Value,
	err error) {

	sv1 := item.GetAttachment(s)
	switch sv1 := sv1.(type) {
	case value.Value:
		rv = sv1
	default:
		rv, err = expr.Evaluate(item.Original(), context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "original cached value"))
			return
		}
		item.SetAttachment(s, rv)
	}
	return
}
