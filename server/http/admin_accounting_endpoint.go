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

	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/gorilla/mux"
)

const (
	accountingPrefix = adminPrefix + "/stats"
	expvarsRoute     = "/debug/vars"
)

func expvarsHandler(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, accountingPrefix, http.StatusFound)
}

func registerAccountingHandlers(r *mux.Router, server *server.Server) {
	statsHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server, w, req, doStats)
	}
	statHandler := func(w http.ResponseWriter, req *http.Request) {
		wrapAPI(server, w, req, doStat)
	}

	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:             {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}": {handler: statHandler, methods: []string{"GET", "DELETE"}},
	}

	for route, h := range routeMap {
		r.HandleFunc(route, h.handler).Methods(h.methods...)
	}

	r.HandleFunc(expvarsRoute, expvarsHandler).Methods("GET")

}

func doStats(s *server.Server, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	acctStore := s.AccountingStore()
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

func doStat(s *server.Server, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["stat"]
	acctStore := s.AccountingStore()
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
