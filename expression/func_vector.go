//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
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

type VectorDistance struct {
	FunctionBase
	metric VectorMetric
	flags  uint32
}

func NewVectorDistance(operands Expressions) Function {
	var metric VectorMetric
	// get metric (3rd argument)
	// MinArgs()/MaxArgs() ensures len(operands) == 3
	ev := operands[2].Value()
	if ev != nil && ev.Type() == value.STRING {
		metric = GetVectorMetric(ev.ToString())
	}

	rv := &VectorDistance{
		metric: metric,
	}
	rv.Init("vector_distance", operands...)
	rv.expr = rv
	return rv
}

func (this *VectorDistance) Copy() Expression {
	rv := &VectorDistance{
		metric: this.metric,
		flags:  this.flags,
	}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	rv.BaseCopy(this)
	return rv
}

func (this *VectorDistance) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *VectorDistance) PropagatesNull() bool { return false }
func (this *VectorDistance) Indexable() bool      { return false }
func (this *VectorDistance) Type() value.Type     { return value.NUMBER }
func (this *VectorDistance) MinArgs() int {
	return 3
}

func (this *VectorDistance) MaxArgs() int {
	return 3
}

func (this *VectorDistance) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewVectorDistance(operands)
	}
}

func (this *VectorDistance) ValidOperands() error {
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

func (this *VectorDistance) Metric() VectorMetric {
	return this.metric
}

func (this *VectorDistance) Field() Expression {
	return this.operands[0]
}

func (this *VectorDistance) QueryVector() Expression {
	return this.operands[1]
}

func (this *VectorDistance) Evaluate(item value.Value, context Context) (value.Value, error) {
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

type ApproxVectorDistance struct {
	FunctionBase
	metric VectorMetric
	flags  uint32
}

func NewApproxVectorDistance(operands Expressions) Function {
	var metric VectorMetric
	// get metric (3rd argument)
	// MinArgs() ensures len(operands) >= 3
	if ev := operands[2].Value(); ev != nil {
		if ev.Type() == value.STRING {
			metric = GetVectorMetric(ev.ToString())
		}
	}

	rv := &ApproxVectorDistance{
		metric: metric,
	}
	rv.Init("approx_vector_distance", operands...)
	rv.expr = rv
	return rv
}

func (this *ApproxVectorDistance) Copy() Expression {
	rv := &ApproxVectorDistance{
		metric: this.metric,
		flags:  this.flags,
	}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	rv.BaseCopy(this)
	return rv
}

func (this *ApproxVectorDistance) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ApproxVectorDistance) PropagatesNull() bool { return false }
func (this *ApproxVectorDistance) Indexable() bool      { return false }
func (this *ApproxVectorDistance) Type() value.Type     { return value.NUMBER }
func (this *ApproxVectorDistance) MinArgs() int {
	return 3
}

func (this *ApproxVectorDistance) MaxArgs() int {
	return 5
}

func (this *ApproxVectorDistance) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewApproxVectorDistance(operands)
	}
}

