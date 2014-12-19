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

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/server"
	"github.com/gorilla/mux"
)

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

}

func doStats(s *server.Server, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	acctStore := s.AccountingStore()
	reg := acctStore.MetricRegistry()

	switch req.Method {
	case "GET":
		var stats map[string]interface{}
		stats["Counters"] = reg.Counters()
		stats["Gauges"] = reg.Gauges()
		stats["Timers"] = reg.Timers()
		stats["Meters"] = reg.Meters()
		stats["Histogram"] = reg.Histograms()
		return stats, nil
	default:
		return nil, nil
	}
}

func doStat(s *server.Server, w http.ResponseWriter, req *http.Request) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["stat"]
	acctStore := s.AccountingStore()
	reg := acctStore.MetricRegistry()

	switch req.Method {
	case "GET":
		return reg.Get(name), nil
	case "DELETE":
		return nil, reg.Unregister(name)
	default:
		return nil, nil
	}
}
