//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package arrayIndex

import (
	"encoding/json"
	go_er "errors"
	"fmt"
	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
	"io/ioutil"
	http_base "net/http"
	"os"
	"reflect"
	"strconv"
	"time"
)

/*
Global variables accessed by individual test cases for
Couchbase server. Site_CBS, Auth_param, Pool_CBS
and Namespace_CBS represent the site, server authentication
parameters the ip of the couchbase server instance
and the namespace.
*/
var Site_CBS = "http://"
var Auth_param = "Administrator:password"
var Pool_CBS = "127.0.0.1:8091/"
var Namespace_CBS = "default"
var Consistency_parameter = datastore.SCAN_PLUS

func init() {
	logger, _ := log_resolver.NewLogger("golog")
	logging.SetLogger(logger)

}

type MockQuery struct {
	server.BaseRequest
	response    *MockResponse
	resultCount int
}

type MockServer struct {
	server    *server.Server
	acctstore accounting.AccountingStore
}

func (this *MockQuery) OriginalHttpRequest() *http_base.Request {
	return nil
}

func (this *MockQuery) Output() execution.Output {
	return this
}

func (this *MockQuery) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)
	this.response.err = err
	close(this.response.done)
}

func (this *MockQuery) Execute(srvr *server.Server, signature value.Value, stopNotify chan int) {
	defer this.stopAndClose(server.COMPLETED)

	this.NotifyStop(stopNotify)
	this.writeResults()
	close(this.response.done)
}

func (this *MockQuery) Failed(srvr *server.Server) {
	defer this.stopAndClose(server.FATAL)
}

