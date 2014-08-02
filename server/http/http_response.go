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
