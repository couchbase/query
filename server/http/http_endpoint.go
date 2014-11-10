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

	"github.com/couchbaselabs/query/logging"
	"github.com/couchbaselabs/query/server"
)

type HttpEndpoint struct {
	server  *server.Server
	metrics bool
	httpsrv http.Server
}

func NewHttpEndpoint(server *server.Server, metrics bool, addr string) *HttpEndpoint {
	rv := &HttpEndpoint{
		server:  server,
		metrics: metrics,
	}

	rv.httpsrv.Addr = addr
	rv.httpsrv.Handler = rv

	// Bind HttpEndpoint object to /query/service endpoint; use default Server Mux
	http.Handle("/query/service", rv)

	// TODO: Deprecate (remove) this binding after QE has migrated to /query/service
	http.Handle("/query", rv)

	http.HandleFunc("/stats", statsHandler)

	return rv
}

func statsHandler(w http.ResponseWriter, req *http.Request) {
	http.Redirect(w, req, "/debug/vars", http.StatusFound)
}

func (this *HttpEndpoint) ListenAndServe() error {
	return http.ListenAndServe(this.httpsrv.Addr, nil)
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	request := newHttpRequest(resp, req)

	if request.State() == server.FATAL {
		// There was problems creating the request: Fail it and return
		request.Failed(this.server)
		return
	}
	select {
	case this.server.Channel() <- request:
		// Wait until the request exits.
		<-request.CloseNotify()
	default:
		// Timeout.
		resp.WriteHeader(http.StatusServiceUnavailable)
	}

	// Update metrics TODO:
	request_time := time.Since(request.RequestTime())
	service_time := time.Since(request.ServiceTime())
	ms := this.server.AccountingStore().MetricRegistry()
	ms.Counter("request_count").Inc(1)
	ms.Counter("request_overall_time").Inc(int64(request_time))
	ms.Meter("request_rate").Mark(1)
	ms.Meter("error_rate").Mark(int64(request.errorCount))
	ms.Histogram("response_count").Update(int64(request.resultCount))
	ms.Timer("request_time").Update(request_time)
	ms.Timer("service_time").Update(service_time)

	logging.Infop("Finished request:",
		logging.Pair{"request_state", request.State()},
		logging.Pair{"result_count", request.resultCount},
		logging.Pair{"error_count", request.errorCount},
		logging.Pair{"warning_count", request.warningCount},
		logging.Pair{"request_time", request_time},
		logging.Pair{"service_time", service_time},
		logging.Pair{"request_time_nanosecs", int64(request_time)},
		logging.Pair{"service_time_nanosecs", int64(service_time)},
	)

}
