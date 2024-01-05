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

const _DEF_NUM_CUSTOM_ATT = 4

func getAttachmentIndexFor(item value.AnnotatedValue, s string) int16 {
	index := int16(-1)
	custIndex := item.GetAttachment(value.ATT_CUSTOM_INDEX)
	var arr []string
	ok := false
	if custIndex != nil {
		if arr, ok = custIndex.([]string); ok {
			for i := range arr {
				if arr[i] == s {
					index = int16(i + 1)
					break
				}
			}
		}
	}
	if index == -1 {
		if arr == nil {
			arr = make([]string, 0, _DEF_NUM_CUSTOM_ATT)
		}
		arr = append(arr, s)
		index = int16(len(arr))
		item.SetAttachment(value.ATT_CUSTOM_INDEX, arr)
	}
	return index + value.ATT_CUSTOM_INDEX
}

func getCachedValue(item value.AnnotatedValue, expr expression.Expression, s string, context *opContext) (
	rv value.Value, err error) {

	i := getAttachmentIndexFor(item, s)
	sv1 := item.GetAttachment(i)
	switch sv1 := sv1.(type) {
	case value.Value:
		rv = sv1
	default:
		rv, err = expr.Evaluate(item, context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "cached value"))
			return
		}

		item.SetAttachment(i, rv)
	}
	return
}

func getOriginalCachedValue(item value.AnnotatedValue, expr expression.Expression, s string, context *opContext) (
	rv value.Value, err error) {

	i := getAttachmentIndexFor(item, s)
	sv1 := item.GetAttachment(i)
	switch sv1 := sv1.(type) {
	case value.Value:
		rv = sv1
	default:
		rv, err = expr.Evaluate(item.Original(), context)
		if err != nil {
			context.Error(errors.NewEvaluationError(err, "original cached value"))
			return
		}
		item.SetAttachment(i, rv.CopyForUpdate())
	}
	return
}
