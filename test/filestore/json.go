//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package test

import (
	"encoding/json"
	go_er "errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/functions/constructor"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"
)

var Namespace_FS = "dimestore"

func init() {
	logger, _ := log_resolver.NewLogger("golog")
	logging.SetLogger(logger)
	runtime.GOMAXPROCS(1)
}

type MockQuery struct {
	server.BaseRequest
	response    *MockResponse
	resultCount int
}

type MockServer struct {
	server    *server.Server
	acctstore accounting.AccountingStore
	dstore    datastore.Datastore
}

func (this *MockQuery) Output() execution.Output {
	return this
}

func (this *MockQuery) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)
	this.response.err = err
	close(this.response.done)
}

func (this *MockQuery) Execute(srvr *server.Server, context *execution.Context, reqType string, signature value.Value, dummy bool) {
	select {
	case <-this.Results():
		this.Stop(server.COMPLETED)
	case <-this.StopExecute():
		this.Stop(server.STOPPED)

		// wait for operator before continuing
		<-this.Results()
	}
	close(this.response.done)
}

func (this *MockQuery) Failed(srvr *server.Server) {
	this.Stop(server.FATAL)
}

func (this *MockQuery) Expire(state server.State, timeout time.Duration) {
	defer this.Stop(state)

	this.response.err = errors.NewError(nil, "Query timed out")
	close(this.response.done)
}

func (this *MockQuery) SetUp() {
}

func (this *MockQuery) Result(item value.AnnotatedValue) bool {
	bytes, err := json.Marshal(item)
	if err != nil {
		this.SetState(server.FATAL)
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

type scanConfigImpl struct {
}

func (this *scanConfigImpl) ScanConsistency() datastore.ScanConsistency {
	return datastore.SCAN_PLUS
}

func (this *scanConfigImpl) ScanWait() time.Duration {
	return 0
}

func (this *scanConfigImpl) SetScanConsistency(consistency datastore.ScanConsistency) interface{} {
	return this
}

func (this *scanConfigImpl) ScanVectorSource() timestamp.ScanVectorSource {
	return &http.ZeroScanVectorSource{}
}

func (this *MockServer) doStats(request *MockQuery) {
	request.CompleteRequest(0, 0, 0, request.resultCount, 0, 0, nil, this.server)
}

func Run(mockServer *MockServer, p bool, q string, namedArgs map[string]value.Value, positionalArgs []value.Value, namespace string) ([]interface{}, []errors.Error, errors.Error) {
	var metrics value.Tristate
	scanConfiguration := &scanConfigImpl{}

	pretty := value.TRUE
	if !p {
		pretty = value.FALSE
	}

	mr := &MockResponse{
		results: []interface{}{}, warnings: []errors.Error{}, done: make(chan bool),
	}
	query := &MockQuery{
		response: mr,
	}
	server.NewBaseRequest(&query.BaseRequest)
	query.SetStatement(q)
	query.SetNamedArgs(namedArgs)
	query.SetPositionalArgs(positionalArgs)
	query.SetNamespace(namespace)
	query.SetReadonly(value.FALSE)
	query.SetMetrics(metrics)
	query.SetSignature(value.TRUE)
	query.SetPretty(pretty)
	query.SetScanConfiguration(scanConfiguration)

	defer mockServer.doStats(query)

	if !mockServer.server.ServiceRequest(query) {
		return nil, nil, errors.NewError(nil, "Query timed out")
	}

	// wait till all the results are ready
	<-mr.done
	return mr.results, mr.warnings, mr.err
}

func Start(site, pool, namespace string) *MockServer {

	mockServer := &MockServer{}
	ds, err := resolver.NewDatastore(site + pool)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}
	datastore.SetDatastore(ds)

	sys, err := system.NewDatastore(ds)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}
	datastore.SetSystemstore(sys)

	configstore, err := config_resolver.NewConfigstore("stub:")
	if err != nil {
		logging.Errorf("Could not connect to configstore: %v", err)
	}

	acctstore, err := acct_resolver.NewAcctstore("stub:")
	if err != nil {
		logging.Errorf("Could not connect to acctstore: %v", err)
	}

	// Start the completed requests log - keep it small and busy
	server.RequestsInit(0, 8)

	// Start the prepared statement cache
	prepareds.PreparedsInit(1024)

	srv, err := server.NewServer(ds, sys, configstore, acctstore, namespace,
		false, 10, 10, 4, 4, 0, 0, false, false, false, true,
		server.ProfOff, false)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}
	server.SetActives(http.NewActiveRequests(srv))
	prepareds.PreparedsReprepareInit(ds, sys)
	constructor.Init(nil, 6)

	srv.SetKeepAlive(1 << 10)
	srv.SetMaxIndexAPI(datastore.INDEX_API_MAX)

	mockServer.server = srv
	mockServer.acctstore = acctstore
	mockServer.dstore = ds
	return mockServer
}

func dropResultEntry(result interface{}, e string) {
	switch v := result.(type) {
	case map[string]interface{}:
		delete(v, e)
		for _, f := range v {
			dropResultEntry(f, e)
		}
	case []interface{}:
		for _, f := range v {
			dropResultEntry(f, e)
		}
	}
}

