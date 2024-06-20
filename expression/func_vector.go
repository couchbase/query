//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"math"
	"strings"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type VectorMetric string

const (
	EUCLIDEAN   VectorMetric = "euclidean"
	L2          VectorMetric = "l2"
	COSINE_SIM  VectorMetric = "cosine_sim"
	DOT_PRODUCT VectorMetric = "dot_product"
	DEF_DIST    VectorMetric = "def_dist"
	EMPTY_DIST  VectorMetric = ""
)

func GetVectorMetric(arg string) (metric VectorMetric) {
	switch strings.ToLower(arg) {
	case "euclidean":
		metric = EUCLIDEAN
	case "l2":
		metric = L2
	case "cosine_sim":
		metric = COSINE_SIM
	case "dot_product":
		metric = DOT_PRODUCT
	}
	return
}

type Knn struct {
	FunctionBase
	metric VectorMetric
}

func NewKnn(metric VectorMetric, operands Expressions) Function {
	var name string
	switch metric {
	case EUCLIDEAN:
		name = "euclidean_dist"
	case L2:
		name = "l2_dist"
	case COSINE_SIM:
		name = "cosine_sim_dist"
	case DOT_PRODUCT:
		name = "dot_product_dist"
	case DEF_DIST:
		name = "vector_dist"
	case EMPTY_DIST:
		name = "knn"
	default:
		name = "knn"
	}

	if metric == EMPTY_DIST || metric == DEF_DIST {
		// get metric (3rd argument)
		// MinArgs()/MaxArgs() ensures len(operands) == 3
		ev := operands[2].Value()
		if ev != nil && ev.Type() == value.STRING {
			metric = GetVectorMetric(ev.ToString())
		}
	}

	rv := &Knn{
		metric: metric,
	}
	rv.Init(name, operands...)
	rv.expr = rv
	return rv
}

func (this *Knn) Copy() Expression {
	rv := &Knn{
		metric: this.metric,
	}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	return rv
}

func (this *Knn) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Knn) PropagatesNull() bool { return false }
func (this *Knn) Indexable() bool      { return false }
func (this *Knn) Type() value.Type     { return value.NUMBER }
func (this *Knn) MinArgs() int {
	if this.metric == EMPTY_DIST || this.metric == DEF_DIST {
		return 3
	}
	return 2
}

func (this *Knn) MaxArgs() int {
	return this.MinArgs()
}

func (this *Knn) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewKnn(this.metric, operands)
	}
}

func (this *Knn) ValidOperands() error {
	switch this.metric {
	case EUCLIDEAN, L2, COSINE_SIM, DOT_PRODUCT:
		return nil
	}
	return errors.NewVectorFuncInvalidMetric(this.name, string(this.metric))
}

func (this *Knn) Metric() VectorMetric {
	return this.metric
}

func (this *Knn) QueryVector() Expression {
	return this.operands[1]
}

func (this *Knn) Evaluate(item value.Value, context Context) (value.Value, error) {
	return vectorDistance(this.metric, this.operands, item, context)
}

func vectorDistance(metric VectorMetric, operands Expressions, item value.Value, context Context) (value.Value, error) {
	var queryVec value.Value
	vec, err := operands[0].Evaluate(item, context)
	if err == nil {
		queryVec, err = operands[1].Evaluate(item, context)
	}
	if err != nil {
		return nil, err
	}
	if vec.Type() == value.MISSING || queryVec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}
	if vec.Type() != value.ARRAY || queryVec.Type() != value.ARRAY {
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
			case EUCLIDEAN, L2:
				dist += (df - qf) * (df - qf)
			case DOT_PRODUCT:
				dist += df * qf
			case COSINE_SIM:
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
	case DOT_PRODUCT:
		return value.NewValue(dist), nil
	case COSINE_SIM:
		return value.NewValue(1.0 - (dist / (math.Sqrt(sdf) * math.Sqrt(sqf)))), nil
	}
	return value.NULL_VALUE, nil
}

type Ann struct {
	FunctionBase
	metric VectorMetric
}

func NewAnn(metric VectorMetric, operands Expressions) Function {
	var name string
	switch metric {
	case EUCLIDEAN:
		name = "approx_euclidean_dist"
	case L2:
		name = "approx_l2_dist"
	case COSINE_SIM:
		name = "approx_cosine_sim_dist"
	case DOT_PRODUCT:
		name = "approx_dot_product_dist"
	case DEF_DIST:
		name = "approx_vector_dist"
	case EMPTY_DIST:
		name = "ann"
	default:
		name = "ann"
	}

	if metric == EMPTY_DIST || metric == DEF_DIST {
		// get metric (3rd argument)
		// MinArgs() ensures len(operands) >= 3
		if ev := operands[2].Value(); ev != nil {
			if ev.Type() == value.STRING {
				metric = GetVectorMetric(ev.ToString())
			}
		}
	}

	rv := &Ann{
		metric: metric,
	}
	rv.Init(name, operands...)
	rv.expr = rv
	return rv
}

