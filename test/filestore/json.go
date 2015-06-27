//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package test

import (
	"encoding/json"
	"os"

	acct_resolver "github.com/couchbase/query/accounting/resolver"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
)

func init() {
	logger, _ := log_resolver.NewLogger("golog")
	logging.SetLogger(logger)
}

type MockQuery struct {
	server.BaseRequest
	response    *MockResponse
	resultCount int
}

func (this *MockQuery) Output() execution.Output {
	return this
}

func (this *MockQuery) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)
	this.response.err = err
	close(this.response.done)
}

func (this *MockQuery) Execute(srvr *server.Server, signature value.Value, stopNotify chan bool) {
	defer this.stopAndClose(server.COMPLETED)

	this.NotifyStop(stopNotify)
	this.writeResults()
	close(this.response.done)
}

func (this *MockQuery) Failed(srvr *server.Server) {
	this.stopAndClose(server.FATAL)
}

func (this *MockQuery) Expire() {
	defer this.stopAndClose(server.TIMEOUT)

	this.response.err = errors.NewError(nil, "Query timed out")
	close(this.response.done)
}

func (this *MockQuery) stopAndClose(state server.State) {
	this.Stop(state)
	this.Close()
}

func (this *MockQuery) writeResults() bool {
	var item value.Value

	ok := true
	for ok {
		select {
		case <-this.StopExecute():
			this.SetState(server.STOPPED)
			return true
		default:
		}

		select {
		case item, ok = <-this.Results():
			if ok {
				if !this.writeResult(item) {
					this.SetState(server.FATAL)
					return false
				}
			}
		case <-this.StopExecute():
			this.SetState(server.STOPPED)
			return true
		}
	}

	this.SetState(server.COMPLETED)
	return true
}

func (this *MockQuery) writeResult(item value.Value) bool {
	bytes, err := json.Marshal(item)
	if err != nil {
		panic(err.Error())
	}

	this.resultCount++

	var resultLine map[string]interface{}
	json.Unmarshal(bytes, &resultLine)

	this.response.results = append(this.response.results, resultLine)
	return true
}

type MockResponse struct {
	err      errors.Error
	results  []interface{}
	warnings []errors.Error
	done     chan bool
}

func (this *MockResponse) NoMoreResults() {
	close(this.done)
}

func Run(mockServer *server.Server, q string) ([]interface{}, []errors.Error, errors.Error) {
	var metrics value.Tristate
	base := server.NewBaseRequest(q, nil, nil, nil, "json", 0, value.FALSE, metrics, value.TRUE, nil, "", nil)

	mr := &MockResponse{
		results: []interface{}{}, warnings: []errors.Error{}, done: make(chan bool),
	}

	query := &MockQuery{
		BaseRequest: *base,
		response:    mr,
	}

	select {
	case mockServer.Channel() <- query:
		// Wait until the request exits.
		<-query.CloseNotify()
	default:
		// Timeout.
		return nil, nil, errors.NewError(nil, "Query timed out")
	}

	// wait till all the results are ready
	<-mr.done
	return mr.results, mr.warnings, mr.err
}

func Start(site, pool string) *server.Server {

	datastore, err := resolver.NewDatastore("dir:./json")
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	configstore, err := config_resolver.NewConfigstore("stub:")
	if err != nil {
		logging.Errorp("Could not connect to configstore",
			logging.Pair{"error", err},
		)
	}

	acctstore, err := acct_resolver.NewAcctstore("stub:")
	if err != nil {
		logging.Errorp("Could not connect to acctstore",
			logging.Pair{"error", err},
		)
	}

	channel := make(server.RequestChannel, 10)
	server, err := server.NewServer(datastore, configstore, acctstore, "json",
		false, channel, 4, 0, 0, false, false)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	server.SetKeepAlive(1 << 10)

	go server.Serve()
	return server
}
