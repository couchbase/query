//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
	"github.com/gorilla/mux"
)

const (
	accountingPrefix = adminPrefix + "/stats"
	vitalsPrefix     = adminPrefix + "/vitals"
	preparedsPrefix  = adminPrefix + "/prepareds"
	requestsPrefix   = adminPrefix + "/active_requests"
	completedsPrefix = adminPrefix + "/completed_requests"
	functionsPrefix  = adminPrefix + "/functions_cache"
	indexesPrefix    = adminPrefix + "/indexes"
	expvarsRoute     = "/debug/vars"
)

func expvarsHandler(w http.ResponseWriter, req *http.Request) {
	// Do not audit directly.
	// Will be handled and audited by /admin/stats auditing.
	http.Redirect(w, req, accountingPrefix, http.StatusFound)
}

func (this *HttpEndpoint) registerAccountingHandlers() {
	statsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStats)
	}
	statHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStat)
	}
	notFoundHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNotFound)
	}
	vitalsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doVitals)
	}
	preparedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepared)
	}
	preparedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepareds)
	}
	requestsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequests)
	}
	requestHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequest)
	}
	completedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequests)
	}
	completedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequest)
	}
	preparedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPreparedIndex)
	}
	requestIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doRequestIndex)
	}
	completedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedIndex)
	}
	functionIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctionIndex)
	}
	functionHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunction)
	}
	functionsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctions)
	}
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:                      {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}":          {handler: statHandler, methods: []string{"GET", "DELETE"}},
		vitalsPrefix:                          {handler: vitalsHandler, methods: []string{"GET"}},
		preparedsPrefix:                       {handler: preparedsHandler, methods: []string{"GET"}},
		preparedsPrefix + "/{name}":           {handler: preparedHandler, methods: []string{"GET", "POST", "DELETE", "PUT"}},
		requestsPrefix:                        {handler: requestsHandler, methods: []string{"GET"}},
		requestsPrefix + "/{request}":         {handler: requestHandler, methods: []string{"GET", "POST", "DELETE"}},
		completedsPrefix:                      {handler: completedsHandler, methods: []string{"GET"}},
		completedsPrefix + "/{request}":       {handler: completedHandler, methods: []string{"GET", "POST", "DELETE"}},
		functionsPrefix:                       {handler: functionsHandler, methods: []string{"GET"}},
		functionsPrefix + "/{name}":           {handler: functionHandler, methods: []string{"GET", "POST", "DELETE"}},
		indexesPrefix + "/prepareds":          {handler: preparedIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/active_requests":    {handler: requestIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/completed_requests": {handler: completedIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/function_cache":     {handler: functionIndexHandler, methods: []string{"GET"}},
	}

	for route, h := range routeMap {
		this.mux.HandleFunc(route, h.handler).Methods(h.methods...)
	}

	this.mux.HandleFunc(expvarsRoute, expvarsHandler).Methods("GET")

	this.mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}

