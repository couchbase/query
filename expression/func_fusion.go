//  Copyright 2025-Present Couchbase, Inc.
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
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type RankFusionValue struct {
	val   value.Value
	score float64
}

type FusionArgOptions struct {
	aggregator    string
	normalization string
	pathId        string
	pathScore     string
	weight        float64
	penalty       float64
	isScore       bool
}

type FusionBase struct {
	sync.Mutex
	fusion      string
	scorer      string
	limit       int
	score       bool
	argsOptions []*FusionArgOptions
	args        []map[string]float64
	index       int
	errs        []error
}

const (
	_RECIPROCAL_FUSION_FN_NAME         = "reciprocal_fusion"
	_RECIPROCAL_FUSION_MIN_ARGS        = 3
	_RECIPROCAL_FUSION_MAX_ARGS        = 17
	_FUSION_UNION_ALL                  = "unionall"
	_FUSION_UNION                      = "union"
	_FUSION_RRF                        = "rrf"
	_FUSION_RSF                        = "rsf"
	_FUSION_SUM                        = "sum"
	_FUSION_MIN                        = "min"
	_FUSION_MAX                        = "max"
	_FUSION_AVG                        = "avg"
	_FUSION_MEDIAN                     = "median"
	_FUSION_NORMALIZATION_NONE         = "none"
	_FUSION_NORMALIZATION_MINMAXSCALER = "minmaxscaler"
	_FUSION_NORMALIZATION_SIGMOID      = "sigmoid"
	_FUSION_RRF_PENALTY                = 60
	_FUSION_PATHID                     = "id"
	_FUSION_PATHSCORE                  = "score"

	_FUSION_FUSION_FN            = "fusion"
	_FUSION_SCORER_FN            = "scorer"
	_FUSION_SCORE_FN             = "score"
	_FUSION_LIMIT_FN             = "limit"
	_FUSION_ARGS_FN              = "args"
	_FUSION_ARG_ARGGREGATOR_FN   = "aggregator"
	_FUSION_ARG_NORMALIZATION_FN = "normalization"
	_FUSION_ARG_WEIGHT_FN        = "weight"
	_FUSION_ARG_PENALTY_FN       = "penalty"
	_FUSION_ARG_PATHID_FN        = "pathId"
	_FUSION_ARG_PATHSCORE_FN     = "pathScore"
	_FUSION_ARG_ISSCORE_FN       = "isScore"

	_FUSION_ARRAY_SIZE = 32
)

/*
{  "fusion"         : "unionall",
   "scorer"         : "RRF|RSF|SUM|AVG|MIN|MAX|MEDIAN",
   "score"          : true, (score, score_1, score_2, score_3, ......)
   "limit"          :  <number>,
   "args"           : [ {
                        "aggregator": "SUM|MIN|MAX|AVG|MEDIAN"
                        "normalization: "none|sigmoid|minMaxScaler",
                        "weight"  : weight,
                        "penalty" :  60, -- "penalty of vector ranking for RRF"
                        "pathId"  : "id", --- document field id
                        "pathScore": "score",  --- score/distance field
                        "score": false  means pathScore is distance score = (1- pathScore/maxPathScore)
                        },
                        {.....}
                      ]
  }
*/