func (this *MockQuery) Expire(state server.State, timeout time.Duration) {
	defer this.stopAndClose(state)

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

/*
Scan consistency implementation. The default
is set to REQUEST_PLUS.
*/
type scanConfigImpl struct {
	scan_level datastore.ScanConsistency
}

func (this *scanConfigImpl) ScanConsistency() datastore.ScanConsistency {
	return this.scan_level
}

func (this *scanConfigImpl) ScanWait() time.Duration {
	return 0
}

func (this *scanConfigImpl) ScanVectorSource() timestamp.ScanVectorSource {
	return &http.ZeroScanVectorSource{}
}

func (this *MockServer) doStats(request *MockQuery) {
	request.CompleteRequest(0, 0, request.resultCount, 0, 0, nil, this.server)
}

var _ALL_USERS = datastore.Credentials{
	"customerowner":  "customerpass",
	"ordersowner":    "orderspass",
	"productowner":   "productpass",
	"purchaseowner":  "purchasepass",
	"reviewowner":    "reviewpass",
	"shellTestowner": "shellTestpass",
}

/*
This method is used to execute the N1QL query represented by
the input argument (q) string using the NewBaseRequest method
as defined in the server request.go.
*/
func Run(mockServer *MockServer, q, namespace string) ([]interface{}, []errors.Error, errors.Error) {
	var metrics value.Tristate
	consistency := &scanConfigImpl{scan_level: datastore.SCAN_PLUS}

	base := server.NewBaseRequest(q, nil, nil, nil, namespace, 0, 0, 0, 0,
		value.FALSE, metrics, value.TRUE, value.TRUE, consistency, "", _ALL_USERS, "", "")

	mr := &MockResponse{
		results: []interface{}{}, warnings: []errors.Error{}, done: make(chan bool),
	}

	query := &MockQuery{
		BaseRequest: *base,
		response:    mr,
	}
	defer mockServer.doStats(query)

	select {
	case mockServer.server.Channel() <- query:
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

/*
Used to specify the N1QL nodes options using the method NewServer
as defined in server/server.go.
*/
func Start(site, pool, namespace string) *MockServer {

	mockServer := &MockServer{}
	datastore, err := resolver.NewDatastore(site + pool)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	sys, err := system.NewDatastore(datastore)
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

	// Start the completed requests log - keep it small and busy
	server.RequestsInit(0, 8)

	channel := make(server.RequestChannel, 10)
	plusChannel := make(server.RequestChannel, 10)

	// need to do it before NewServer() or server scope's changes to
	// the variable and not the package...
	server.SetActives(http.NewActiveRequests())
	server, err := server.NewServer(datastore, sys, configstore, acctstore, namespace,
		false, channel, plusChannel, 4, 4, 0, 0, false, false, false, true,
		server.ProfOff, false)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	server.SetKeepAlive(1 << 10)

	go server.Serve()
	mockServer.server = server
	mockServer.acctstore = acctstore
	return mockServer
}

func dropResultsEntry(results []interface{}, entry interface{}) {
	e := fmt.Sprintf("%v", entry)
	for _, r := range results {
		v, ok := r.(map[string]interface{})
		if ok {
			delete(v, e)
		}
	}
}

func addResultsEntry(newResults, results []interface{}, entry interface{}) {
	e := fmt.Sprintf("%v", entry)
	for i, r := range results {
		v, ok := r.(map[string]interface{})
		if ok {
			newV, ok := newResults[i].(map[string]interface{})
			if ok {
				newV[e] = v[e]
			}
		}
	}
}

func FtestCaseFile(fname string, qc *MockServer, namespace string) (fin_stmt string, errstring error) {
	fin_stmt = ""

	/* Reads the input file and returns its contents in the form
	   of a byte array.
	*/
	b, err := ioutil.ReadFile(fname)
	if err != nil {
		errstring = go_er.New(fmt.Sprintf("ReadFile failed: %v", err))
		return
	}

	var cases []map[string]interface{}

	err = json.Unmarshal(b, &cases)
	if err != nil {
		errstring = go_er.New(fmt.Sprintf("couldn't json unmarshal: %v, err: %v", string(b), err))
		return
	}
	for i, c := range cases {
		d, ok := c["disabled"]
		if ok {
			disabled := d.(bool)
			if disabled == true {
				continue
			}
		}

		/* Handles all queries to be run against CBServer and Datastore */
		v, ok := c["statements"]
		if !ok || v == nil {
			errstring = go_er.New(fmt.Sprintf("missing statements for case file: %v, index: %v", fname, i))
			return
		}
		statements := v.(string)
		//t.Logf("  %d: %v\n", i, statements)
		fin_stmt = strconv.Itoa(i) + ": " + statements
		resultsActual, _, errActual := Run(qc, statements, namespace)

		errExpected := ""
		v, ok = c["error"]
		if ok {
			errExpected = v.(string)
		}

		if errActual != nil {
			if errExpected == "" {
				errstring = go_er.New(fmt.Sprintf("unexpected err: %v, statements: %v"+
					", for case file: %v, index: %v", errActual, statements, fname, i))
				return
			}
			// TODO: Check that the actual err matches the expected err.
			continue
		}

		if errExpected != "" {
			errstring = go_er.New(fmt.Sprintf("did not see the expected err: %v, statements: %v"+
				", for case file: %v, index: %v", errActual, statements, fname, i))
			return
		}

		// ignore certain parts of the results if we need to
		// we handle scalars and array of scalars, ignore the rest
		// filter only applied to first level fields
		ignore, ok := c["ignore"]
		if ok {
			switch ignore.(type) {
			case []interface{}:
				for _, v := range ignore.([]interface{}) {
					switch v.(type) {
					case []interface{}:
					case map[string]interface{}:
					default:
						dropResultsEntry(resultsActual, v)
					}
				}
			case map[string]interface{}:
			default:
				dropResultsEntry(resultsActual, ignore)
			}
		}

		// opposite of ignore - only select certain fields
		// again, we handle scalars and the scalars in an array
		accept, ok := c["accept"]
		if ok {
			newResults := make([]interface{}, len(resultsActual))
			switch accept.(type) {
			case []interface{}:
				for i, _ := range resultsActual {
					newResults[i] = make(map[string]interface{}, len(accept.([]interface{})))
				}
				for _, v := range accept.([]interface{}) {
					switch v.(type) {
					case []interface{}:
					case map[string]interface{}:
					default:
						addResultsEntry(newResults, resultsActual, v)
					}
				}
			case map[string]interface{}:
			default:
				for i, _ := range resultsActual {
					newResults[i] = make(map[string]interface{}, 1)
				}
				addResultsEntry(newResults, resultsActual, accept)
			}
			resultsActual = newResults
		}
		v, ok = c["results"]
		if ok {
			resultsExpected := v.([]interface{})
			okres := doResultsMatch(resultsActual, resultsExpected, fname, i)
			if okres != nil {
				errstring = okres
				return
			}
		}
	}
	return fin_stmt, nil
}

/*
Matches expected results with the results obtained by
running the queries.
*/
func doResultsMatch(resultsActual, resultsExpected []interface{}, fname string, i int) (errstring error) {
	if len(resultsActual) != len(resultsExpected) {
		errstring = go_er.New(fmt.Sprintf("results len don't match, %v vs %v, %v vs %v"+
			", for case file: %v, index: %v",
			len(resultsActual), len(resultsExpected),
			resultsActual, resultsExpected, fname, i))
		return
	}

	if !reflect.DeepEqual(resultsActual, resultsExpected) {
		errstring = go_er.New(fmt.Sprintf("results don't match, actual: %#v, expected: %#v"+
			", for case file: %v, index: %v",
			resultsActual, resultsExpected, fname, i))

		return
	}
	return nil
}