func dropResultsEntry(results []interface{}, entry interface{}) {
	e := fmt.Sprintf("%v", entry)
	for _, r := range results {
		dropResultEntry(r, e)
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

	ffname, e := filepath.Abs(fname)
	if e != nil {
		ffname = fname
	}

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

		v, ok := c["preStatements"]
		if ok {
			preStatements := v.(string)
			_, _, err := Run(qc, true, preStatements, nil, nil, namespace)
			if err != nil {
				go_er.New(fmt.Sprintf("preStatements resulted in error: %v, for case file: %v, index: %v%s", err, ffname, i, findIndex(b, i)))
				return
			}
		}

		var namedArgs map[string]value.Value
		var positionalArgs value.Values

		if n, ok1 := c["namedArgs"]; ok1 {
			nv := value.NewValue(n)
			size := len(nv.Fields())
			if size == 0 {
				size = 1
			}
			namedArgs = make(map[string]value.Value, size)
			for f, v := range nv.Fields() {
				namedArgs[f] = value.NewValue(v)
			}
		}

		if p, ok2 := c["positionalArgs"]; ok2 {
			if pa, ok3 := p.([]interface{}); ok3 {
				for _, v := range pa {
					positionalArgs = append(positionalArgs, value.NewValue(v))
				}
			}
		}

		/* Handles all queries to be run against CBServer and Datastore */
		v, ok = c["statements"]
		if !ok || v == nil {
			errstring = go_er.New(fmt.Sprintf("missing statements for case file: %v, index: %v%s", ffname, i, findIndex(b, i)))
			return
		}
		statements := v.(string)
		//t.Logf("  %d: %v\n", i, statements)
		fin_stmt = strconv.Itoa(i) + ": " + statements
		resultsActual, _, errActual := Run(qc, true, statements, namedArgs, positionalArgs, namespace)

		errCodeExpected := int(0)
		errExpected := ""

		v, ok = c["postStatements"]
		if ok {
			postStatements := v.(string)
			_, _, err := Run(qc, true, postStatements, nil, nil, namespace)
			if err != nil {
				errstring = go_er.New(fmt.Sprintf("postStatements resulted in error: %v\nfor case file: %v, index: %v%s", err, ffname, i, findIndex(b, i)))
				return
			}
		}

		v, ok = c["matchStatements"]
		if ok {
			matchStatements := v.(string)
			resultsMatch, _, errMatch := Run(qc, true, matchStatements, nil, nil, namespace)
			if !reflect.DeepEqual(errActual, errActual) {
				errstring = go_er.New(fmt.Sprintf("errors don't match\n  actual: %#v\nexpected: %#v\n"+
					" for case file: %v, index: %v%s",
					errActual, errMatch, ffname, i, findIndex(b, i)))
				return
			}
			doResultsMatch(resultsActual, resultsMatch, fname, i, b, matchStatements)
		}

		v, ok = c["error"]
		if ok {
			errExpected = v.(string)
		}

		if v, ok = c["errorCode"]; ok {
			errCodeExpectedf, _ := v.(float64)
			errCodeExpected = int(errCodeExpectedf)
		}

		if errActual != nil {
			if errCodeExpected == int(errActual.Code()) {
				continue
			}

			if errExpected == "" {
				errstring = go_er.New(fmt.Sprintf("unexpected err: %v\nstatements: %v\n"+
					" for case file: %v, index: %v%s", errActual, statements, ffname, i, findIndex(b, i)))
				return
			}
			if !errActual.ContainsText(errExpected) {
				errstring = go_er.New(fmt.Sprintf("Mismatched error:\nexpected: %s\n  actual: %s\n"+
					" for case file: %v, index: %v%s", errExpected, errActual.Error(), ffname, i, findIndex(b, i)))
				return
			}
			continue
		}

		if errExpected != "" {
			errstring = go_er.New(fmt.Sprintf("did not see the expected err: %v\nstatements: %v\n"+
				" for case file: %v, index: %v%s", errActual, statements, ffname, i, findIndex(b, i)))
			return
		}

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
			okres := doResultsMatch(resultsActual, resultsExpected, fname, i, b, statements)
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
func doResultsMatch(resultsActual, resultsExpected []interface{}, fname string, i int, content []byte, s string) (errstring error) {
	ffname, e := filepath.Abs(fname)
	if e != nil {
		ffname = fname
	}
	if len(resultsActual) != len(resultsExpected) {
		errstring = go_er.New(fmt.Sprintf("results len don't match, %v vs %v\n  actual: %v\nexpected: %v\n"+
			" (%v) for case file: %v, index: %v%s",
			len(resultsActual), len(resultsExpected),
			resultsActual, resultsExpected, s, ffname, i, findIndex(content, i)))
		return
	}

	if !reflect.DeepEqual(resultsActual, resultsExpected) {
		errstring = go_er.New(fmt.Sprintf("results don't match\n  actual: %#v\nexpected: %#v\n"+
			" (%v) for case file: %v, index: %v%s",
			resultsActual, resultsExpected, s, ffname, i, findIndex(content, i)))
		return
	}
	return nil
}

// Search the file content trying to locate the line the index in question starts on
func findIndex(content []byte, index int) string {
	if content == nil {
		return ""
	}
	curIdx := 0
	elementLevel := 0
	line := 1
	quote := byte(0)
	skipNext := false
	for _, b := range content {
		if skipNext {
			skipNext = false
			continue
		}
		if quote != byte(0) {
			if b == byte('\\') {
				skipNext = true
			} else if b == quote {
				quote = byte(0)
			} else if b == byte('\n') {
				line++
			}
		} else if b == byte('"') {
			quote = b
		} else if b == byte('\n') {
			line++
		} else if b == byte('{') {
			if elementLevel == 0 {
				if curIdx == index {
					return fmt.Sprintf(" (line: %d)", line)
				}
				curIdx++
			}
			elementLevel++
		} else if b == byte('}') {
			elementLevel--
		}
	}
	return ""
}