func (this *Ann) Copy() Expression {
	rv := &Ann{
		metric: this.metric,
	}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	return rv
}

func (this *Ann) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Ann) PropagatesNull() bool { return false }
func (this *Ann) Indexable() bool      { return false }
func (this *Ann) Type() value.Type     { return value.NUMBER }
func (this *Ann) MinArgs() int {
	if this.metric == EMPTY_DIST || this.metric == DEF_DIST {
		return 3
	}
	return 2
}

func (this *Ann) MaxArgs() int {
	return this.MinArgs() + 2
}

func (this *Ann) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAnn(this.metric, operands)
	}
}

func (this *Ann) ValidOperands() error {
	switch this.metric {
	case EUCLIDEAN, L2, COSINE_SIM, DOT_PRODUCT:
		return nil
	}
	return errors.NewVectorFuncInvalidMetric(this.name, string(this.metric))
}

func (this *Ann) Metric() VectorMetric {
	return this.metric
}

func (this *Ann) QueryVector() Expression {
	return this.operands[1]
}

func (this *Ann) Nprobes() Expression {
	switch this.name {
	case "ann", "approx_vector_dist":
		if len(this.operands) > 3 {
			return this.operands[3]
		}
	case "approx_euclidean_dist", "approx_l2_dist", "approx_cosine_sim_dist", "approx_dot_product_dist":
		if len(this.operands) > 2 {
			return this.operands[2]
		}
	}
	return nil
}

func (this *Ann) ActualVector() Expression {
	switch this.name {
	case "ann", "approx_vector_dist":
		if len(this.operands) > 4 {
			return this.operands[4]
		}
	case "approx_euclidean_dist", "approx_l2_dist", "approx_cosine_sim_dist", "approx_dot_product_dist":
		if len(this.operands) > 3 {
			return this.operands[3]
		}
	}
	return nil
}

func (this *Ann) Evaluate(item value.Value, context Context) (value.Value, error) {
	return vectorDistance(this.metric, this.operands, item, context)
}

type IsVector struct {
	FunctionBase
	dimension int
}

func NewIsVector(operands Expressions) Function {
	rv := &IsVector{}
	rv.Init("isvector", operands...)
	rv.expr = rv
	return rv
}

func (this *IsVector) Copy() Expression {
	rv := &IsVector{}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	return rv
}

func (this *IsVector) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *IsVector) Type() value.Type { return value.BOOLEAN }
func (this *IsVector) MinArgs() int {
	return 3
}

func (this *IsVector) MaxArgs() int {
	return this.MinArgs()
}

func (this *IsVector) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIsVector(operands)
	}
}

func (this *IsVector) getDimension(ev value.Value) (int, error) {
	if ev.Type() == value.NUMBER {
		if n, nok := value.IsIntValue(ev); nok && n > 0 {
			return int(n), nil
		}
	}
	return 0, errors.NewIsVectorInvalidDimension(ev.ToString())
}

func (this *IsVector) ValidOperands() (err error) {
	ev := this.operands[1].Value()
	if ev != nil {
		this.dimension, err = this.getDimension(ev)
		if err != nil {
			return err
		}
	} else if esv := this.operands[1].Static(); esv == nil {
		return errors.NewIsVectorInvalidArg("2nd argument must be constant or positional/named parameter")
	}
	ev = this.operands[2].Value()
	if ev == nil || ev.Type() != value.STRING || strings.ToLower(ev.ToString()) != "float32" {
		return errors.NewIsVectorInvalidArg("3rd argument must be 'float32'")
	}
	return nil
}

func (this *IsVector) Evaluate(item value.Value, context Context) (value.Value, error) {
	vec, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if vec.Type() != value.ARRAY {
		return value.FALSE_VALUE, nil
	}
	if this.dimension <= 0 {
		var dv value.Value
		dv, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
		this.dimension, err = this.getDimension(dv)
		if err != nil {
			return nil, err
		}
	}

	for i := 0; ; i++ {
		v, ok := vec.Index(i)
		if !ok {
			if i == this.dimension {
				return value.TRUE_VALUE, nil
			} else {
				return value.FALSE_VALUE, nil
			}
		} else if i >= this.dimension {
			return value.FALSE_VALUE, nil
		} else {
			if v.Type() != value.NUMBER {
				return value.FALSE_VALUE, nil
			}
			vf := value.AsNumberValue(v).Float64()
			if vf < -math.MaxFloat32 || vf > math.MaxFloat32 {
				return value.FALSE_VALUE, nil
			}
		}
	}

}
