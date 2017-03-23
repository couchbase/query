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
	"net/http"
	"time"

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
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
	indexesPrefix    = adminPrefix + "/indexes"
	expvarsRoute     = "/debug/vars"
)

func expvarsHandler(w http.ResponseWriter, req *http.Request) {
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
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:                      {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}":          {handler: statHandler, methods: []string{"GET", "DELETE"}},
		vitalsPrefix:                          {handler: vitalsHandler, methods: []string{"GET"}},
		preparedsPrefix:                       {handler: preparedsHandler, methods: []string{"GET"}},
		preparedsPrefix + "/{name}":           {handler: preparedHandler, methods: []string{"GET", "POST", "DELETE"}},
		requestsPrefix:                        {handler: requestsHandler, methods: []string{"GET"}},
		requestsPrefix + "/{request}":         {handler: requestHandler, methods: []string{"GET", "POST", "DELETE"}},
		completedsPrefix:                      {handler: completedsHandler, methods: []string{"GET"}},
		completedsPrefix + "/{request}":       {handler: completedHandler, methods: []string{"GET", "POST", "DELETE"}},
		indexesPrefix + "/prepareds":          {handler: preparedIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/active_requests":    {handler: requestIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/completed_requests": {handler: completedIndexHandler, methods: []string{"GET"}},
	}

	for route, h := range routeMap {
		this.mux.HandleFunc(route, h.handler).Methods(h.methods...)
	}

	this.mux.HandleFunc(expvarsRoute, expvarsHandler).Methods("GET")

	this.mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}

func doStats(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

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

func doStat(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["stat"]
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

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

func doNotFound(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()
	if reg == nil {
		logging.Errorf("http.NotFoundHandler - nil metric registry")
	} else {
		reg.Counter(accounting.INVALID_REQUESTS).Inc(1)
	}

	return nil, nil
}

func doVitals(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
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
func getCredentialsFromRequest(req *http.Request) (datastore.Credentials, errors.Error) {
	creds := make(datastore.Credentials)
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

func verifyCredentialsFromRequest(api string, req *http.Request) errors.Error {
	creds, err := getCredentialsFromRequest(req)
	if err != nil {
		return err
	}
	privs := datastore.NewPrivileges()
	privs.Add("system:"+api, datastore.PRIV_SYSTEM_READ)
	_, err = datastore.GetDatastore().Authorize(privs, creds, req)
	return err
}

func doPrepared(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

	if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("prepareds", req)
		if err != nil {
			return nil, err
		}
		err = plan.DeletePrepared(name)
		if err != nil {
			return nil, err
		}
		return true, nil
	} else if req.Method == "GET" || req.Method == "POST" {
		err := verifyCredentialsFromRequest("prepareds", req)
		if err != nil {
			return nil, err
		}

		var itemMap map[string]interface{}

		plan.PreparedDo(name, func(entry *plan.CacheEntry) {
			itemMap = map[string]interface{}{
				"name":         name,
				"uses":         entry.Uses,
				"statement":    entry.Prepared.Text(),
				"encoded_plan": entry.Prepared.EncodedPlan(),
			}
			if req.Method == "POST" {
				itemMap["plan"] = entry.Prepared.Operator
			}
			if entry.Uses > 0 {
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

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	switch req.Method {
	case "GET":
		err := verifyCredentialsFromRequest("prepareds", req)
		if err != nil {
			return nil, err
		}
		return plan.SnapshotPrepared(), nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	if req.Method == "GET" || req.Method == "POST" {
		err := verifyCredentialsFromRequest("actives", req)
		if err != nil {
			return nil, err
		}
		reqMap := activeRequestWorkHorse(endpoint, requestId, (req.Method == "POST"))

		return reqMap, nil
	} else if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("actives", req)
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
		reqMap["requestTime"] = request.RequestTime()
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
			prof := request.Profile()
			if prof == server.ProfUnset {
				prof = endpoint.server.Profile()
			}
			if prof != server.ProfOff {
				reqMap["phaseTimes"] = request.Output().FmtPhaseTimes()
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

func doActiveRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	err := verifyCredentialsFromRequest("actives", req)
	if err != nil {
		return nil, err
	}

	numRequests, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}

	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request server.Request) {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return
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
		requests[i]["requestTime"] = request.RequestTime()
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
	}
	endpoint.actives.ForEach(snapshot)
	return requests, nil
}

func doCompletedRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	if req.Method == "GET" || req.Method == "POST" {
		err := verifyCredentialsFromRequest("completed_requests", req)
		if err != nil {
			return nil, err
		}
		reqMap := completedRequestWorkHorse(requestId, (req.Method == "POST"))
		return reqMap, nil
	} else if req.Method == "DELETE" {
		err := verifyCredentialsFromRequest("completed_requests", req)
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
		reqMap["requestTime"] = request.Time
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

func doCompletedRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	err := verifyCredentialsFromRequest("completed_requests", req)
	if err != nil {
		return nil, err
	}

	numRequests := server.RequestsCount()
	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request *server.RequestLogEntry) {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return
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
		requests[i]["requestTime"] = request.Time
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
	}
	server.RequestsForeach(snapshot)
	return requests, nil
}

func doPreparedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	return plan.NamePrepareds(), nil
}

func doRequestIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	numEntries, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}
	requests := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request server.Request) {
		if i >= numEntries {
			requests = append(requests, requestId)
		} else {
			requests[i] = requestId
		}
		i++
	}
	endpoint.actives.ForEach(snapshot)
	return requests, nil
}

func doCompletedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	numEntries := server.RequestsCount()
	completed := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request *server.RequestLogEntry) {
		if i >= numEntries {
			completed = append(completed, requestId)
		} else {
			completed[i] = requestId
		}
		i++
	}
	server.RequestsForeach(snapshot)
	return completed, nil
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
