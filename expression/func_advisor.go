//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"github.com/couchbase/query/tenant"
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
		_ACTIVE:    " AND state IN [\"running\", \"scheduled\"]",
		_COMPLETED: " AND state = \"completed\"",
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
	if context == nil {
		return nil, errors.NewNilEvaluateParamError("context")
	}
	arg, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	} else if arg.Type() == value.MISSING {
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
		vali, ok := actual["action"]
		if ok {
			newContext := context
			if tenant.IsServerless() && !context.IsAdmin() {
				ctx, err := context.AdminContext()
				if err != nil {
					return nil, err
				}
				newContext = ctx.(Context)
			}
			val := strings.ToLower(value.NewValue(vali).ToString())
			if val == "start" {
				sessionName, err := util.UUIDV4()
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
				err = this.scheduleTask(sessionName, duration, newContext, settings_stop, analyzeWorkload(profile, response_limit,
					duration.Seconds(), query_count, context))
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

				return getResults(sessionName, context, newContext)
			} else if val == "purge" || val == "abort" {
				sessionName, err := this.getSession(actual)
				if err != nil {
					return nil, err
				}

				return purgeResults(sessionName, context, newContext, false)
			} else if val == "list" {
				status, err := this.getStatus(actual)
				if err != nil {
					return nil, err
				}

				return listSessions(status, context, newContext)
			} else if val == "stop" {
				sessionName, err := this.getSession(actual)
				if err != nil {
					return nil, err
				}
				state, err := getState(sessionName, context, newContext)
				if err != nil {
					return nil, err
				}
				if len(state) == 0 {
					return nil, errors.NewAdvisorSessionNotFoundError(sessionName)
				} else if state != scheduler.RUNNING && state != scheduler.SCHEDULED {
					return value.EMPTY_ARRAY_VALUE, nil
				}

				return purgeResults(sessionName, context, newContext, true)
			} else {
				return nil, errors.NewAdvisorActionNotValid(val)
			}
		} else {
			return nil, errors.NewAdvisorActionMissing()
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
			nil, nil, false, true, false, "")

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

func (this *Advisor) Indexable() bool {
	return false
}

func (this *Advisor) Privileges() *auth.Privileges {
	// For session management user must have priviledge to
	// access system keyspaces.
	// Priviledges for underlying statements are checked at
	// the time of the ADVISE statement
	// For serverless we allow users to advise on their own bucket
	// without any restrictions
	privs := auth.NewPrivileges()
	if !tenant.IsServerless() && this.isSession() {
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

func (this *Advisor) scheduleTask(sessionName string, duration time.Duration, context Context, settings map[string]interface{},
	query string) error {

	return scheduler.ScheduleTask(sessionName, _CLASS, _ANALYZE, duration,
		func(context scheduler.Context, parms interface{}) (interface{}, []errors.Error) {
			// stop monitoring
			distributed.RemoteAccess().Settings(settings)
			// collect completed requests
			res, _, err := context.EvaluateStatement(query, nil, nil, false, true, false, "")
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
			res, _, err := context.EvaluateStatement(query, nil, nil, false, true, false, "")
			if err != nil {
				return nil, []errors.Error{errors.NewError(err, "")}
			}
			return res, nil
		},

		nil, "", context)
}

func queryContext(context Context) string {
	elems := context.QueryContextParts()

	// this can't happen with serverless, but for correctness
	if len(elems) < 2 {
		return ""
	}
	if elems[0] == "" {
		elems[0] = "default"
	}
	queryContext := elems[0] + ":" + elems[1]
	return " AND (queryContext = \"" + queryContext + "\" OR queryContext LIKE \"" + queryContext + ".%\")"
}

func analyzeWorkload(profile, response_limit string, delta, query_count float64, context Context) string {
	start_time := time.Now().Format(DEFAULT_FORMAT)
	workload := "SELECT statement, queryContext AS query_context FROM system:completed_requests" +
		" WHERE statementType IN ['SELECT','UPSERT','UPDATE','INSERT','DELETE','MERGE']" +
		" AND preparedName IS NOT VALUED"

	if tenant.IsServerless() && !context.IsAdmin() {
		workload += queryContext(context)
	}
	if len(profile) > 0 {
		workload += " AND users LIKE \"%" + profile + "%\""
	}
	if response_limit != "" {
		workload += " AND str_to_duration(elapsedTime)/1000000 > " + response_limit
	}

	// exclude SELECT ADVISOR(...) statements
	workload += " AND phaseOperators.advisor IS MISSING"

	// exclude internal statements from UI
	workload += " AND (clientContextID IS MISSING OR clientContextID NOT LIKE \"INTERNAL%\")"

	workload += " AND requestTime BETWEEN \"" + start_time + "\" AND date_add_str(\"" + start_time + "\", " +
		strconv.FormatFloat(delta, 'f', 0, 64) + ",\"second\") "
	workload += " ORDER BY requestTime LIMIT " + strconv.FormatFloat(query_count, 'f', 0, 64)
	return "SELECT RAW Advisor((" + workload + "))"
}

func getResults(sessionName string, context Context, newContext Context) (value.Value, error) {
	query := "SELECT RAW results FROM system:tasks_cache WHERE class = \"" + _CLASS + "\" AND name = \"" +
		sessionName + "\" AND ANY v IN results SATISFIES v <> {} END"
	if tenant.IsServerless() && !context.IsAdmin() {
		query += queryContext(context)
	}
	r, _, err := newContext.(Context).EvaluateStatement(query, nil, nil, false, true, false, "")
	if err != nil {
		return nil, err
	}
	return value.NewValue(r), nil
}

const _EMPTY_STATE scheduler.State = ""

func getState(sessionName string, context Context, newContext Context) (scheduler.State, error) {
	query := "SELECT state FROM system:tasks_cache WHERE class = \"" + _CLASS + "\" AND name = \"" + sessionName + "\""
	if tenant.IsServerless() && !context.IsAdmin() {
		query += queryContext(context)
	}
	res, _, err := newContext.(Context).EvaluateStatement(query, nil, nil, false, false, false, "")
	if err != nil {
		return _EMPTY_STATE, err
	}
	v, ok := res.Index(0)
	if !ok {
		return _EMPTY_STATE, nil
	}
	val, ok := v.Field("state")
	if !ok {
		return _EMPTY_STATE, errors.NewAdviseInvalidResultsError()
	}
	return scheduler.State(val.ToString()), nil
}

func purgeResults(sessionName string, context Context, newContext Context, analysis bool) (value.Value, error) {
	query := "DELETE FROM system:tasks_cache WHERE class = \"" + _CLASS + "\" AND name = \"" + sessionName + "\""
	if tenant.IsServerless() && !context.IsAdmin() {
		query += queryContext(context)
	}
	_, _, err := newContext.(Context).EvaluateStatement(query, nil, nil, false, false, false, "")
	if !analysis {
		//For purge and abort, scheduler.stop func will run upon deletion when task is not nil.
		//Need to run deleting for another time to reset scheduler.stop to nil and delete the entry.
		if err == nil {
			_, _, err = context.(Context).EvaluateStatement(query, nil, nil, false, false, false, "")
		}
	}
	if err != nil {
		return nil, err
	}
	return value.EMPTY_ARRAY_VALUE, nil
}

func listSessions(status string, context Context, newContext Context) (value.Value, error) {
	query := "SELECT * FROM system:tasks_cache WHERE class = \"" + _CLASS + "\"" + queryDict()(status)
	if tenant.IsServerless() && !context.IsAdmin() {
		query += queryContext(context)
	}
	r, _, err := newContext.(Context).EvaluateStatement(query, nil, nil, false, true, false, "")
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
		return errors.NewAdvisorInvalidArgs(invalidNames)
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

func (this *Advisor) getSettingInfo(m map[string]interface{}) (profile, response_limit string, duration time.Duration,
	query_count float64, err error) {

	val1, ok := m["profile"]
	if ok {
		if val1 == nil || val1.(value.Value).Type() != value.STRING {
			err = fmt.Errorf("%s() not valid argument for 'profile'", this.Name())
			return
		} else {
			profile = val1.(value.Value).ToString()
		}
	}

	//duration is mandatory
	val2, ok := m["duration"]
	if !ok || (ok && (val2 == nil || val2.(value.Value).Type() != value.STRING)) {
		err = fmt.Errorf("%s() not valid argument for 'duration'", this.Name())
		return
	}

	duration, err = time.ParseDuration(val2.(value.Value).ToString())
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
			r, err = time.ParseDuration(val3.(value.Value).ToString())
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

	return strings.ToLower(val.(value.Value).ToString()), nil
}

func (this *Advisor) getStatus(m map[string]interface{}) (string, error) {
	val, ok := m["status"]
	if !ok {
		return _ALL, nil
	}
	if ok && val.(value.Value).Type() == value.STRING {
		status := strings.ToLower(val.(value.Value).ToString())
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
	// Consistent with parser
	if strings.HasPrefix(s, "advise") ||
		strings.HasPrefix(s, "explain") ||
		strings.HasPrefix(s, "prepare") ||
		strings.HasPrefix(s, "execute") ||
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
					str := stmt.(value.Value).ToString()
					if validateStmt(str) {
						key := str
						qc := ""
						if s, ok := m["query_context"]; ok {
							qc = s.(value.Value).ToString()
							if qc != "" {
								key += "_" + qc
							}
						}
						if s, ok := m["queryContext"]; ok {
							qc2 := s.(value.Value).ToString()
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
			ae := NewAdviseError(fmt.Errorf("%s", e.Error), q)
			this.errors = append(this.errors, ae)
		}
	}

	return nil
}
