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
	return rv
}

func (this *HttpEndpoint) ListenAndServe() error {
	return this.httpsrv.ListenAndServe()
}

// If the server channel is full and we are unable to queue a request,
// we respond with a timeout status.
func (this *HttpEndpoint) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	request := newHttpRequest(resp, req)
	select {
	case this.server.Channel() <- request:
		// Wait until the request exits.
		<-request.CloseNotify()
	default:
		// Timeout.
		resp.WriteHeader(http.StatusServiceUnavailable)
	}
}
