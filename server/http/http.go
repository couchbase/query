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
	//"github.com/couchbaselabs/query/value"
)

const MAX_REQUEST_BYTES = 1 << 20

type HttpReceptor struct {
	server  *server.Server
	metrics bool
	httpsrv http.Server
}

func NewHttpReceptor(server *server.Server, metrics bool, addr string) *HttpReceptor {
	rv := &HttpReceptor{
		server:  server,
		metrics: metrics,
	}

	rv.httpsrv.Addr = addr
	rv.httpsrv.Handler = rv
	return rv
}

func (this *HttpReceptor) ListenAndServe() error {
	return this.httpsrv.ListenAndServe()
}

func (this *HttpReceptor) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	request := newHttpRequest(resp, req)
	this.server.Channel() <- request
	request.Await()
}

type httpRequest struct {
	server.BaseRequest
	resp http.ResponseWriter
	req  *http.Request
}

func newHttpRequest(resp http.ResponseWriter, req *http.Request) *httpRequest {
	rv := &httpRequest{}

	req.Body = http.MaxBytesReader(resp, req.Body, MAX_REQUEST_BYTES)
	return rv
}

func (this *httpRequest) Output() execution.Output {
	return this
}

func (this *httpRequest) Fail(err errors.Error) {
	this.resp.WriteHeader(http.StatusExpectationFailed)
	io.WriteString(this.resp, err.Error())
	this.Stop(server.FATAL)
}

func (this *httpRequest) Execute() {
	this.resp.WriteHeader(http.StatusOK)
	this.Stop(server.COMPLETED)
}

func (this *httpRequest) Expire() {
	this.Stop(server.TIMEOUT)
}
