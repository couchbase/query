//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"math"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type VectorMetric string

const (
	EUCLIDEAN         VectorMetric = "euclidean"
	EUCLIDEAN_SQUARED VectorMetric = "euclidean_squared"
	L2                VectorMetric = "l2"         // same as euclidean
	L2_SQUARED        VectorMetric = "l2_squared" // same as euclidean_squared
	COSINE            VectorMetric = "cosine"     // 1 - cosine_sim
	DOT               VectorMetric = "dot"        // negate of dot_product
	EMPTY_METRIC      VectorMetric = ""
)

const FLOAT32_SIZE = 4

func GetVectorMetric(arg string) (metric VectorMetric) {
	switch strings.ToLower(arg) {
	case "euclidean":
		metric = EUCLIDEAN
	case "euclidean_squared":
		metric = EUCLIDEAN_SQUARED
	case "l2":
		metric = L2
	case "l2_squared":
		metric = L2_SQUARED
	case "cosine":
		metric = COSINE
	case "dot":
		metric = DOT
	default:
		metric = EMPTY_METRIC
	}
	return
}

const (
	_VECTOR_QVEC_CHECKED = 1 << iota
)

type Knn struct {
	FunctionBase
	metric VectorMetric
	flags  uint32
}

func NewKnn(operands Expressions) Function {
	var metric VectorMetric
	// get metric (3rd argument)
	// MinArgs()/MaxArgs() ensures len(operands) == 3
	ev := operands[2].Value()
	if ev != nil && ev.Type() == value.STRING {
		metric = GetVectorMetric(ev.ToString())
	}

	rv := &Knn{
		metric: metric,
	}
	rv.Init("vector_distance", operands...)
	rv.expr = rv
	return rv
}

func (this *Knn) Copy() Expression {
	rv := &Knn{
		metric: this.metric,
		flags:  this.flags,
	}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	rv.BaseCopy(this)
	return rv
}

func (this *Knn) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Knn) PropagatesNull() bool { return false }
func (this *Knn) Indexable() bool      { return false }
func (this *Knn) Type() value.Type     { return value.NUMBER }
func (this *Knn) MinArgs() int {
	return 3
}

func (this *Knn) MaxArgs() int {
	return 3
}

func (this *Knn) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewKnn(operands)
	}
}

func (this *Knn) ValidOperands() error {
	field := this.operands[0].Static()
	if field != nil {
		return errors.NewVectorFuncInvalidField(this.name, field.String())
	}
	switch this.metric {
	case EUCLIDEAN, EUCLIDEAN_SQUARED, L2, L2_SQUARED, COSINE, DOT:
		// no-op
	default:
		return errors.NewVectorFuncInvalidMetric(this.name, string(this.metric))
	}
	qVec := this.operands[1].Value()
	if qVec != nil {
		if valid, errStr := validVector(qVec, 0); !valid {
			return errors.NewInvalidQueryVector(errStr)
		}
		this.flags |= _VECTOR_QVEC_CHECKED
	}
	return nil
}

func (this *Knn) Metric() VectorMetric {
	return this.metric
}

func (this *Knn) Field() Expression {
	return this.operands[0]
}

func (this *Knn) QueryVector() Expression {
	return this.operands[1]
}

func (this *Knn) Evaluate(item value.Value, context Context) (value.Value, error) {
	if (this.flags & _VECTOR_QVEC_CHECKED) == 0 {
		rv, err := vectorDistance(this.metric, this.operands, true, item, context)
		this.flags |= _VECTOR_QVEC_CHECKED
		return rv, err
	}
	return vectorDistance(this.metric, this.operands, false, item, context)
}

func vectorDistance(metric VectorMetric, operands Expressions, checkQVec bool, item value.Value, context Context) (value.Value, error) {
	var queryVec value.Value
	vec, err := operands[0].Evaluate(item, context)
	if err == nil {
		queryVec, err = operands[1].Evaluate(item, context)
	}
	if err != nil {
		return nil, err
	}
	if vec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if vec.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}
	if checkQVec {
		if valid, errStr := validVector(queryVec, 0); !valid {
			return nil, errors.NewInvalidQueryVector(errStr)
		}
	} else if queryVec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if queryVec.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	var dist, sqf, sdf float64
	for i := 0; ; i++ {
		qv, qOk := queryVec.Index(i)
		dv, dOk := vec.Index(i)
		if !qOk && !dOk {
			break
		} else if qOk && dOk {
			if qv.Type() != value.NUMBER || dv.Type() != value.NUMBER {
				return value.NULL_VALUE, nil
			}
			qf := value.AsNumberValue(qv).Float64()
			df := value.AsNumberValue(dv).Float64()
			if qf < -math.MaxFloat32 || qf > math.MaxFloat32 ||
				df < -math.MaxFloat32 || df > math.MaxFloat32 {
				return value.NULL_VALUE, nil
			}
			switch metric {
			case EUCLIDEAN, L2, EUCLIDEAN_SQUARED, L2_SQUARED:
				dist += (df - qf) * (df - qf)
			case DOT:
				dist += df * qf
			case COSINE:
				dist += df * qf
				sdf += df * df
				sqf += qf * qf
			}
		} else {
			return value.NULL_VALUE, nil
		}
	}
	switch metric {
	case EUCLIDEAN, L2:
		return value.NewValue(math.Sqrt(dist)), nil
	case EUCLIDEAN_SQUARED, L2_SQUARED:
		return value.NewValue(dist), nil
	case DOT:
		return value.NewValue(-dist), nil
	case COSINE:
		if sdf == float64(0) || sqf == float64(0) {
			return value.NULL_VALUE, nil
		}
		return value.NewValue(1.0 - (dist / (math.Sqrt(sdf) * math.Sqrt(sqf)))), nil
	}
	return value.NULL_VALUE, nil
}

func validVector(vec value.Value, dimension int) (bool, string) {
	if vec == nil {
		return false, "nil value used for vector"
	} else if vec.Type() != value.ARRAY {
		return false, "not an array"
	}
	for i := 0; ; i++ {
		v, ok := vec.Index(i)
		if !ok {
			if dimension > 0 {
				if i == dimension {
					return true, ""
				} else {
					return false, fmt.Sprintf("number of dimension (%d) does not match dimension specified (%d)", i, dimension)
				}
			}
			break
		} else if dimension > 0 && i >= dimension {
			return false, fmt.Sprintf("number of dimension (%d) does not match dimension specified (%d)", i, dimension)
		} else {
			if v.Type() != value.NUMBER {
				return false, fmt.Sprintf("array element (%v) not a number", v)
			}
			vf := value.AsNumberValue(v).Float64()
			if vf < -math.MaxFloat32 || vf > math.MaxFloat32 {
				return false, fmt.Sprintf("array element (%v) not a float32", vf)
			}
		}
	}
	return true, ""
}
