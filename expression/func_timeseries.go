//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"fmt"
	"strconv"

	"github.com/couchbase/query/value"
)

type TimeSeries struct {
	FunctionBase
	tsPaths map[string]string
}

const (
	_TSDATA     = "ts_data"
	_TSINTERVAL = "ts_interval"
	_TSSTART    = "ts_start"
	_TSEND      = "ts_end"
	_TSKEEP     = "ts_keep"
	_TSRANGES   = "ts_ranges"
	_TSPROJECT  = "ts_project"
)

var defTsPaths = []string{_TSDATA, _TSINTERVAL, _TSSTART, _TSEND}

func NewTimeSeries(operands ...Expression) Function {
	rv := &TimeSeries{}
	rv.FunctionBase = *NewFunctionBase("_timeseries", operands...)

	// Get the ts paths from options
	var options Expression
	if len(operands) > 1 {
		options = operands[1]
	}
	tsPaths, _, _, _, err := rv.GetOptionFields(options, nil, nil)
	if err == nil {
		rv.tsPaths = tsPaths
	}

	rv.expr = rv
	return rv
}

func (this *TimeSeries) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *TimeSeries) Copy() Expression {
	rv := &TimeSeries{}
	rv.FunctionBase = *NewFunctionBase("_timeseries", this.operands.Copy()...)
	rv.BaseCopy(this)
	rv.tsPaths = make(map[string]string, len(this.tsPaths))
	for f, v := range this.tsPaths {
		rv.tsPaths[f] = v
	}
	return rv
}

/*
returns type ARRAY
*/
func (this *TimeSeries) Type() value.Type { return value.ARRAY }

func (this *TimeSeries) Indexable() bool { return false }

// Validate options (semantic checks)

func (this *TimeSeries) ValidOperands() (err error) {
	// First argument must be path (only identifier used for keyspace)
	op := this.operands[0]
	a, _, e := PathString(op)
	if a == "" || e != nil {
		return fmt.Errorf("First argument must be keyspace alias.")
	}

	if len(this.operands) > 1 {
		// second argument must be OBJECT
		op = this.operands[1]
		val := op.Value()
		if (val != nil && val.Type() != value.OBJECT) || op.Static() == nil {
			return fmt.Errorf("Second argument must be OBJECT and can only contain constants or positional/named parameters.")
		}

		// second argument mandatory fields check
		_, _, _, _, err = this.GetOptionFields(op, nil, nil)
	}

	return err
}

// fields retrieval and validation. Also check mandatory fields
func (this *TimeSeries) GetOptionFields(arg Expression, item value.Value, context Context) (
	tsPaths map[string]string, tsKeep bool, tsRanges, tsProject value.Value, err error) {
	tsPaths = make(map[string]string, 4)
	if arg != nil {
		var options value.Value
		if context != nil {
			options, err = arg.Evaluate(item, context)
			if err != nil {
				return
			}
		} else {
			options = arg.Value()
		}
		if oc, ok := arg.(*ObjectConstruct); ok && options == nil {
			m := make(map[string]interface{}, 6)
			for name, val := range oc.Mapping() {
				n := name.Value()
				if n == nil || n.Type() != value.STRING {
					continue
				}
				s := n.ToString()
				switch s {
				case _TSDATA, _TSINTERVAL, _TSSTART, _TSEND:
					v := val.Value()
					if v == nil {
						err = fmt.Errorf("'%s' in the second argument must be constant.", s)
						return
					}
					if v.Type() > value.MISSING {
						m[s] = v.Actual()
					}
				case _TSKEEP, _TSRANGES, _TSPROJECT:
					v := val.Value()
					if v != nil && v.Type() > value.MISSING {
						m[s] = v.Actual()
					}
				}
			}
			options = value.NewValue(m)
		}
		if options.Type() == value.OBJECT {
			for f, v1 := range options.Fields() {
				v := value.NewValue(v1)
				switch f {
				case _TSDATA, _TSINTERVAL, _TSSTART, _TSEND:
					if v.Type() == value.STRING {
						tsPaths[f] = v.ToString()
					} else {
						err = fmt.Errorf("'%s' must be string in the second argument.", f)
						return
					}
				case _TSKEEP:
					if v.Type() == value.BOOLEAN {
						tsKeep = v.Truth()
					}
				case _TSRANGES:
					tsRanges = v
				case _TSPROJECT:
					tsProject = v
				}
			}
		}
	}

	for _, s := range defTsPaths {
		p, ok := tsPaths[s]
		if !ok {
			tsPaths[s] = s
		} else if p == "" {
			err = fmt.Errorf("'%s' invalid in the second argument.", s)
			break
		}
	}
	return
}