func (this *FusionBase) GetOptions(nargs int, item value.Value) error {
	this.fusion = _FUSION_UNION_ALL
	this.scorer = _FUSION_RRF
	this.score = false
	this.limit = 0
	this.argsOptions = make([]*FusionArgOptions, nargs)
	for i := 0; i < nargs; i++ {
		this.argsOptions[i] = &FusionArgOptions{aggregator: _FUSION_SUM,
			normalization: _FUSION_NORMALIZATION_NONE,
			weight:        1.0,
			penalty:       _FUSION_RRF_PENALTY,
			pathId:        _FUSION_PATHID,
			pathScore:     _FUSION_PATHSCORE}
	}
	if item != nil && item.Type() == value.OBJECT {
		for f, v1 := range item.Fields() {
			v := value.NewValue(v1)
			switch f {
			case _FUSION_FUSION_FN:
				if v.Type() != value.STRING {
					return fmt.Errorf("'%s' must be string", f)
				}
				s := strings.ToLower(v.ToString())
				switch s {
				case _FUSION_UNION_ALL:
				default:
					return fmt.Errorf("'%s' must be '%s'", f, _FUSION_UNION_ALL)
				}
				this.fusion = s
			case _FUSION_LIMIT_FN:
				if v.Type() != value.NUMBER {
					return fmt.Errorf("'%s' must be number", f)
				}
				if n, nok := value.IsIntValue(v); nok && n > 0 {
					this.limit = int(n)
				} else {
					return fmt.Errorf("'%s' must be positive integer", f)
				}
			case _FUSION_SCORER_FN:
				if v.Type() != value.STRING {
					return fmt.Errorf("'%s' must be string", f)
				}
				s := strings.ToLower(v.ToString())
				switch s {
				case _FUSION_RRF, _FUSION_RSF:
				case _FUSION_SUM, _FUSION_MIN, _FUSION_MAX, _FUSION_AVG, _FUSION_MEDIAN:
				default:
					return fmt.Errorf("'%s' must be '%s|%s|%s|%s|%s|%s|%s'",
						f, _FUSION_RRF, _FUSION_RSF, _FUSION_SUM, _FUSION_MIN, _FUSION_MAX,
						_FUSION_AVG, _FUSION_MEDIAN)
				}
				this.scorer = s
			case _FUSION_SCORE_FN:
				if v.Type() != value.BOOLEAN {
					return fmt.Errorf("'%s' must be boolean", f)
				}
				this.score = v.Truth()
			case _FUSION_ARGS_FN:
				if v.Type() != value.ARRAY {
					return fmt.Errorf("'%s' must be ARRAY", f)
				}
				idx := 0
				av, avok := v.Index(idx)
				for avok {
					if av.Type() != value.OBJECT {
						return fmt.Errorf("'%s' must be ARRAY of objects", f)
					}
					for aof, aov1 := range av.Fields() {
						aov := value.NewValue(aov1)
						switch aof {
						case _FUSION_ARG_ARGGREGATOR_FN:
							if aov.Type() != value.STRING {
								return fmt.Errorf("args[%v]: '%s' must be string", idx, aof)
							}
							s := strings.ToLower(aov.ToString())
							switch s {
							case _FUSION_SUM, _FUSION_MIN, _FUSION_MAX, _FUSION_AVG, _FUSION_MEDIAN:
							default:
								return fmt.Errorf("args[%v]: '%s' must be '%s|%s|%s|%s|%s'",
									idx, aof, _FUSION_SUM, _FUSION_MIN, _FUSION_MAX,
									_FUSION_AVG, _FUSION_MEDIAN)
							}
							this.argsOptions[idx].aggregator = s
						case _FUSION_ARG_NORMALIZATION_FN:
							if aov.Type() != value.STRING {
								return fmt.Errorf("args[%v]: '%s' must be string", idx, aof)
							}
							s := strings.ToLower(aov.ToString())
							if s == "" {
								s = _FUSION_NORMALIZATION_NONE
							}
							switch s {
							case _FUSION_NORMALIZATION_NONE, _FUSION_NORMALIZATION_MINMAXSCALER, _FUSION_NORMALIZATION_SIGMOID:
							default:
								return fmt.Errorf("args[%v]: '%s' must be '%s|%s|%s'",
									idx, aof,
									_FUSION_NORMALIZATION_NONE,
									_FUSION_NORMALIZATION_MINMAXSCALER,
									_FUSION_NORMALIZATION_SIGMOID)
							}
							this.argsOptions[idx].normalization = s
						case _FUSION_ARG_WEIGHT_FN, _FUSION_ARG_PENALTY_FN:
							if aov.Type() != value.NUMBER {
								return fmt.Errorf("args[%v]: '%s' must be number", idx, aof)
							}
							if n := value.AsNumberValue(aov).Float64(); n > 0 {
								if aof == _FUSION_ARG_WEIGHT_FN {
									this.argsOptions[idx].weight = n
								} else {
									this.argsOptions[idx].penalty = n
								}
							} else {
								return fmt.Errorf("args[%v]: '%s' must be positive number", idx, aof)
							}
						case _FUSION_ARG_PATHID_FN, _FUSION_ARG_PATHSCORE_FN:
							if aov.Type() != value.STRING || aov.ToString() == "" {
								return fmt.Errorf("args[%v]: '%s' must be non-empty string", idx, aof)
							}
							if aof == _FUSION_ARG_PATHID_FN {
								this.argsOptions[idx].pathId = aov.ToString()
							} else {
								this.argsOptions[idx].pathScore = aov.ToString()
							}
						case _FUSION_ARG_ISSCORE_FN:
							if aov.Type() != value.BOOLEAN {
								return fmt.Errorf("args[%v]: '%s' must be boolean", idx, aof)
							}
							this.argsOptions[idx].isScore = aov.Truth()
						}
					}

					idx++
					av, avok = v.Index(idx)
				}
			}
		}
	}
	return nil
}

