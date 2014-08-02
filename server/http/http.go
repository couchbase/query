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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/plan"
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
	err := req.ParseForm()

	var command string
	if err == nil {
		command, err = getCommand(req)
	}

	var prepared plan.Operator
	if err == nil {
		prepared, err = getPrepared(req)
	}

	if err == nil && command == "" && prepared == nil {
		err = fmt.Errorf("Either command or prepared must be provided.")
	}

	var arguments map[string]value.Value
	if err == nil {
		arguments, err = getArguments(req)
	}

	var namespace string
	if err == nil {
		namespace, err = formValue(req, "namespace")
	}

	var timeout time.Duration
	if err == nil {
		timeout, err = getTimeout(req)
	}

	readonly := req.Method == "GET"
	if err == nil {
		readonly, err = getReadonly(req)
	}

	var metrics value.Tristate
	if err == nil {
		metrics, err = getMetrics(req)
	}

	base := server.NewBaseRequest(command, prepared, arguments, namespace, readonly, metrics)
	rv := &httpRequest{
		BaseRequest: *base,
		resp:        resp,
		req:         req,
	}

	rv.SetTimeout(rv, timeout)

	// Limit body size in case of denial-of-service attack
	req.Body = http.MaxBytesReader(resp, req.Body, MAX_REQUEST_BYTES)

	// Abort if client closes connection
	closeNotify := resp.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		rv.Stop(server.TIMEOUT)
	}()

	if err != nil {
		rv.Fail(errors.NewError(err, ""))
	}

	return rv
}

func (this *httpRequest) Output() execution.Output {
	return this
}

func (this *httpRequest) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)

	this.resp.WriteHeader(http.StatusInternalServerError)
	this.writeString(err.Error())
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
	return this.writeString("{\n  \"results\": [")
}

func (this *httpRequest) writeResults() bool {
	var item value.Value

	ok := true
	for ok {
		select {
		case <-this.StopExecute():
			return true
		default:
		}

		select {
		case item, ok = <-this.Results():
			if ok {
				if !this.writeResult(item) {
					return false
				}
			}
		case <-this.StopExecute():
			return true
		}
	}

	return true
}

func (this *httpRequest) writeResult(item value.Value) bool {
	// XXX TODO
	return true
}

func (this *httpRequest) writeSuffix() bool {
	// XXX TODO
	return this.writeString("\n  ]\n}\n")
}

func (this *httpRequest) writeString(s string) bool {
	_, err := io.WriteString(this.resp, s)
	return err == nil
}

func formValue(req *http.Request, field string) (string, error) {
	values := req.Form[field]

	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	default:
		return "", fmt.Errorf("Multiple values for field %s.", field)
	}
}

func getCommand(req *http.Request) (string, error) {
	command, err := formValue(req, "command")
	if err != nil {
		return "", err
	}

	if command == "" && req.Method == "POST" {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return "", err
		}

		command = string(bytes)
	}

	return command, nil
}

func getPrepared(req *http.Request) (plan.Operator, error) {
	var prepared plan.Operator

	prepared_field, err := formValue(req, "prepared")
	if err == nil && prepared_field != "" {
		// XXX TODO unmarshal
		prepared = nil
	}

	return prepared, err
}

// XXX TODO
func getArguments(req *http.Request) (map[string]value.Value, error) {
	var arguments map[string]value.Value

	// XXX TODO
	return arguments, nil
}

func getTimeout(req *http.Request) (time.Duration, error) {
	var timeout time.Duration

	timeout_field, err := formValue(req, "timeout")
	if err == nil && timeout_field != "" {
		timeout, err = time.ParseDuration(timeout_field)
	}

	return timeout, err
}

func getReadonly(req *http.Request) (bool, error) {
	readonly := req.Method == "GET"

	readonly_field, err := formValue(req, "readonly")
	if err == nil && readonly_field != "" {
		readonly, err = strconv.ParseBool(readonly_field)
		if err != nil && !readonly && req.Method == "GET" {
			readonly = true
			err = fmt.Errorf("readonly=false cannot be used with HTTP GET method.")
		}
	}

	return readonly, err
}

func getMetrics(req *http.Request) (value.Tristate, error) {
	var metrics value.Tristate

	metrics_field, err := formValue(req, "metrics")
	if err == nil && metrics_field != "" {
		m, err := strconv.ParseBool(metrics_field)
		if err == nil {
			metrics = value.ToTristate(m)
		}
	}

	return metrics, err
}