// keyspace name
func (this *TimeSeries) AliasName() string {
	a, _, _ := PathString(this.operands[0])
	return a
}

// path names strings from options to retrive from the document
func (this *TimeSeries) TsPaths() map[string]string {
	return this.tsPaths
}

/*
 config = _timeseries(alias, {"ts_data":"timeline",
                              "ts_interval":"interval",
			      "ts_start":"dstart",
			      "ts_end":"dend",
			      "ts_keep":true,
			      "ts_ranges":$ts_ranges})
	  $ts_ranges = [[1677730943000, 1677730948000], [1677730954000, 1677730957000]]
	  $ts_ranges = [1677730943000, 1677730948000]
	  ts_keep, ts_ranges can be static (constant or parameters)
*/

func (this *TimeSeries) Evaluate(item value.Value, context Context) (value.Value, error) {
	// retrive tsRanges
	var tsRanges, tsProject value.Value
	var err error
	if len(this.operands) > 1 {
		_, _, tsRanges, tsProject, err = this.GetOptionFields(this.operands[1], item, context)
		if err != nil {
			return value.NULL_VALUE, err
		}
	}
	// construct ts data structure. It also construct expression from ts pathnames
	// This need to do every invocation becase no way to store this in function (except in context).
	// expression is shared across statements due to prepare statements.

	timeSeriesData, err := NewTimeSeriesData(this.AliasName(), this.tsPaths, false, tsRanges, tsProject, context)
	if err != nil {
		return value.NULL_VALUE, err
	}

	// Evaluate paths against document and store
	rv, err := timeSeriesData.Evaluate(item, context)
	if err != nil || (rv != nil && rv.Type() <= value.NULL) {
		return rv, err
	}

	var result []interface{}
	if ok, _ := timeSeriesData.Qualified(false); !ok {
		return value.NewValue(result), nil
	}

	var actv value.Value
	var ok bool
	idx := 0

	// iterate over time series data
	for {
		actv, idx, ok = timeSeriesData.GetNextValue(idx)
		if !ok {
			return value.NewValue(result), nil
		}
		result = append(result, actv)
	}
}

func (this *TimeSeries) MinArgs() int { return 1 }

func (this *TimeSeries) MaxArgs() int { return 2 }

func (this *TimeSeries) Constructor() FunctionConstructor {
	return NewTimeSeries
}

// Store all relavent info about timeseries data
type TimeSeriesData struct {
	tsPathsExpr  map[string]Expression
	tsData       value.Value
	tsInterval   int64
	tsStart      int64
	tsEnd        int64
	tsKeep       bool
	tsRanges     TimeSeriesRanges
	tsProjectAll bool
	tsProject    map[int]bool
}

type CheckAndSetPathValue func(*TimeSeriesData, value.Value) bool

var _CHECK_SET = map[string]CheckAndSetPathValue{
	_TSDATA: func(t *TimeSeriesData, v value.Value) bool {
		if v.Type() == value.ARRAY {
			t.tsData = v
			return true
		}
		return false
	},
	_TSINTERVAL: func(t *TimeSeriesData, v value.Value) bool {
		if v.Type() == value.MISSING {
			t.tsInterval = 0
			return true
		} else if n, nok := value.IsIntValue(v); nok {
			t.tsInterval = n
			return true
		}
		return false
	},
	_TSSTART: func(t *TimeSeriesData, v value.Value) bool {
		if n, nok := value.IsIntValue(v); nok {
			t.tsStart = n
			return true
		}
		return false
	},
	_TSEND: func(t *TimeSeriesData, v value.Value) bool {
		if n, nok := value.IsIntValue(v); nok {
			t.tsEnd = n
			return true
		}
		return false
	},
}