func (this *FusionBase) TransformResult(smv []map[string]float64) map[string][]float64 {
	rv := make(map[string][]float64, _FUSION_ARRAY_SIZE)
	for _, mv := range smv {
		for id, v := range mv {
			if _, ok := rv[id]; !ok {
				rv[id] = make([]float64, 0, 2)
				rv[id] = append(rv[id], float64(0))
			}
			rv[id] = append(rv[id], v)
		}
	}
	return rv
}

func (this *FusionBase) Aggregator(aggType string, mrv map[string][]float64) map[string]float64 {
	for _, ascores := range mrv {
		switch aggType {
		case _FUSION_RRF, _FUSION_RSF, _FUSION_SUM, _FUSION_AVG:
			for _, f := range ascores[1:] {
				ascores[0] += f
			}
			if aggType == _FUSION_AVG && len(ascores) > 1 {
				ascores[0] = ascores[0] / float64(len(ascores)-1)
			}
		case _FUSION_MIN:
			minScore := float64(math.MaxFloat64)
			for _, score := range ascores[1:] {
				if score < minScore {
					minScore = score
				}
			}
			ascores[0] = minScore
		case _FUSION_MAX:
			maxScore := float64(math.SmallestNonzeroFloat64)
			for _, score := range ascores[1:] {
				if score > maxScore {
					maxScore = score
				}
			}
			ascores[0] = maxScore
		case _FUSION_MEDIAN:
			sort.Sort(sort.Float64Slice(ascores[1:]))
			l := len(ascores[1:])
			ascores[0] = ascores[1+l/2]
			if l%2 == 0 {
				ascores[0] = (ascores[0] + ascores[l/2]) / 2.0 // we need to skip 0
			}
		}
	}
	rv := make(map[string]float64, len(mrv))
	for s, af := range mrv {
		rv[s] = af[0]
	}
	return rv
}

