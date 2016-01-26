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
	"bytes"
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
	completedPrefix  = adminPrefix + "/completed_requests"
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
	completedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequests)
	}
	requestHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequest)
	}
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:              {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}":  {handler: statHandler, methods: []string{"GET", "DELETE"}},
		vitalsPrefix:                  {handler: vitalsHandler, methods: []string{"GET"}},
		preparedsPrefix:               {handler: preparedsHandler, methods: []string{"GET"}},
		preparedsPrefix + "/{name}":   {handler: preparedHandler, methods: []string{"GET", "DELETE"}},
		requestsPrefix:                {handler: requestsHandler, methods: []string{"GET"}},
		requestsPrefix + "/{request}": {handler: requestHandler, methods: []string{"GET", "DELETE"}},
		completedPrefix:               {handler: completedHandler, methods: []string{"GET"}},
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
		return nil, nil
	}
}

func addMetricData(name string, stats map[string]interface{}, metrics map[string]interface{}) {
	var key_name bytes.Buffer
	for metric_type, metric_value := range metrics {
		key_name.WriteString(name)
		key_name.WriteString(".")
		key_name.WriteString(metric_type)
		stats[key_name.String()] = metric_value
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
		return nil, nil
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
		return nil, nil
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
		return nil, nil
	}
}

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	switch req.Method {
	case "GET":
		return plan.SnapshotPrepared(), nil
	default:
		return nil, nil
	}
}

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

	switch req.Method {
	case "GET":
		request, _ := endpoint.actives.Get(requestId)
		reqMap := map[string]interface{}{}
		reqMap["requestId"] = request.Id().String()
		if request.Statement() != "" {
			reqMap["request.statement"] = request.Statement()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			reqMap["prepared.name"] = p.Name()
			reqMap["prepared.statement"] = p.Text()
		}
		reqMap["requestTime"] = request.RequestTime()
		reqMap["elapsedTime"] = time.Since(request.RequestTime()).String()
		reqMap["executionTime"] = time.Since(request.ServiceTime()).String()
		reqMap["state"] = request.State()

		// FIXME more stats
		// PhaseTimes() is not in server.Request API
		httpRequest, isHttp := request.(*httpRequest)
		if isHttp {
			for phase, phaseTime := range httpRequest.PhaseTimes() {
				reqMap[phase] = phaseTime.String()
			}
		}
		return reqMap, nil
	case "DELETE":
		if endpoint.actives.Delete(requestId, true) {
			return nil, errors.NewServiceErrorHttpReq(requestId)
		}

		return true, nil
	default:
		return nil, nil
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
		if request.Statement() != "" {
			requests[i]["request.statement"] = request.Statement()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			requests[i]["prepared.name"] = p.Name()
			requests[i]["prepared.statement"] = p.Text()
		}
		requests[i]["requestTime"] = request.RequestTime()
		requests[i]["elapsedTime"] = time.Since(request.RequestTime()).String()
		requests[i]["executionTime"] = time.Since(request.ServiceTime()).String()
		requests[i]["state"] = request.State()

		// FIXME more stats
		// PhaseTimes() is not in server.Request API
		httpRequest, isHttp := request.(*httpRequest)
		if isHttp {
			for phase, phaseTime := range httpRequest.PhaseTimes() {
				requests[i][phase] = phaseTime.String()
			}
		}
		i++
	}
	endpoint.actives.ForEach(snapshot)
	return requests, nil
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
		if request.Statement != "" {
			requests[i]["statement"] = request.Statement
		}
		if request.PreparedName != "" {
			requests[i]["preparedName"] = request.PreparedName
			requests[i]["preparedText"] = request.PreparedText
		}
		requests[i]["requestTime"] = request.Time
		requests[i]["elapsedTime"] = request.ElapsedTime
		requests[i]["serviceTime"] = request.ServiceTime
		requests[i]["resultCount"] = request.ResultCount
		requests[i]["resultSize"] = request.ResultSize
		requests[i]["errorCount"] = request.ErrorCount
		requests[i]["sortCount"] = request.SortCount

		// FIXME more stats
		i++
	}
	accounting.RequestsForeach(snapshot)
	return requests, nil
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
