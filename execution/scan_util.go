//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"fmt"
	"math"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/plan"
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

func getIndexVector(planIndexVector *plan.IndexVector, indexVector *datastore.IndexVector, parent value.Value,
	dimension int, allowRerank bool, context *opContext) errors.Error {

	qvVal, err := planIndexVector.QueryVector.Evaluate(parent, context)
	if err != nil {
		return errors.NewEvaluationError(err, "index vector parameter: query vector")
	}

	qvAct := qvVal.Actual()
	qvArr, ok := qvAct.([]interface{})
	if !ok {
		return errors.NewInvalidQueryVector("not an array")
	} else if len(qvArr) != dimension {
		return errors.NewInvalidQueryVector(fmt.Sprintf("number of dimension (%d) does not match index vector dimension (%d)",
			len(qvArr), dimension))
	}

	queryVector := make([]float32, len(qvArr))
	for i, v := range qvArr {
		var vf float64
		switch val := v.(type) {
		case int:
			vf = float64(val)
		case int64:
			vf = float64(val)
		case int32:
			vf = float64(val)
		case int16:
			vf = float64(val)
		case int8:
			vf = float64(val)
		case uint:
			vf = float64(val)
		case uint64:
			vf = float64(val)
		case uint32:
			vf = float64(val)
		case uint16:
			vf = float64(val)
		case uint8:
			vf = float64(val)
		case uintptr:
			vf = float64(val)
		case float32:
			vf = float64(val)
		case float64:
			vf = val
		case value.Value:
			if val.Type() != value.NUMBER {
				return errors.NewInvalidQueryVector(fmt.Sprintf("array element (%v) not a number", val))
			}
			vf = value.AsNumberValue(val).Float64()
		default:
			return errors.NewInvalidQueryVector(fmt.Sprintf("array element (%v of type %T) not a valid type", val, val))
		}
		if vf < -math.MaxFloat32 || vf > math.MaxFloat32 {
			return errors.NewInvalidQueryVector(fmt.Sprintf("array element (%v) not a float32", vf))
		}

		queryVector[i] = float32(vf)
	}
	indexVector.QueryVector = queryVector

	if planIndexVector.Probes != nil {
		probesVal, err := planIndexVector.Probes.Evaluate(parent, context)
		if err != nil {
			return errors.NewEvaluationError(err, "index vector parameter: probes")
		}
		probes, ok := value.IsIntValue(probesVal)
		if !ok {
			return errors.NewInvalidProbes("not an integer")
		}
		indexVector.Probes = int(probes)
	}

	if allowRerank && planIndexVector.ReRank != nil {
		avVal, err := planIndexVector.ReRank.Evaluate(parent, context)
		if err != nil {
			return errors.NewEvaluationError(err, "index vector parameter: rerank")
		}
		if avVal.Type() != value.BOOLEAN {
			return errors.NewInvalidReRank("not a boolean")
		}
		avAct := avVal.Actual().(bool)
		indexVector.ReRank = avAct
		indexVector.ActualVector = avAct // TODO: remove
	}

	return nil
}

func getIndexPartitionSets(planIndexPartitionSets plan.IndexPartitionSets, parent value.Value,
	context *opContext) (datastore.IndexPartitionSets, errors.Error) {

	indexPartitionSets := make(datastore.IndexPartitionSets, len(planIndexPartitionSets))
	for i, planPartitionSet := range planIndexPartitionSets {
		partitionSet := make(value.Values, len(planPartitionSet.PartitionSet))
		for j, partitionExpr := range planPartitionSet.PartitionSet {
			psVal, err := partitionExpr.Evaluate(parent, context)
			if err != nil {
				return nil, errors.NewEvaluationError(err, "index partition set")
			}
			partitionSet[j] = psVal
		}
		indexPartitionSets[i] = &datastore.IndexPartitionSet{partitionSet}
	}

	return indexPartitionSets, nil
}

var _INDEX_SCAN_POOL = NewOperatorPool(16)
var _INDEX_VALUE_POOL = value.NewStringAnnotatedPool(1024)
var _INDEX_BIT_POOL = util.NewStringInt64Pool(1024)
