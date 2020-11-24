//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	_CLASS         = "advisor"
	_ANALYZE       = "analyze"
	_ACTIVE        = "active"
	_COMPLETED     = "completed"
	_ALL           = "all"
	_MAXCOUNT      = 20000
	_MAXCNTPERNODE = 4000
)

func queryDict() func(string) string {
	innerMap := map[string]string{
		_ACTIVE:    " and state in [\"running\", \"scheduled\"]",
		_COMPLETED: " and state = \"completed\"",
		_ALL:       "",
	}

	return func(key string) string {
		return innerMap[key]
	}
}

type Advisor struct {
	UnaryFunctionBase
}

func NewAdvisor(operand Expression) Function {
	rv := &Advisor{
		*NewUnaryFunctionBase("advisor", operand),
	}

	rv.expr = rv
	rv.setVolatile()
	return rv
}

func (this *Advisor) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Advisor) Type() value.Type {
	return value.OBJECT
}

func (this *Advisor) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.UnaryEval(this, item, context)
}

func (this *Advisor) Indexable() bool {
	return false
}

func (this *Advisor) Privileges() *auth.Privileges {
	// For session management user must have priviledge to
	// access system keyspaces.
	// Priledges for underlying statements are checked at
	// the time of the ADVISE statement
	privs := auth.NewPrivileges()
	if this.isSession() {
		privs.Add("", auth.PRIV_SYSTEM_READ, auth.PRIV_PROPS_NONE)
	}
	return privs
}

func (this *Advisor) isSession() bool {
	arg := this.operands[0].Value()
	if arg != nil && arg.Type() == value.OBJECT {
		actual := arg.Actual().(map[string]interface{})
		if _, ok := actual["action"]; ok {
			return true
		}
	}
	return false
}

func (this *Advisor) Apply(context Context, arg value.Value) (value.Value, error) {
	if arg.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	}

	context.SetAdvisor()

	/*
		Advisor({ “action” : “start”,
		  “profile”: ”john”,
		  “response” : “3s”,
		  “duration” : “10m” ,
		  “query_count” : 20000})
	*/
	if arg.Type() == value.OBJECT {
		err := validateSessionArgs(arg)
		if err != nil {
			return nil, err
		}
		actual := arg.Actual().(map[string]interface{})
		val, ok := actual["action"]
		if ok {
			val = strings.ToLower(value.NewValue(val).Actual().(string))
			if val == "start" {
				sessionName, err := util.UUIDV3()
				if err != nil {
					return nil, err
				}

				profile, response_limit, duration, query_count, err := this.getSettingInfo(actual)
				if err != nil {
					return nil, err
				}

				numOfQueryNodes := len(distributed.RemoteAccess().GetNodeNames())
				settings_start := getSettings(profile, sessionName, response_limit, query_count, numOfQueryNodes, true)
				distributed.RemoteAccess().Settings(settings_start)

				settings_stop := getSettings(profile, sessionName, response_limit, query_count, numOfQueryNodes, false)
				err = this.scheduleTask(sessionName, duration, context, settings_stop, analyzeWorkload(profile, response_limit, duration.Seconds(), query_count))
				if err != nil {
					return nil, err
				}

				m := make(map[string]interface{}, 1)
				m["session"] = sessionName
				return value.NewValue(m), nil
			} else if val == "get" {
				sessionName, err := this.getSession(actual)
				if err != nil {
					return nil, err
				}

				return getResults(sessionName, context)
			} else if val == "purge" || val == "abort" {
				sessionName, err := this.getSession(actual)
				if err != nil {
					return nil, err
				}

				return purgeResults(sessionName, context, false)
			} else if val == "list" {
				status, err := this.getStatus(actual)
				if err != nil {
					return nil, err
				}

				return listSessions(status, context)
			} else if val == "stop" {
				sessionName, err := this.getSession(actual)
				if err != nil {
					return nil, err
				}

				return purgeResults(sessionName, context, true)
			} else {
				return nil, fmt.Errorf("%s() not valid argument for 'action'", this.Name())
			}
		} else {
			return nil, fmt.Errorf("%s() missing argument for 'action'", this.Name())
		}
	}

	//{
	//	"query_context": "default:bucket1.myscope",
	//	"statement": "create index idx1 on mycollection(name, type);"
	//}

	stmtMap, err := this.extractStrs(arg)
	if len(stmtMap) == 0 || err != nil {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	curMap := make(map[string]*mapEntry, len(stmtMap))
	recIdxMap := make(map[string]*mapEntry, len(stmtMap))
	recCidxMap := make(map[string]*mapEntry, len(stmtMap))
	errs := make([]*adviseError, 0, len(stmtMap))
	for _, v := range stmtMap {
		newContext := context.(Context)
		if v.queryContext != "" {
			newContext = newContext.NewQueryContext(v.queryContext, context.Readonly()).(Context)
		}
		r, _, err := newContext.EvaluateStatement(addPrefix(v.stmt, "advise "),
			nil, nil, false, true)

		if err != nil {
			errs = append(errs, NewAdviseError(err, v))
		} else {
			this.processResult(v, r, curMap, recIdxMap, recCidxMap)
		}
	}

	r := NewReport(curMap, recIdxMap, recCidxMap, errs)
	bytes, err := r.MarshalJSON()
	if err != nil {
		return value.EMPTY_ARRAY_VALUE, nil
	}

	return value.NewValue(bytes), nil
}