// Constrct TimeSeries Data and path expression where to look fields in the document
func NewTimeSeriesData(alias string, tsPaths map[string]string, tsKeep bool, tsRanges, tsProject value.Value,
	context Context) (*TimeSeriesData, error) {
	rv := &TimeSeriesData{}
	rv.tsKeep = tsKeep
	rv.tsPathsExpr = make(map[string]Expression)
	for f, p := range tsPaths {
		p = "`" + alias + "`" + "." + p
		expr, err := context.Parse(p)
		if err != nil {
			return nil, err
		}
		if e, ok := expr.(Expression); ok {
			rv.tsPathsExpr[f] = e
		}
	}
	rv.tsRanges = GetTimeSeriesRanges(tsRanges)
	rv.tsProjectAll, rv.tsProject = GetTimeSeriesProject(tsProject)
	return rv, nil
}

/*

"ts_project":  MISSING   project all values from data point
               0,        project 0th value from data point
               [0,2,3]  project 0th, 2nd, 3rd value from data point

*/

func GetTimeSeriesProject(tsProject value.Value) (all bool, rv map[int]bool) {
	if tsProject == nil {
		return true, nil
	}
	rv = make(map[int]bool, 4)
	if tsProject.Type() == value.ARRAY {
		idx := 0
		p, pok := tsProject.Index(0)
		for pok {
			if n, ok := value.IsIntValue(p); ok && n >= 0 {
				rv[int(n)] = true
			}
			idx++
			p, pok = tsProject.Index(idx)
		}
	} else if n, ok := value.IsIntValue(tsProject); ok && n >= 0 {
		rv[int(n)] = true
	}

	return false, rv
}

// Convert single range (Ignore invalid once) predicate array into Structure.

func getTimeSeriesRange(trange value.Value) (rv *TimeSeriesRange, skip bool) {
	s, sok := trange.Index(0)
	e, eok := trange.Index(1)
	if sok && eok {
		sn, snok := value.IsIntValue(s)
		en, enok := value.IsIntValue(e)
		if snok && enok && sn <= en {
			return NewTimeSeriesRange(sn, en), false
		}
	}
	return nil, true
}

// See the format near Evaluate()
// Convert multiple range (Ignore invalid once) predicate array into Structure.
// If all are invalid ranges then add range of -1 to 0.

func GetTimeSeriesRanges(tranges value.Value) (rv TimeSeriesRanges) {
	if tranges == nil || tranges.Type() == value.MISSING {
		return rv
	} else if tranges.Type() == value.ARRAY {
		var trange value.Value
		idx := 0
		ok := true
		for ok {
			trange, ok = tranges.Index(idx)
			if ok {
				if trange.Type() == value.NUMBER {
					tr, skip := getTimeSeriesRange(tranges)
					if !skip {
						rv = append(rv, tr)
					}
					ok = false
				} else if trange.Type() == value.ARRAY {
					tr, skip := getTimeSeriesRange(trange)
					if !skip {
						rv = append(rv, tr)
					}
					idx++
				} else {
					ok = false
				}
			}
		}
	}
	if len(rv) == 0 {
		rv = append(rv, NewTimeSeriesRange(-1, 0))
	}
	return rv
}

func (this *TimeSeriesData) TsKeep() bool {
	return this.tsKeep
}

func (this *TimeSeriesData) TsDataExpr() Expression {
	return this.tsPathsExpr[_TSDATA]
}
func (this *TimeSeriesData) ResetTsData() {
	this.tsData = nil
}

// Evaluate TimeSeries Data from the document
// on error consider as NULL
func (this *TimeSeriesData) Evaluate(item value.Value, context Context) (rv value.Value, err error) {
	var v value.Value
	for f, expr := range this.tsPathsExpr {
		v, err = expr.Evaluate(item, context)
		if err != nil {
			return value.NULL_VALUE, err
		}
		if fn, ok := _CHECK_SET[f]; ok && !fn(this, v) {
			return value.NULL_VALUE, nil
		}
	}
	return nil, nil
}

func (this *TimeSeriesData) AllData() bool {
	return this.tsRanges.All()
}

func (this *TimeSeriesData) Qualified(outer bool) (bool, bool) {
	//  out of range document with tsRanges predicate
	rv := this.tsRanges.Qualified(this.tsStart, this.tsEnd)
	if !rv && outer && this.tsRanges.All() {
		_, idx, ok := this.GetNextValue(0)
		if !ok && idx == 0 {
			return rv, outer
		}

	}
	return rv, false
}

