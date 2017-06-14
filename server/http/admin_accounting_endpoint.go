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
	"net/http"
	"time"

	"github.com/couchbase/query/accounting"
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
		preparedsPrefix + "/{name}":           {handler: preparedHandler, methods: []string{"GET", "DELETE"}},
		requestsPrefix:                        {handler: requestsHandler, methods: []string{"GET"}},
		requestsPrefix + "/{request}":         {handler: requestHandler, methods: []string{"GET", "DELETE"}},
		completedsPrefix:                      {handler: completedsHandler, methods: []string{"GET"}},
		completedsPrefix + "/{request}":       {handler: completedHandler, methods: []string{"GET"}},
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

func doPrepared(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

	switch req.Method {
	case "DELETE":
		err := plan.DeletePrepared(name)
		if err != nil {
			return nil, err
		}
		return true, nil
	case "GET":
		return plan.GetPrepared(value.NewValue(name))
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	switch req.Method {
	case "GET":
		numPrepareds := plan.CountPrepareds()
		data := make([]map[string]interface{}, numPrepareds)
		i := 0

		snapshot := func(name string, d *plan.CacheEntry) {

			// FIXME quick hack to avoid overruns
			if i >= numPrepareds {
				return
			}
			data[i] = map[string]interface{}{}
			data[i]["name"] = d.Prepared.Name()
			data[i]["encoded_plan"] = d.Prepared.EncodedPlan()
			data[i]["statement"] = d.Prepared.Text()
			data[i]["uses"] = d.Uses
			if d.Uses > 0 {
				data[i]["lastUse"] = d.LastUse.String()
			}
			i++
		}

		plan.PreparedsForeach(snapshot)
		return data, nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	switch req.Method {
	case "GET":
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

			p := request.Output().FmtPhaseTimes()
			if p != nil {
				reqMap["phaseTimes"] = p
			}
			p = request.Output().FmtPhaseCounts()
			if p != nil {
				reqMap["phaseCounts"] = p
			}
			p = request.Output().FmtPhaseOperators()
			if p != nil {
				reqMap["phaseOperators"] = p
			}
		})

		return reqMap, nil
	case "DELETE":
		if endpoint.actives.Delete(requestId, true) {
			return nil, errors.NewServiceErrorHttpReq(requestId)
		}

		return true, nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doActiveRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
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

		p := request.Output().FmtPhaseTimes()
		if p != nil {
			requests[i]["phaseTimes"] = p
		}
		p = request.Output().FmtPhaseCounts()
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

	switch req.Method {
	case "GET":
		reqMap := map[string]interface{}{}
		accounting.RequestDo(requestId, func(request *accounting.RequestLogEntry) {
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
			if request.PhaseTimes != nil {
				reqMap["phaseTimes"] = request.PhaseTimes
			}
			if request.PhaseCounts != nil {
				reqMap["phaseCounts"] = request.PhaseCounts
			}
			if request.PhaseOperators != nil {
				reqMap["phaseOperators"] = request.PhaseOperators
			}
		})
		return reqMap, nil
	case "DELETE":
		err := accounting.RequestDelete(requestId)
		if err != nil {
			return nil, err
		} else {
			return true, nil
		}
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doCompletedRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	numRequests := accounting.RequestsCount()
	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request *accounting.RequestLogEntry) {

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
		if request.PhaseTimes != nil {
			requests[i]["phaseTimes"] = request.PhaseTimes
		}
		if request.PhaseCounts != nil {
			requests[i]["phaseCounts"] = request.PhaseCounts
		}
		if request.PhaseOperators != nil {
			requests[i]["phaseOperators"] = request.PhaseOperators
		}

		// FIXME more stats
		i++
	}
	accounting.RequestsForeach(snapshot)
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
	numEntries := accounting.RequestsCount()
	completed := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request *accounting.RequestLogEntry) {
		if i >= numEntries {
			completed = append(completed, requestId)
		} else {
			completed[i] = requestId
		}
		i++
	}
	accounting.RequestsForeach(snapshot)
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