func (this *Advisor) scheduleTask(sessionName string, duration time.Duration, context Context, settings map[string]interface{}, query string) error {
	return scheduler.ScheduleTask(sessionName, _CLASS, _ANALYZE, duration,
		func(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {
			// stop monitoring
			distributed.RemoteAccess().Settings(settings)
			// collect completed requests
			res, _, err := context.EvaluateStatement(query, nil, nil, false, true)
			if err != nil {
				return nil, []errors.Error{errors.NewError(err, "")}
			}
			return res, nil
		},

		// stop task
		func(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {
			// stop monitoring
			distributed.RemoteAccess().Settings(settings)
			// collect completed requests afterwards
			res, _, err := context.EvaluateStatement(query, nil, nil, false, true)
			if err != nil {
				return nil, []errors.Error{errors.NewError(err, "")}
			}
			return res, nil
		}, nil, context)
}

func analyzeWorkload(profile, response_limit string, delta, query_count float64) string {
	start_time := time.Now().Format(DEFAULT_FORMAT)
	workload := "SELECT statement, queryContext as query_context from system:completed_requests where "
	if len(profile) > 0 {
		workload += "users like \"%" + profile + "%\" and "
	}
	if response_limit != "" {
		workload += "str_to_duration(elapsedTime)/1000000 > " + response_limit + " and "
	}

	workload += "phaseOperators.advisor is missing and "
	workload += "requestTime between \"" + start_time + "\" and DATE_ADD_STR(\"" + start_time + "\", " + strconv.FormatFloat(delta, 'f', 0, 64) + ",\"second\") "
	workload += "order by requestTime limit " + strconv.FormatFloat(query_count, 'f', 0, 64)
	return "SELECT RAW Advisor((" + workload + "))"
}

func getResults(sessionName string, context Context) (value.Value, error) {
	query := "select raw results from system:tasks_cache where class = \"" + _CLASS + "\"  and name = \"" + sessionName + "\" and ANY v in results satisfies v <> {} END"
	r, _, err := context.(Context).EvaluateStatement(query, nil, nil, false, true)
	if err != nil {
		return nil, err
	}
	return value.NewValue(r), nil
}

func purgeResults(sessionName string, context Context, analysis bool) (value.Value, error) {
	query := "DELETE from system:tasks_cache where class = \"" + _CLASS + "\"  and name = \"" + sessionName + "\""
	_, _, err := context.(Context).EvaluateStatement(query, nil, nil, false, false)
	if !analysis {
		//For purge and abort, scheduler.stop func will run upon deletion when task is not nil.
		//Need to run deleting for another time to reset scheduler.stop to nil and delete the entry.
		if err == nil {
			_, _, err = context.(Context).EvaluateStatement(query, nil, nil, false, false)
		}
	}
	if err != nil {
		return nil, err
	}
	return value.EMPTY_ARRAY_VALUE, nil
}

func listSessions(status string, context Context) (value.Value, error) {
	query := "select * from system:tasks_cache where class = \"" + _CLASS + "\"" + queryDict()(status)
	r, _, err := context.(Context).EvaluateStatement(query, nil, nil, false, true)
	if err != nil {
		return nil, err
	}
	return value.NewValue(r), nil
}

func validateSessionArgs(arg value.Value) error {
	if arg == nil {
		return nil
	}

	invalidNames := make([]string, 0, 8)
	for fieldName, _ := range arg.Fields() {
		if !strings.EqualFold(fieldName, "action") &&
			!strings.EqualFold(fieldName, "profile") &&
			!strings.EqualFold(fieldName, "response") &&
			!strings.EqualFold(fieldName, "duration") &&
			!strings.EqualFold(fieldName, "query_count") &&
			!strings.EqualFold(fieldName, "status") &&
			!strings.EqualFold(fieldName, "session") {
			invalidNames = append(invalidNames, fieldName)
		}
	}

	if len(invalidNames) > 0 {
		return fmt.Errorf("Invalid arguments to Advisor() function: %v", invalidNames)
	}

	return nil
}

func getSettings(profile, tag, response_limit string, query_count float64, numberOfNodes int, start bool) map[string]interface{} {
	n, _ := strconv.ParseInt(response_limit, 10, 64)
	settings := make(map[string]interface{}, 2)
	tagMap := make(map[string]interface{}, 3)
	if numberOfNodes == 0 {
		numberOfNodes = 1
	}
	countPerNode := int64(query_count / float64(numberOfNodes))
	if countPerNode > _MAXCNTPERNODE {
		countPerNode = _MAXCNTPERNODE
	}
	if start {
		if len(profile) > 0 {
			tagMap["user"] = profile
		}
		if n > 0 || (len(profile) == 0 && n == 0) {
			tagMap["threshold"] = n
		}

		tagMap["tag"] = tag
		settings["+completed-limit"] = countPerNode
	} else {
		if len(profile) > 0 {
			tagMap["-user"] = profile
		}
		if n > 0 || (len(profile) == 0 && n == 0) {
			tagMap["-threshold"] = n
		}

		tagMap["tag"] = tag
		settings["-completed-limit"] = countPerNode
	}

	settings["completed"] = tagMap
	settings["distribute"] = true
	return settings
}

func (this *Advisor) getSettingInfo(m map[string]interface{}) (profile, response_limit string, duration time.Duration, query_count float64, err error) {
	val1, ok := m["profile"]
	if ok {
		if val1 == nil || val1.(value.Value).Type() != value.STRING {
			err = fmt.Errorf("%s() not valid argument for 'profile'", this.Name())
			return
		} else {
			profile = val1.(value.Value).Actual().(string)
		}
	}

	//duration is mandatory
	val2, ok := m["duration"]
	if !ok || (ok && (val2 == nil || val2.(value.Value).Type() != value.STRING)) {
		err = fmt.Errorf("%s() not valid argument for 'duration'", this.Name())
		return
	}

	duration, err = time.ParseDuration(val2.(value.Value).Actual().(string))
	if err != nil {
		return
	}

	val3, ok := m["response"]
	if ok {
		if val3 == nil || val3.(value.Value).Type() != value.STRING {
			err = fmt.Errorf("%s() not valid argument for 'response'", this.Name())
			return
		} else {
			var r time.Duration
			r, err = time.ParseDuration(val3.(value.Value).Actual().(string))
			if err != nil {
				return
			}
			//threshold is in millisecond
			response_limit = strconv.FormatFloat(r.Seconds()*1000, 'f', 0, 64)
		}
	}

	val4, ok := m["query_count"]
	if ok {
		if val4 == nil || val4.(value.Value).Type() != value.NUMBER {
			err = fmt.Errorf("%s() not valid argument for 'query_count'", this.Name())
			return
		} else {
			query_count = val4.(value.Value).Actual().(float64)
		}
	} else {
		query_count = _MAXCOUNT
	}

	return
}

func (this *Advisor) getSession(m map[string]interface{}) (string, error) {
	val, ok := m["session"]
	if !ok || (ok && (val == nil || val.(value.Value).Type() != value.STRING)) {
		return "", fmt.Errorf("%s() not valid argument for 'session'", this.Name())
	}

	return strings.ToLower(val.(value.Value).Actual().(string)), nil
}

func (this *Advisor) getStatus(m map[string]interface{}) (string, error) {
	val, ok := m["status"]
	if !ok {
		return _ALL, nil
	}
	if ok && val.(value.Value).Type() == value.STRING {
		status := strings.ToLower(val.(value.Value).Actual().(string))
		if status == _ACTIVE || status == _COMPLETED || status == _ALL {
			return status, nil
		}
	}
	return "", fmt.Errorf("%s() not valid argument for 'status'", this.Name())
}

func (this *Advisor) processResult(obj *queryObject, res value.Value, curMap, recIdxMap, recCidxMap map[string]*mapEntry) {
	val := res.Actual().([]interface{})
	for _, v := range val {
		v := value.NewValue(v).Actual().(map[string]interface{})
		v1, ok := v["advice"]
		if !ok {
			continue
		}
		v1a := value.NewValue(v1).Actual().(map[string]interface{})
		v2, ok := v1a["adviseinfo"]
		if !ok {
			continue
		}
		v2a, ok := value.NewValue(v2).Actual().(map[string]interface{})
		if !ok {
			continue
		}
		for k3, v3 := range v2a {
			if k3 == "recommended_indexes" {
				v4 := value.NewValue(v3).Actual()
				v4a, ok := v4.(map[string]interface{})
				if !ok {
					continue
				}
				for k5, v5 := range v4a {
					if k5 == "indexes" {
						addToMap(recIdxMap, v5, obj)
					} else if k5 == "covering_indexes" {
						addToMap(recCidxMap, v5, obj)
					}
				}
			} else if k3 == "current_indexes" {
				addToMap(curMap, v3, obj)
			}
		}
	}
}

func addToMap(m map[string]*mapEntry, v interface{}, obj *queryObject) {
	v1 := value.NewValue(v).Actual().([]interface{})
	for _, v2 := range v1 {
		v2 := value.NewValue(v2).Actual().(map[string]interface{})
		if v3, ok := v2["index_statement"]; ok {
			_, ok := m[v3.(string)]
			if !ok {
				m[v3.(string)] = NewMapEntry()
			}
			m[v3.(string)].addToQueryList(obj)
			if v4, ok := v2["update_statistics"]; ok {
				m[v3.(string)].setUpdStats(v4.(string))
			}
		}
	}
}

func addPrefix(s, prefix string) string {
	s = strings.TrimSpace(s)
	if len(s) < len(prefix) || !strings.HasPrefix(strings.ToLower(s[0:len(prefix)]), prefix) {
		s = prefix + s
	}
	return s
}

func validateStmt(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(s, "advise") ||
		strings.Contains(s, "system:") {
		return false
	}
	return true
}

func (this *Advisor) extractStrs(arg value.Value) (map[string]*queryObject, error) {
	entryMap := make(map[string]*queryObject, 1)
	actuals := arg.Actual()

	switch actuals := actuals.(type) {
	case []interface{}:
		for _, key := range actuals {
			k := value.NewValue(key).Actual()
			if str, ok := k.(string); ok {
				if validateStmt(str) {
					if _, ok := entryMap[str]; ok {
						entryMap[str].addOne()
					} else {
						entryMap[str] = NewQueryObject(str, "", 1)

					}
				}
			} else if m, ok := k.(map[string]interface{}); ok {
				if stmt, ok := m["statement"]; ok {
					str := stmt.(value.Value).Actual().(string)
					if validateStmt(str) {
						key := str
						qc := ""
						if s, ok := m["query_context"]; ok {
							qc = s.(value.Value).Actual().(string)
							if qc != "" {
								key += "_" + qc
							}
						}
						if s, ok := m["queryContext"]; ok {
							qc2 := s.(value.Value).Actual().(string)
							if qc != "" && qc2 != "" && qc != qc2 {
								return nil, fmt.Errorf("Two different query_context have been set: %s and %s.", qc, qc2)
							}
							if qc == "" && qc2 != "" {
								key += "_" + qc2
								qc = qc2
							}
						}

						if _, ok := entryMap[key]; ok {
							entryMap[key].addOne()
						} else {
							entryMap[key] = NewQueryObject(str, qc, 1)
						}
					}
				}
			}
		}
	case string:
		if validateStmt(actuals) {
			entryMap[actuals] = NewQueryObject(actuals, "", 1)
		}
	default:
		return nil, fmt.Errorf("No proper formatted input.")
	}

	return entryMap, nil
}

func (this *Advisor) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewAdvisor(operands[0])
	}
}