func doStats(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

	af.EventTypeId = audit.API_ADMIN_STATS

	switch req.Method {
	case "GET":
		stats := make(map[string]interface{})
		for name, metric := range reg.Counters() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Gauges() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Timers() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Meters() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Histograms() {
			addMetricData(name, stats, getMetricData(metric))
		}
		return stats, nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func addMetricData(name string, stats map[string]interface{}, metrics map[string]interface{}) {
	for metric_type, metric_value := range metrics {

		// MB-20521 avoid buffers that are reused
		stats[name+"."+metric_type] = metric_value
	}
}

func doStat(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["stat"]
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

	af.EventTypeId = audit.API_ADMIN_STATS
	af.Stat = name

	switch req.Method {
	case "GET":
		metric := reg.Get(name)
		if metric != nil {
			return getMetricData(metric), nil
		} else {
			return nil, nil
		}
	case "DELETE":
		return nil, reg.Unregister(name)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doNotFound(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	accounting.UpdateCounter(accounting.INVALID_REQUESTS)
	return nil, nil
}

func doVitals(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_VITALS
	switch req.Method {
	case "GET":
		acctStore := endpoint.server.AccountingStore()
		return acctStore.Vitals()
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

// Credentials can come from two sources: the basic username/password
// from basic authorizatio, and from a "creds" value, which encodes
// in JSON an array of username/password pairs, like this:
//   [{"user":"foo", "pass":"foopass"}, {"user":"bar", "pass": "barpass"}]
func getCredentialsFromRequest(req *http.Request) (auth.Credentials, errors.Error) {
	creds := make(auth.Credentials)
	user, pass, ok := req.BasicAuth()
	if ok {
		creds[user] = pass
	}
	creds_json := req.FormValue("creds")
	if creds_json != "" {
		cred_list := make([]map[string]string, 0, 2)
		err := json.Unmarshal([]byte(creds_json), &cred_list)
		if err != nil {
			return nil, errors.NewAdminCredsError(creds_json, err)
		} else {
			for _, v := range cred_list {
				user, user_ok := v["user"]
				pass, pass_ok := v["pass"]
				if !user_ok || !pass_ok {
					return nil, errors.NewAdminCredsError(creds_json, nil)
				}
				creds[user] = pass
			}
		}
	}
	return creds, nil
}

func verifyCredentialsFromRequest(api string, req *http.Request, af *audit.ApiAuditFields) errors.Error {
	creds, err := getCredentialsFromRequest(req)
	if err != nil {
		return err
	}

	users := make([]string, 0, len(creds))
	for user := range creds {
		users = append(users, user)
	}
	af.Users = users

	privs := auth.NewPrivileges()
	privs.Add("system:"+api, auth.PRIV_SYSTEM_READ)
	_, err = datastore.GetDatastore().Authorize(privs, creds, req)
	return err
}

func doPrepared(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	af.Name = name

	if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("prepareds", req, af)
		if err != nil {
			return nil, err
		}
		err = prepareds.DeletePrepared(name)
		if err != nil {
			return nil, err
		}
		return true, nil
	} else if req.Method == "PUT" {
		body, err1 := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		// http.BasicAuth eats the body, so verify credentials after getting the body.
		err := verifyCredentialsFromRequest("prepareds", req, af)
		if err != nil {
			return nil, err
		}

		if err1 != nil {
			return nil, errors.NewAdminBodyError(err1)
		}

		prepared, _ := prepareds.GetPrepared(value.NewValue(name), 0, nil)

		// nothing to do if the prepared is there and the plan matches
		if prepared != nil && !prepared.MismatchingEncodedPlan(string(body)) {
			return "", nil
		}
		_, err = prepareds.DecodePrepared(name, string(body), false, false, nil)
		if err != nil {
			return nil, err
		}
		return "", nil
	} else if req.Method == "GET" || req.Method == "POST" {
		if req.Method == "POST" {
			// Do not audit POST requests. They are an internal API used
			// only for queries to system:prepareds, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		err := verifyCredentialsFromRequest("prepareds", req, af)
		if err != nil {
			return nil, err
		}

		var itemMap map[string]interface{}

		prepareds.PreparedDo(name, func(entry *prepareds.CacheEntry) {
			itemMap = map[string]interface{}{
				"name":            name,
				"uses":            entry.Uses,
				"statement":       entry.Prepared.Text(),
				"indexApiVersion": entry.Prepared.IndexApiVersion(),
				"featureControls": entry.Prepared.FeatureControls(),
			}
			if entry.Prepared.EncodedPlan() != "" {
				itemMap["encoded_plan"] = entry.Prepared.EncodedPlan()
			}
			if req.Method == "POST" {
				itemMap["plan"] = entry.Prepared.Operator
			}

			// only give times for entries that have completed at least one execution
			if entry.Uses > 0 && entry.RequestTime > 0 {
				itemMap["lastUse"] = entry.LastUse.String()
				itemMap["avgElapsedTime"] = (time.Duration(entry.RequestTime) /
					time.Duration(entry.Uses)).String()
				itemMap["avgServiceTime"] = (time.Duration(entry.ServiceTime) /
					time.Duration(entry.Uses)).String()
				itemMap["minElapsedTime"] = time.Duration(entry.MinRequestTime).String()
				itemMap["minServiceTime"] = time.Duration(entry.MinServiceTime).String()
				itemMap["maxElapsedTime"] = time.Duration(entry.MaxRequestTime).String()
				itemMap["maxServiceTime"] = time.Duration(entry.MaxServiceTime).String()
			}
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	switch req.Method {
	case "GET":
		err := verifyCredentialsFromRequest("prepareds", req, af)
		if err != nil {
			return nil, err
		}

		numPrepareds := prepareds.CountPrepareds()
		data := make([]map[string]interface{}, numPrepareds)
		i := 0

		snapshot := func(name string, d *prepareds.CacheEntry) bool {

			// FIXME quick hack to avoid overruns
			if i >= numPrepareds {
				return false
			}
			data[i] = map[string]interface{}{}
			data[i]["name"] = d.Prepared.Name()
			if d.Prepared.EncodedPlan() != "" {
				data[i]["encoded_plan"] = d.Prepared.EncodedPlan()
			}
			data[i]["statement"] = d.Prepared.Text()
			data[i]["uses"] = d.Uses
			if d.Uses > 0 {
				data[i]["lastUse"] = d.LastUse.String()
			}
			i++
			return true
		}

		prepareds.PreparedsForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunction(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

	af.EventTypeId = audit.API_ADMIN_FUNCTIONS
	af.Name = name

	if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("functions_cache", req, af)
		if err != nil {
			return nil, err
		}
		functions.FunctionClear(name, nil)
		return true, nil
	} else if req.Method == "GET" || req.Method == "POST" {
		if req.Method == "POST" {
			// Do not audit POST requests. They are an internal API used
			// only for queries to system:functions_cache, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		err := verifyCredentialsFromRequest("functions_cache", req, af)
		if err != nil {
			return nil, err
		}

		var itemMap map[string]interface{}

		functions.FunctionDo(name, func(entry *functions.FunctionEntry) {
			itemMap = map[string]interface{}{
				"uses": entry.Uses,
			}
			entry.Signature(itemMap)
			entry.Body(itemMap)

			// only give times for entries that have completed at least one execution
			if entry.Uses > 0 && entry.ServiceTime > 0 {
				itemMap["lastUse"] = entry.LastUse.String()
				itemMap["avgServiceTime"] = (time.Duration(entry.ServiceTime) /
					time.Duration(entry.Uses)).String()
				itemMap["minServiceTime"] = time.Duration(entry.MinServiceTime).String()
				itemMap["maxServiceTime"] = time.Duration(entry.MaxServiceTime).String()
			}
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunctions(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_FUNCTIONS
	switch req.Method {
	case "GET":
		err := verifyCredentialsFromRequest("functions_cache", req, af)
		if err != nil {
			return nil, err
		}

		numFunctions := functions.CountFunctions()
		data := make([]map[string]interface{}, numFunctions)
		i := 0

		snapshot := func(name string, d *functions.FunctionEntry) bool {

			// FIXME quick hack to avoid overruns
			if i >= numFunctions {
				return false
			}
			data[i] = map[string]interface{}{}
			data[i]["uses"] = d.Uses
			d.Signature(data[i])
			d.Body(data[i])
			if d.Uses > 0 {
				data[i]["lastUse"] = d.LastUse.String()
			}
			i++
			return true
		}

		functions.FunctionsForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	af.EventTypeId = audit.API_ADMIN_ACTIVE_REQUESTS
	af.Request = requestId

	if req.Method == "GET" || req.Method == "POST" {
		if req.Method == "POST" {
			// Do not audit POST requests. They are an internal API used
			// only for queries to system:active_requests, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		err := verifyCredentialsFromRequest("actives", req, af)
		if err != nil {
			return nil, err
		}
		reqMap := activeRequestWorkHorse(endpoint, requestId, (req.Method == "POST"))

		return reqMap, nil
	} else if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("actives", req, af)
		if err != nil {
			return nil, err
		}
		if endpoint.actives.Delete(requestId, true) {
			return nil, errors.NewServiceErrorHttpReq(requestId)
		}

		return true, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func activeRequestWorkHorse(endpoint *HttpEndpoint, requestId string, profiling bool) map[string]interface{} {
	reqMap := map[string]interface{}{}
	_ = endpoint.actives.Get(requestId, func(request server.Request) {
		reqMap["requestId"] = request.Id().String()
		cId := request.ClientID().String()
		if cId != "" {
			reqMap["clientContextID"] = cId
		}
		if request.Statement() != "" {
			reqMap["statement"] = request.Statement()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			reqMap["preparedName"] = p.Name()
			reqMap["preparedText"] = p.Text()
		}
		reqMap["requestTime"] = request.RequestTime().String()
		reqMap["elapsedTime"] = time.Since(request.RequestTime()).String()
		reqMap["executionTime"] = time.Since(request.ServiceTime()).String()
		reqMap["state"] = request.State()
		reqMap["scanConsistency"] = request.ScanConsistency()

		p := request.Output().FmtPhaseCounts()
		if p != nil {
			reqMap["phaseCounts"] = p
		}
		p = request.Output().FmtPhaseOperators()
		if p != nil {
			reqMap["phaseOperators"] = p
		}
		if profiling {
			reqMap["phaseTimes"] = request.Output().FmtPhaseTimes()

			prof := request.Profile()
			if prof == server.ProfUnset {
				prof = endpoint.server.Profile()
			}
			t := request.GetTimings()

			// TODO - check lifetime of entry
			// by the time we marshal, is this still valid?
			if prof == server.ProfOn && t != nil {
				reqMap["timings"] = t
			}

			var ctrl bool
			ctr := request.Controls()
			if ctr == value.NONE {
				ctrl = endpoint.server.Controls()
			} else {
				ctrl = (ctr == value.TRUE)
			}
			if ctrl {
				na := request.NamedArgs()
				if na != nil {
					reqMap["namedArgs"] = na
				}
				pa := request.PositionalArgs()
				if pa != nil {
					reqMap["positionalArgs"] = pa
				}
			}
		}
		credsString := datastore.CredsString(request.Credentials(), request.OriginalHttpRequest())
		if credsString != "" {
			reqMap["users"] = credsString
		}
		remoteAddr := request.RemoteAddr()
		if remoteAddr != "" {
			reqMap["remoteAddr"] = remoteAddr
		}
		userAgent := request.UserAgent()
		if userAgent != "" {
			reqMap["userAgent"] = userAgent
		}
	})
	return reqMap
}

func doActiveRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	err := verifyCredentialsFromRequest("actives", req, af)
	if err != nil {
		return nil, err
	}

	numRequests, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}

	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request server.Request) bool {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return false
		}
		requests[i] = map[string]interface{}{}
		requests[i]["requestId"] = request.Id().String()
		cId := request.ClientID().String()
		if cId != "" {
			requests[i]["clientContextID"] = cId
		}
		if request.Statement() != "" {
			requests[i]["statement"] = request.Statement()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			requests[i]["preparedName"] = p.Name()
			requests[i]["preparedStatement"] = p.Text()
		}
		requests[i]["requestTime"] = request.RequestTime().String()
		requests[i]["elapsedTime"] = time.Since(request.RequestTime()).String()
		requests[i]["executionTime"] = time.Since(request.ServiceTime()).String()
		requests[i]["state"] = request.State()
		requests[i]["scanConsistency"] = request.ScanConsistency()

		credsString := datastore.CredsString(request.Credentials(), request.OriginalHttpRequest())
		if credsString != "" {
			requests[i]["users"] = credsString
		}

		p := request.Output().FmtPhaseCounts()
		if p != nil {
			requests[i]["phaseCounts"] = p
		}
		p = request.Output().FmtPhaseOperators()
		if p != nil {
			requests[i]["phaseOperators"] = p
		}
		i++
		return true
	}
	endpoint.actives.ForEach(snapshot, nil)
	return requests, nil
}

func doCompletedRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	af.Request = requestId

	if req.Method == "GET" || req.Method == "POST" {
		if req.Method == "POST" {
			// Do not audit POST requests. They are an internal API used
			// only for queries to system:completed_requests, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		err := verifyCredentialsFromRequest("completed_requests", req, af)
		if err != nil {
			return nil, err
		}
		reqMap := completedRequestWorkHorse(requestId, (req.Method == "POST"))
		return reqMap, nil
	} else if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("completed_requests", req, af)
		if err != nil {
			return nil, err
		}
		err = server.RequestDelete(requestId)
		if err != nil {
			return nil, err
		} else {
			return true, nil
		}
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func completedRequestWorkHorse(requestId string, profiling bool) map[string]interface{} {
	reqMap := map[string]interface{}{}
	server.RequestDo(requestId, func(request *server.RequestLogEntry) {
		reqMap["requestId"] = request.RequestId
		if request.ClientId != "" {
			reqMap["clientContextID"] = request.ClientId
		}
		reqMap["state"] = request.State
		reqMap["scanConsistency"] = request.ScanConsistency
		if request.Statement != "" {
			reqMap["statement"] = request.Statement
		}
		if request.PreparedName != "" {
			reqMap["preparedName"] = request.PreparedName
			reqMap["preparedText"] = request.PreparedText
		}
		reqMap["requestTime"] = request.Time.String()
		reqMap["elapsedTime"] = request.ElapsedTime.String()
		reqMap["serviceTime"] = request.ServiceTime.String()
		reqMap["resultCount"] = request.ResultCount
		reqMap["resultSize"] = request.ResultSize
		reqMap["errorCount"] = request.ErrorCount
		if request.PhaseCounts != nil {
			reqMap["phaseCounts"] = request.PhaseCounts
		}
		if request.PhaseOperators != nil {
			reqMap["phaseOperators"] = request.PhaseOperators
		}
		if profiling {
			if request.PhaseTimes != nil {
				reqMap["phaseTimes"] = request.PhaseTimes
			}
			if request.NamedArgs != nil {
				reqMap["namedArgs"] = request.NamedArgs
			}
			if request.PositionalArgs != nil {
				reqMap["positionalArgs"] = request.PositionalArgs
			}
			if request.Timings != nil {
				reqMap["timings"] = request.Timings
			}
			if request.Errors != nil {
				errors := make([]map[string]interface{}, len(request.Errors))
				for i, e := range request.Errors {
					errors[i] = e.Object()
				}
				reqMap["errors"] = errors
			}
		}
		if request.Users != "" {
			reqMap["users"] = request.Users
		}
		if request.RemoteAddr != "" {
			reqMap["remoteAddr"] = request.RemoteAddr
		}
		if request.UserAgent != "" {
			reqMap["userAgent"] = request.UserAgent
		}
	})
	return reqMap
}

func doCompletedRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	err := verifyCredentialsFromRequest("completed_requests", req, af)
	if err != nil {
		return nil, err
	}

	numRequests := server.RequestsCount()
	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request *server.RequestLogEntry) bool {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return false
		}
		requests[i] = map[string]interface{}{}
		requests[i]["requestId"] = request.RequestId
		if request.ClientId != "" {
			requests[i]["clientContextID"] = request.ClientId
		}
		requests[i]["state"] = request.State
		requests[i]["scanConsistency"] = request.ScanConsistency
		if request.Statement != "" {
			requests[i]["statement"] = request.Statement
		}
		if request.PreparedName != "" {
			requests[i]["preparedName"] = request.PreparedName
			requests[i]["preparedText"] = request.PreparedText
		}
		requests[i]["requestTime"] = request.Time.String()
		requests[i]["elapsedTime"] = request.ElapsedTime.String()
		requests[i]["serviceTime"] = request.ServiceTime.String()
		requests[i]["resultCount"] = request.ResultCount
		requests[i]["resultSize"] = request.ResultSize
		requests[i]["errorCount"] = request.ErrorCount
		if request.PhaseCounts != nil {
			requests[i]["phaseCounts"] = request.PhaseCounts
		}
		if request.PhaseOperators != nil {
			requests[i]["phaseOperators"] = request.PhaseOperators
		}
		if request.Users != "" {
			requests[i]["users"] = request.Users
		}

		// FIXME more stats
		i++
		return true
	}
	server.RequestsForeach(snapshot, nil)
	return requests, nil
}

func doPreparedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_INDEXES_PREPAREDS
	return prepareds.NamePrepareds(), nil
}

func doRequestIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_INDEXES_ACTIVE_REQUESTS
	numEntries, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}
	requests := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request server.Request) bool {
		if i >= numEntries {
			requests = append(requests, requestId)
		} else {
			requests[i] = requestId
		}
		i++
		return true
	}
	endpoint.actives.ForEach(snapshot, nil)
	return requests, nil
}

func doCompletedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_INDEXES_COMPLETED_REQUESTS
	numEntries := server.RequestsCount()
	completed := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request *server.RequestLogEntry) bool {
		if i >= numEntries {
			completed = append(completed, requestId)
		} else {
			completed[i] = requestId
		}
		i++
		return true
	}
	server.RequestsForeach(snapshot, nil)
	return completed, nil
}

func doFunctionIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_INDEXES_FUNCTIONS
	return functions.NameFunctions(), nil
}

func getMetricData(metric accounting.Metric) map[string]interface{} {
	values := make(map[string]interface{})
	switch metric := metric.(type) {
	case accounting.Counter:
		values["count"] = metric.Count()
	case accounting.Gauge:
		values["value"] = metric.Value()
	case accounting.Meter:
		values["count"] = metric.Count()
		values["1m.rate"] = metric.Rate1()
		values["5m.rate"] = metric.Rate5()
		values["15m.rate"] = metric.Rate15()
		values["mean.rate"] = metric.RateMean()
	case accounting.Timer:
		ps := metric.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = metric.Count()
		values["min"] = metric.Min()
		values["max"] = metric.Max()
		values["mean"] = metric.Mean()
		values["stddev"] = metric.StdDev()
		values["median"] = ps[0]
		values["75%"] = ps[1]
		values["95%"] = ps[2]
		values["99%"] = ps[3]
		values["99.9%"] = ps[4]
		values["1m.rate"] = metric.Rate1()
		values["5m.rate"] = metric.Rate5()
		values["15m.rate"] = metric.Rate15()
		values["mean.rate"] = metric.RateMean()
	case accounting.Histogram:
		ps := metric.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = metric.Count()
		values["min"] = metric.Min()
		values["max"] = metric.Max()
		values["mean"] = metric.Mean()
		values["stddev"] = metric.StdDev()
		values["median"] = ps[0]
		values["75%"] = ps[1]
		values["95%"] = ps[2]
		values["99%"] = ps[3]
		values["99.9%"] = ps[4]
	}
	return values
}