func (this *ApproxVectorDistance) ValidOperands() error {
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

func (this *ApproxVectorDistance) Metric() VectorMetric {
	return this.metric
}

func (this *ApproxVectorDistance) Field() Expression {
	return this.operands[0]
}

func (this *ApproxVectorDistance) QueryVector() Expression {
	return this.operands[1]
}

func (this *ApproxVectorDistance) Nprobes() Expression {
	if len(this.operands) > 3 {
		return this.operands[3]
	}
	return nil
}

func (this *ApproxVectorDistance) ReRank() Expression {
	if len(this.operands) > 4 {
		return this.operands[4]
	}
	return nil
}

func (this *ApproxVectorDistance) NeedSquareRoot() bool {
	return this.metric == EUCLIDEAN || this.metric == L2
}

func (this *ApproxVectorDistance) Evaluate(item value.Value, context Context) (value.Value, error) {
	if (this.flags & _VECTOR_QVEC_CHECKED) == 0 {
		rv, err := vectorDistance(this.metric, this.operands, true, item, context)
		this.flags |= _VECTOR_QVEC_CHECKED
		return rv, err
	}
	return vectorDistance(this.metric, this.operands, false, item, context)
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
	rv.BaseCopy(this)
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
	if valid, _ := validVector(vec, this.dimension); valid {
		return value.TRUE_VALUE, nil
	}
	return value.FALSE_VALUE, nil
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

type DecodeVector struct {
	FunctionBase
}

func NewDecodeVector(operands ...Expression) Function {
	rv := &DecodeVector{}
	rv.Init("decode_vector", operands...)

	rv.expr = rv
	return rv
}

func (this *DecodeVector) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *DecodeVector) Type() value.Type { return value.ARRAY }
func (this *DecodeVector) MinArgs() int     { return 1 }
func (this *DecodeVector) MaxArgs() int     { return 2 }

func (this *DecodeVector) Evaluate(item value.Value, context Context) (value.Value, error) {
	var decodeStr string
	var byteOrder binary.ByteOrder
	byteOrder = binary.LittleEndian
	null := false
	missing := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !null && !missing {
			if i == 0 {
				if arg.Type() != value.STRING {
					null = true
				} else {
					decodeStr = arg.ToString()
				}
			} else {
				if arg.Type() != value.BOOLEAN {
					null = true
				} else if arg.Truth() {
					byteOrder = binary.BigEndian
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	// We first decode the encoded string into a byte array.
	// The array is expected to be divisible by FLOAT32_SIZE because each float32
	// should occupy 4 bytes
	decodedString, err := base64.StdEncoding.DecodeString(decodeStr)
	if err != nil || len(decodedString)%FLOAT32_SIZE != 0 {
		return value.NULL_VALUE, nil
	}

	dims := int(len(decodedString) / FLOAT32_SIZE)
	decodedVector := make([]interface{}, dims)

	offset := 0
	// We iterate through the array 4 bytes at a time and convert each of
	// them to a float32 value by reading them in a little endian notation
	for i := 0; i < dims; i++ {
		decodedVector[i] = math.Float32frombits(byteOrder.Uint32(decodedString[offset : offset+FLOAT32_SIZE]))
		offset += FLOAT32_SIZE
	}

	return value.NewValue(decodedVector), nil
}

func (this *DecodeVector) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewDecodeVector(operands...)
	}
}

type EncodeVector struct {
	FunctionBase
}

func NewEncodeVector(operands ...Expression) Function {
	rv := &EncodeVector{}
	rv.Init("encode_vector", operands...)

	rv.expr = rv
	return rv
}

func (this *EncodeVector) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *EncodeVector) Type() value.Type { return value.STRING }
func (this *EncodeVector) MinArgs() int     { return 1 }
func (this *EncodeVector) MaxArgs() int     { return 2 }

func (this *EncodeVector) Evaluate(item value.Value, context Context) (value.Value, error) {
	var vec value.Value
	var byteOrder binary.ByteOrder
	byteOrder = binary.LittleEndian
	null := false
	missing := false

	for i, op := range this.operands {
		arg, err := op.Evaluate(item, context)
		if err != nil {
			return nil, err
		} else if arg.Type() == value.MISSING {
			missing = true
		} else if arg.Type() == value.NULL {
			null = true
		} else if !null && !missing {
			if i == 0 {
				if arg.Type() != value.ARRAY {
					null = true
				} else {
					vec = arg
				}
			} else {
				if arg.Type() != value.BOOLEAN {
					null = true
				} else if arg.Truth() {
					byteOrder = binary.BigEndian
				}
			}
		}
	}

	if missing {
		return value.MISSING_VALUE, nil
	} else if null {
		return value.NULL_VALUE, nil
	}

	buf := new(bytes.Buffer)
	for i := 0; ; i++ {
		v, ok := vec.Index(i)
		if !ok {
			break
		} else {
			if v.Type() != value.NUMBER {
				return value.NULL_VALUE, nil
			}
			f := value.AsNumberValue(v).Float64()
			if f < -math.MaxFloat32 || f > math.MaxFloat32 {
				return value.NULL_VALUE, nil
			}
			// Convert each float64 to float32 since we're working with 32-bit floating points.
			if err := binary.Write(buf, byteOrder, float32(f)); err != nil {
				return value.NULL_VALUE, nil
			}
		}
	}

	// Encode the byte slice to Base64.
	str := base64.StdEncoding.EncodeToString(buf.Bytes())
	return value.NewValue(str), nil
}

func (this *EncodeVector) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewEncodeVector(operands...)
	}
}

type NormalizeVector struct {
	UnaryFunctionBase
}

func NewNormalizeVector(operands Expression) Function {
	rv := &NormalizeVector{}
	rv.Init("normalize_vector", operands)

	rv.expr = rv
	return rv
}

func (this *NormalizeVector) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *NormalizeVector) Type() value.Type { return value.ARRAY }
func (this *NormalizeVector) Indexable() bool  { return false }

func (this *NormalizeVector) Evaluate(item value.Value, context Context) (value.Value, error) {
	vec, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if vec.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if vec.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}
	fvec := make([]interface{}, 0, 32)
	var svf float64
	for i := 0; ; i++ {
		vv, vOk := vec.Index(i)
		if !vOk {
			break
		}
		if vv.Type() != value.NUMBER {
			return value.NULL_VALUE, nil
		}
		vf := value.AsNumberValue(vv).Float64()
		if vf < -math.MaxFloat32 || vf > math.MaxFloat32 {
			return value.NULL_VALUE, nil
		}
		fvec = append(fvec, vf)
		svf += vf * vf
	}
	svf = math.Sqrt(svf)
	if svf == float64(0) {
		return value.NULL_VALUE, nil
	}

	for i, vf := range fvec {
		fvec[i] = vf.(float64) / svf
	}
	return value.NewValue(fvec), nil
}

func (this *NormalizeVector) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewNormalizeVector(operands[0])
	}
}