type queryObject struct {
	stmt         string
	cnt          int
	queryContext string
}

func NewQueryObject(s, qc string, n int) *queryObject {
	return &queryObject{
		stmt:         s,
		queryContext: qc,
		cnt:          n,
	}
}

func (this *queryObject) addOne() {
	this.cnt += 1
}

func (this *queryObject) setQueryContext(qc string) {
	if this.queryContext == "" {
		this.queryContext = qc
	}
}

func (this *queryObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *queryObject) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"statement": this.stmt,
		"run_count": this.cnt,
	}

	if this.queryContext != "" {
		r["query_context"] = this.queryContext
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *queryObject) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Stmt         string `json:"statement"`
		Times        int    `json:"run_count"`
		QueryContext string `json:"query_context"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.stmt = _unmarshalled.Stmt
	this.cnt = _unmarshalled.Times
	if _unmarshalled.QueryContext != "" {
		this.queryContext = _unmarshalled.QueryContext
	}

	return nil
}

type queryList []*queryObject

func NewQueryList() queryList {
	return make(queryList, 0, 1)
}

type mapEntry struct {
	queryList queryList
	upd_stats string
}

func NewMapEntry() *mapEntry {
	return &mapEntry{
		queryList: make(queryList, 0, 1),
	}
}

func (this *mapEntry) setUpdStats(s string) {
	if s == "" {
		return
	}

	/*For one covering index, there may be more than one kind of update_statistics.
	  In this situation, save the longest upd_stats to make sure all its queries can use CBO.
	*/
	if this.upd_stats == "" {
		this.upd_stats = s
	} else if len(strings.Split(s, ",")) > len(strings.Split(this.upd_stats, ",")) {
		this.upd_stats = s
	}
}

func (this *mapEntry) addToQueryList(obj *queryObject) {
	this.queryList = append(this.queryList, obj)
}

type indexMap struct {
	index    string
	updStats string
	queries  queryList
}

func NewIndexMap(idx, upd_stats string, ql queryList) *indexMap {
	sort.Slice(ql, func(i, j int) bool {
		return ql[i].cnt > ql[j].cnt
	})

	return &indexMap{
		index:    idx,
		updStats: upd_stats,
		queries:  ql,
	}
}

func (this *indexMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *indexMap) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{
		"index":      this.index,
		"statements": this.queries,
	}

	if this.updStats != "" {
		r["update_statistics"] = this.updStats
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *indexMap) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Index    string            `json:"index"`
		UpdStats string            `json:"update_statistics"`
		Queries  []json.RawMessage `json:"statements"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.index = _unmarshalled.Index
	if _unmarshalled.UpdStats != "" {
		this.updStats = _unmarshalled.UpdStats
	}

	ql := NewQueryList()
	for _, v1 := range _unmarshalled.Queries {
		r := &queryObject{}
		err = r.UnmarshalJSON(v1)
		if err != nil {
			return err
		}
		ql = append(ql, r)
	}
	this.queries = ql

	return nil
}