func (this *FusionBase) GetResults(index int, item value.Value) (map[string]float64, error) {
	if item.Type() != value.ARRAY {
		return nil, errors.NewEvaluationWithCauseError(fmt.Errorf("'%v' results must be ARRAY", index+1),
			_RECIPROCAL_FUSION_FN_NAME)
	}
	argOptions := this.argsOptions[index]
	rv := make(map[string][]float64, _FUSION_ARRAY_SIZE)
	scores := make([]float64, 0, _FUSION_ARRAY_SIZE)
	ids := make([]string, 0, _FUSION_ARRAY_SIZE)
	switch this.scorer {
	case _FUSION_RRF:
		for valIdx := 0; ; valIdx++ {
			if val, valOk := item.Index(valIdx); valOk {
				if idVal, idOk := val.Field(argOptions.pathId); idOk &&
					idVal.Type() == value.STRING && idVal.ToString() != "" {
					ids = append(ids, idVal.ToString())
					scores = append(scores, (1.0 / (argOptions.penalty + float64(valIdx))))
				} else {
					return nil, errors.NewEvaluationWithCauseError(
						fmt.Errorf("'%v' results Objects must have '%s'",
							index+1, argOptions.pathId), _RECIPROCAL_FUSION_FN_NAME)
				}
			} else {
				break
			}
		}
	case _FUSION_RSF, _FUSION_SUM, _FUSION_AVG, _FUSION_MIN, _FUSION_MAX, _FUSION_MEDIAN:
		isScore := argOptions.isScore
		for valIdx := 0; ; valIdx++ {
			if val, valOk := item.Index(valIdx); valOk {
				if idVal, idOk := val.Field(argOptions.pathId); idOk &&
					idVal.Type() == value.STRING && idVal.ToString() != "" {
					ids = append(ids, idVal.ToString())
				} else {
					return nil, errors.NewEvaluationWithCauseError(
						fmt.Errorf("'%v' results Objects must have '%s'",
							index+1, argOptions.pathId), _RECIPROCAL_FUSION_FN_NAME)
				}
				if scoreVal, scoreOk := val.Field(argOptions.pathScore); scoreOk &&
					scoreVal.Type() == value.NUMBER {
					scores = append(scores, value.AsNumberValue(scoreVal).Float64())
				} else {
					return nil, errors.NewEvaluationWithCauseError(
						fmt.Errorf("'%v' results Objects must have '%s'",
							index+1, argOptions.pathScore), _RECIPROCAL_FUSION_FN_NAME)
				}
			} else {
				break
			}
		}
		if !isScore {
			// convert distance to score max normalization (1 minus normalized value)
			maxScore := float64(math.SmallestNonzeroFloat64)
			negative := false
			for i, score := range scores {
				if score < 0 {
					negative = true
					score = -score
					scores[i] = score
				}
				if score > maxScore {
					maxScore = score
				}
			}
			for i, _ := range scores {
				if maxScore > 0 {
					scores[i] = scores[i] / maxScore
				}
				if !negative {
					scores[i] = 1.0 - scores[i]
				}
			}
		}
		switch argOptions.normalization {
		case _FUSION_NORMALIZATION_NONE:
		case _FUSION_NORMALIZATION_MINMAXSCALER, _FUSION_NORMALIZATION_SIGMOID:
			minScore := float64(math.MaxFloat64)
			maxScore := float64(math.SmallestNonzeroFloat64)
			for _, score := range scores {
				if score < minScore {
					minScore = score
				}
				if score > maxScore {
					maxScore = score
				}
			}
			divScore := maxScore - minScore
			if divScore == 0 {
				divScore = maxScore
			}
			if argOptions.normalization == _FUSION_NORMALIZATION_SIGMOID {
				divScore = maxScore
				minScore = 0
			}
			for i, _ := range scores {
				if divScore > 0 {
					scores[i] = (scores[i] - minScore) / divScore
				}
				if argOptions.normalization == _FUSION_NORMALIZATION_SIGMOID {
					score := scores[i]
					scores[i] = 1.0 / (1.0 + math.Exp(-score))
				}
			}
		}
	}

	// dedup into slice, leave 0th pos for aggregator, also multiply with weight
	for i, id := range ids {
		if _, ok := rv[id]; !ok {
			rv[id] = make([]float64, 0, 2)
			rv[id] = append(rv[id], float64(0))
		}
		rv[id] = append(rv[id], argOptions.weight*scores[i])
	}
	return this.Aggregator(argOptions.aggregator, rv), nil
}

func (this *FusionBase) parallelArgEvaluate(index int, wg *sync.WaitGroup, operand Expression,
	item value.Value, context Context) {
	defer func() {
		r := recover()
		if r != nil {
			logging.Stackf(logging.ERROR, "Parallel execution of ReciprocalFusion Panic: %v", r)
		}
	}()
	defer wg.Done()
	var mrv map[string]float64
	if ctx, ok := context.(interface{ CopyOpContext() interface{} }); ok {
		// context
		if lctx, ok := ctx.CopyOpContext().(Context); ok {
			context = lctx
		}
	}

	rv, err := operand.Evaluate(item, context)
	if err == nil {
		mrv, err = this.GetResults(index, rv)
	}
	if err != nil {
		this.Lock()
		defer this.Unlock()
		this.errs = append(this.errs, err)
		return
	}

	this.Lock()
	defer this.Unlock()
	this.args[index] = mrv
}