/*
irregular data:

	timeline = [ [1677730930000, 16.30, {"x":1},.....], [1677730931000, [16.31, 16.311], ..... ]
	    output : {"_t": 1677730930000, "_v0": 16.30, "_v1": {"x":1},....}
	             {"_t": 1677730930000, "_v0": [16.31, 16.311] }

regular data :

	timeline = [ 16.30, [16.31, 16.311], {"v":[16.31, 16.311]} ,..... ] (manadtory ts_interval, ts_start, ts_end)
	    output : {"_t": ts_start, "_v0": 16.30}
	             {"_t": ts_start+1*ts_interval, "_v0": 16.31, "v1":16.311}
	             {"_t": ts_start+2*ts_interval, "_v0": {"v":[16.31, 16.311]}}
*/

// convert single data point into OBJECT

func (this *TimeSeriesData) GetTimeSeriesValue(tdata value.Value, idx int) (value.Value, bool) {
	var atime int64
	var m map[string]interface{}
	if this.tsInterval > 0 {
		atime = this.tsStart + int64(idx)*this.tsInterval
		if !this.tsRanges.ValidRange(atime) {
			return value.NULL_VALUE, true
		}
		m = make(map[string]interface{}, 2)
		m["_t"] = atime
		if this.tsProjectAll {
			actval := tdata.Actual()
			if ltval, lok := actval.([]interface{}); lok {
				for i, v := range ltval {
					m["_v"+strconv.Itoa(i)] = v
				}
			} else {
				m["_v0"] = actval
			}
		} else if tdata.Type() != value.ARRAY {
			if ok, _ := this.tsProject[0]; ok {
				m["_v0"] = tdata.Actual()
			}
		} else {
			for pos, _ := range this.tsProject {
				if v, lok := tdata.Index(pos); lok {
					m["_v"+strconv.Itoa(pos)] = v.Actual()
				}
			}
		}
		return value.NewValue(m), false
	} else if tdata.Type() == value.ARRAY {
		i := 0
		if v, ok := tdata.Index(i); ok {
			if v.Type() == value.NUMBER {
				atime = value.AsNumberValue(v).Int64()
				if this.tsRanges.ValidRange(atime) {
					m = make(map[string]interface{}, 2)
					m["_t"] = atime
					if this.tsProjectAll {
						i++
						v, ok = tdata.Index(i)
						for ok {
							m["_v"+strconv.Itoa(i-1)] = v.Actual()
							i++
							v, ok = tdata.Index(i)
						}
					} else {
						for pos, _ := range this.tsProject {
							// get value from +1 becuase first one is time
							if v, lok := tdata.Index(pos + 1); lok {
								m["_v"+strconv.Itoa(pos)] = v.Actual()
							}
						}
					}
					return value.NewValue(m), false
				}
			}
		}
	}
	return value.NULL_VALUE, true
}

// Get the next data point

func (this *TimeSeriesData) GetNextValue(idx int) (value.Value, int, bool) {
	for {
		act, ok := this.tsData.Index(idx)
		if !ok || act.Type() == value.MISSING {
			return act, idx, ok
		}
		actv, skip := this.GetTimeSeriesValue(act, idx)
		idx++
		if !skip {
			return actv, idx, true
		}
	}
}

type TimeSeriesRange struct {
	start int64
	end   int64
}

type TimeSeriesRanges []*TimeSeriesRange

func NewTimeSeriesRange(start, end int64) *TimeSeriesRange {
	return &TimeSeriesRange{start, end}
}

func (this TimeSeriesRanges) All() bool {
	return len(this) == 0
}

func (this TimeSeriesRanges) ValidRange(val int64) bool {
	for _, t := range this {
		if val >= t.start && val <= t.end {
			return true
		}
	}
	return len(this) == 0
}

func (this TimeSeriesRanges) Qualified(dstart, dend int64) bool {
	for _, t := range this {
		if dstart <= t.end || dend >= t.start {
			return true
		}
	}
	return len(this) == 0

}

func getInt64(v value.Value) (int64, bool) {
	if v.Type() == value.NUMBER {
		return value.IsIntValue(v)
	}
	return 0, false
}