type indexMaps []*indexMap

func NewIndexMaps(m map[string]*mapEntry) indexMaps {
	ims := make([]*indexMap, 0, len(m))
	for k, v := range m {
		ims = append(ims, NewIndexMap(k, v.upd_stats, v.queryList))
	}
	return ims
}

type adviseError struct {
	err   error
	query *queryObject
}

func NewAdviseError(err error, query *queryObject) *adviseError {
	return &adviseError{
		err:   err,
		query: query,
	}
}

type report struct {
	currentIdxs    indexMaps
	recommendIdxs  indexMaps
	recommendCidxs indexMaps
	errors         []*adviseError
}

func NewReport(current, recIdxs, redCidxs map[string]*mapEntry, errors []*adviseError) *report {
	return &report{
		currentIdxs:    NewIndexMaps(current),
		recommendIdxs:  NewIndexMaps(recIdxs),
		recommendCidxs: NewIndexMaps(redCidxs),
		errors:         errors,
	}
}

func (this *report) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *report) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{}

	if len(this.currentIdxs) > 0 {
		r["current_used_indexes"] = this.currentIdxs
	}

	if len(this.recommendIdxs) > 0 {
		r["recommended_indexes"] = this.recommendIdxs
	}

	if len(this.recommendCidxs) > 0 {
		r["recommended_covering_indexes"] = this.recommendCidxs
	}

	if len(this.errors) > 0 {
		errs := make([]interface{}, 0, len(this.errors))
		for _, e := range this.errors {
			if e == nil {
				continue
			}
			ae := make(map[string]interface{}, 4)
			ae["error"] = e.err.Error()
			ae["statement"] = e.query.stmt
			ae["run_count"] = e.query.cnt
			if e.query.queryContext != "" {
				ae["query_context"] = e.query.queryContext
			}
			errs = append(errs, ae)
		}
		if len(errs) > 0 {
			r["errors"] = errs
		}
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *report) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		CurIdxes []json.RawMessage `json:"current_used_indexes"`
		RecIdexs []json.RawMessage `json:"recommended_indexes"`
		RecCidxs []json.RawMessage `json:"recommended_covering_indexes"`
		Errors   []struct {
			Error   string `json:"error"`
			Stmt    string `json:"statement"`
			Cnt     int    `json:"run_count"`
			Context string `json:"query_context"`
		} `json:"errors"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if len(_unmarshalled.CurIdxes) > 0 {
		this.currentIdxs = make(indexMaps, 0, len(_unmarshalled.CurIdxes))
		for _, v := range _unmarshalled.CurIdxes {
			r := &indexMap{}
			err = r.UnmarshalJSON(v)
			this.currentIdxs = append(this.currentIdxs, r)
		}
	}

	if len(_unmarshalled.RecIdexs) > 0 {
		this.recommendIdxs = make(indexMaps, 0, len(_unmarshalled.RecIdexs))
		for _, v := range _unmarshalled.RecIdexs {
			r := &indexMap{}
			err = r.UnmarshalJSON(v)
			this.recommendIdxs = append(this.recommendIdxs, r)
		}
	}

	if len(_unmarshalled.RecCidxs) > 0 {
		this.recommendCidxs = make(indexMaps, 0, len(_unmarshalled.RecCidxs))
		for _, v := range _unmarshalled.RecCidxs {
			r := &indexMap{}
			err = r.UnmarshalJSON(v)
			this.recommendCidxs = append(this.recommendCidxs, r)
		}
	}

	if len(_unmarshalled.Errors) > 0 {
		this.errors = make([]*adviseError, 0, len(_unmarshalled.Errors))
		for _, e := range _unmarshalled.Errors {
			q := NewQueryObject(e.Stmt, e.Context, e.Cnt)
			ae := NewAdviseError(fmt.Errorf(e.Error), q)
			this.errors = append(this.errors, ae)
		}
	}

	return nil
}