func (this *FusionBase) evaluate(operands Expressions, item value.Value, context Context) (value.Value, error) {
	wg := &sync.WaitGroup{}
	this.args = make([]map[string]float64, len(operands)-1)
	for i, op := range operands {
		if i == 0 {
			optionsVal, err := op.Evaluate(item, context)
			if err != nil {
				return nil, err
			} else if optionsVal.Type() == value.MISSING {
				return value.MISSING_VALUE, nil
			} else if optionsVal.Type() != value.OBJECT {
				return value.NULL_VALUE, nil
			} else if err1 := this.GetOptions(len(operands)-1, optionsVal); err1 != nil {
				return nil, errors.NewEvaluationWithCauseError(err1, _RECIPROCAL_FUSION_FN_NAME)
			}
		} else {
			wg.Add(1)
			go this.parallelArgEvaluate(i-1, wg, op, item, context)
		}
	}
	wg.Wait()
	if len(this.errs) > 0 {
		return nil, this.errs[0]
	}

	return nil, nil
}

type ReciprocalFusion struct {
	FunctionBase
	FusionBase
}

func NewReciprocalFusion(operands ...Expression) Function {
	rv := &ReciprocalFusion{}
	rv.Init(_RECIPROCAL_FUSION_FN_NAME, operands...)

	rv.expr = rv
	return rv
}

func (this *ReciprocalFusion) Copy() Expression {
	rv := &ReciprocalFusion{}
	rv.Init(this.name, this.operands.Copy()...)
	rv.expr = rv
	rv.BaseCopy(this)
	return rv
}

func (this *ReciprocalFusion) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *ReciprocalFusion) PropagatesNull() bool { return false }
func (this *ReciprocalFusion) Indexable() bool      { return false }
func (this *ReciprocalFusion) Type() value.Type     { return value.ARRAY }
func (this *ReciprocalFusion) MinArgs() int         { return _RECIPROCAL_FUSION_MIN_ARGS }
func (this *ReciprocalFusion) MaxArgs() int         { return _RECIPROCAL_FUSION_MAX_ARGS }

func (this *ReciprocalFusion) Constructor() FunctionConstructor {
	return NewReciprocalFusion
}

func (this *ReciprocalFusion) Evaluate(item value.Value, context Context) (value.Value, error) {

	rv, err := this.evaluate(this.operands, item, context)
	if err != nil || rv != nil {
		return rv, err
	}
	msv := this.TransformResult(this.args)
	mv := this.Aggregator(this.scorer, msv)
	rfvs := make([]*RankFusionValue, 0, len(mv))
	l := 1
	if this.score {
		l += len(this.args) + 1
	}
	for id, score := range mv {
		obj := make(map[string]interface{}, l)
		obj[_FUSION_PATHID] = id
		if this.score {
			obj[_FUSION_PATHSCORE] = score
			for i, amv := range this.args {
				if vscore, ok := amv[id]; ok {
					obj[_FUSION_PATHSCORE+"_"+strconv.Itoa(i)] = vscore
				}
			}
		}
		rfvs = append(rfvs, &RankFusionValue{val: value.NewValue(obj), score: score})
	}

	scoreFn := func(p1, p2 *RankFusionValue) bool {
		return p1.score < p2.score
	}
	decreasingScoreFn := func(p1, p2 *RankFusionValue) bool {
		return scoreFn(p2, p1)
	}
	By(decreasingScoreFn).Sort(rfvs)

	limit := this.limit
	if limit == 0 {
		limit = len(rfvs)
	}
	srv := make(value.Values, 0, limit)
	for i, rfv := range rfvs {
		if i >= limit {
			break
		}
		srv = append(srv, rfv.val)
	}
	return value.NewValue(srv), nil
}

type By func(p1, p2 *RankFusionValue) bool

type RankFusionValueSorter struct {
	rankFusionValues []*RankFusionValue
	by               func(p1, p2 *RankFusionValue) bool
}

func (s *RankFusionValueSorter) Len() int {
	return len(s.rankFusionValues)
}

func (s *RankFusionValueSorter) Swap(i, j int) {
	s.rankFusionValues[i], s.rankFusionValues[j] = s.rankFusionValues[j], s.rankFusionValues[i]
}

func (s *RankFusionValueSorter) Less(i, j int) bool {
	return s.by(s.rankFusionValues[i], s.rankFusionValues[j])
}

func (by By) Sort(rankFusionValues []*RankFusionValue) {
	rs := &RankFusionValueSorter{
		rankFusionValues: rankFusionValues,
		by:               by,
	}
	sort.Sort(rs)
}
