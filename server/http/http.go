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
	"io"
	"net/http"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/value"
)

const MAX_REQUEST_BYTES = 1 << 20

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

type httpRequest struct {
	server.BaseRequest
	resp http.ResponseWriter
	req  *http.Request
}

func newHttpRequest(resp http.ResponseWriter, req *http.Request) *httpRequest {
	// XXX TODO
	base := &server.BaseRequest{}

	rv := &httpRequest{
		BaseRequest: *base,
	}

	req.Body = http.MaxBytesReader(resp, req.Body, MAX_REQUEST_BYTES)

	// Abort if client closes connection
	closeNotify := resp.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		rv.Stop(server.TIMEOUT)
	}()

	return rv
}

func (this *httpRequest) Output() execution.Output {
	return this
}

func (this *httpRequest) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)

	this.resp.WriteHeader(http.StatusInternalServerError)
	io.WriteString(this.resp, err.Error())
}

func (this *httpRequest) Execute(stopNotify chan bool) {
	defer this.Stop(server.COMPLETED)

	this.NotifyStop(stopNotify)

	this.resp.WriteHeader(http.StatusOK)
	_ = this.writePrefix() &&
		this.writeResults() &&
		this.writeSuffix()
}

func (this *httpRequest) Expire() {
	defer this.Stop(server.TIMEOUT)

	this.writeSuffix()
}

func (this *httpRequest) writePrefix() bool {
	io.WriteString(this.resp, "{\n")
	io.WriteString(this.resp, "  \"results\": [\n")
	return true
}

func (this *httpRequest) writeResults() bool {
	var item value.Value

	ok := true
	for ok {
		select {
		case <-this.StopExecute():
			break
		default:
		}

		select {
		case item = <-this.Results():
			ok = this.writeResult(item)
		case <-this.StopExecute():
			break
		}
	}

	return ok
}

func (this *httpRequest) writeResult(item value.Value) bool {
	// XXX TODO
	return true
}

func (this *httpRequest) writeSuffix() bool {
	// XXX TODO
	return true
}
