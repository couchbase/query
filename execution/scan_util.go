//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func evalOne(expr expression.Expression, context *opContext, parent value.Value) (v value.Value, empty bool, e error) {
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

func eval(cx expression.Expressions, context *opContext, parent value.Value) (value.Values, bool, error) {
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

func evalLimitOffset(expr expression.Expression, parent value.Value, defval int64, covering bool, context *opContext) (val int64) {
	if expr != nil {
		val, e := expr.Evaluate(parent, context)
		if e == nil && val.Type() == value.NUMBER {
			return val.(value.NumberValue).Int64()
		}
	}

	return defval
}

func getKeyspacePath(expr expression.Expression, context *opContext) (*algebra.Path, error) {
	if expr == nil {
		return nil, nil
	}

	v, e := expr.Evaluate(nil, context)
	if e != nil || v == nil || v.Type() != value.STRING {
		return nil, e
	}
	return algebra.NewVariablePathWithContext(v.Actual().(string), context.Namespace(), context.queryContext)
}

func getKeyspace(keyspace datastore.Keyspace, expr expression.Expression, context *opContext) datastore.Keyspace {
	if keyspace == nil {
		path, err := getKeyspacePath(expr, context)
		if err == nil && path != nil {
			keyspace, err = datastore.GetKeyspace(path.Parts()...)
		}
		if err != nil || keyspace == nil {
			context.Error(errors.NewEvaluationError(err, "expr is not valid"))
			return nil
		}
	}
	return keyspace
}

var _INDEX_SCAN_POOL = NewOperatorPool(16)
var _INDEX_VALUE_POOL = value.NewStringAnnotatedPool(1024)
var _INDEX_BIT_POOL = util.NewStringInt64Pool(1024)
