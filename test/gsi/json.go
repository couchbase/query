//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package gsi

import (
	"encoding/json"
	go_er "errors"
	"fmt"
	"io/ioutil"
	go_http "net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/couchbase/query/accounting"
	acct_resolver "github.com/couchbase/query/accounting/resolver"
	"github.com/couchbase/query/auth"
	config_resolver "github.com/couchbase/query/clustering/resolver"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/constructor"
	"github.com/couchbase/query/functions/storage"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/server/http/router"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
	diffpkg "github.com/kylelemons/godebug/diff"
)

/*
Global variables accessed by individual test cases for
Couchbase server. Site_CBS, Auth_param, Pool_CBS
and Namespace_CBS represent the site, server authentication
parameters the ip of the couchbase server instance
and the namespace.
*/
var Site_CBS = "http://"
var Username = "Administrator"
var Password = "password"
var Auth_param = "Administrator:password"
var Pool_CBS = "127.0.0.1:8091/"
var FTS_CBS = "127.0.0.1:8094/"
var Query_CBS = "127.0.0.1:8093"
var FTS_API_PATH = "api/index/"
var Namespace_CBS = "default"
var Consistency_parameter = datastore.SCAN_PLUS
var curlAllowlist = map[string]interface{}{"all_access": true}
var NodeServices = "pools/default/nodeServices"
var Subpath_advise = []string{"indexes", "covering_indexes"}

func init() {

	Pool_CBS = server.GetIP(true) + ":8091/"

	logger, _ := log_resolver.NewLogger("golog")
	logging.SetLogger(logger)
}

type RunResult struct {
	Results   []interface{}
	Warnings  []errors.Error
	Err       errors.Error
	SortCount int
}

type MockQuery struct {
	server.BaseRequest
	response    *MockResponse
	resultCount int
}

type MockServer struct {
	sync.RWMutex
	prepDone   map[string]bool
	txgroups   []string
	server     *server.Server
	acctstore  accounting.AccountingStore
	httpServer go_http.Server
}

func (this *MockServer) ShutdownHttpServer() {
	this.httpServer.Close()
}

func SetConsistencyParam(consistency_parameter datastore.ScanConsistency) {
	Consistency_parameter = consistency_parameter
}

func (this *MockQuery) Output() execution.Output {
	return this
}

func (this *MockQuery) Fail(err errors.Error) {
	defer this.Stop(server.FATAL)
	this.response.err = err
	close(this.response.done)
}

func (this *MockQuery) Error(err errors.Error) {
	if this.response.err == nil {
		this.response.err = err
	}
}

func (this *MockQuery) SetErrors(errs errors.Errors) {
	for _, err := range errs {
		this.Error(err)
	}
}

func (this *MockQuery) Execute(srvr *server.Server, context *execution.Context, reqType string, signature value.Value, startTx bool) {
	select {
	case <-this.Results():
		this.Stop(server.COMPLETED)
	case <-this.StopExecute():
		this.Stop(server.STOPPED)

		// wait for operator before continuing
		<-this.Results()
	}

	if err, _ := context.DoStatementComplete(reqType, this.response.err == nil); err != nil {
		if this.response.err == nil {
			this.response.err = err
		}
	} else if context.TxContext() != nil && startTx {
		this.SetTransactionStartTime(context.TxContext().TxStartTime())
		this.SetTxTimeout(context.TxContext().TxTimeout())
	}
	close(this.response.done)
	this.response.sortCount = int(this.SortCount())
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

func (this *MockQuery) Alive() bool {
	return true
}

func (this *MockQuery) Result(item value.AnnotatedValue) bool {
	bytes, err := json.Marshal(item)
	if err != nil {
		panic(err.Error())
	}

	this.resultCount++

	var resultLine interface{}

	json.Unmarshal(bytes, &resultLine)

	this.response.results = append(this.response.results, resultLine)
	return true
}

func (this *MockQuery) Loga(l logging.Level, f func() string) {
}

func (this *MockQuery) LogLevel() logging.Level {
	return logging.NONE
}

type MockResponse struct {
	err       errors.Error
	results   []interface{}
	warnings  []errors.Error
	done      chan bool
	sortCount int
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

func (this *scanConfigImpl) SetScanConsistency(consistency datastore.ScanConsistency) interface{} {
	return this
}

func (this *MockServer) doStats(request *MockQuery) {
	request.CompleteRequest(0, 0, 0, request.resultCount, 0, 0, nil, this.server, 0)
}

func (this *MockServer) getTxId(group int) string {
	this.RLock()
	defer this.RUnlock()
	if group < len(this.txgroups) {
		return this.txgroups[group]
	}
	return ""
}

func (this *MockServer) setTxId(group int, txId string) {
	this.Lock()
	defer this.Unlock()
	if group < len(this.txgroups) {
		this.txgroups[group] = txId
	}
}

func (this *MockServer) saveTxId(group int, reqType string, res []interface{}) {
	var txId string
	switch reqType {
	case "START_TRANSACTION":
		if len(res) > 0 {
			if fields, ok := res[0].(map[string]interface{}); ok {
				txId, _ = fields["txid"].(string)
			}
		}
		this.setTxId(group, txId)
	case "ROLLBACK", "COMMIT":
		this.setTxId(group, txId)
	}
}

var _ALL_USERS = auth.Credentials{
	map[string]string{
		"customerowner":  "customerpass",
		"ordersowner":    "orderspass",
		"productowner":   "productpass",
		"purchaseowner":  "purchasepass",
		"reviewowner":    "reviewpass",
		"shellTestowner": "shellTestpass",
	}, nil, nil, nil}

/*
This method is used to execute the N1QL query represented by
the input argument (q) string using the NewBaseRequest method
as defined in the server request.go.
*/
func Run(mockServer *MockServer, queryParams map[string]interface{}, q, namespace string, namedArgs map[string]value.Value,
	positionalArgs value.Values, userArgs map[string]string) *RunResult {
	return run(mockServer, queryParams, q, namespace, namedArgs, positionalArgs, userArgs, false)
}

func RunPrepared(mockServer *MockServer, queryParams map[string]interface{}, q, namespace string, namedArgs map[string]value.Value,
	positionalArgs value.Values) *RunResult {
	return run(mockServer, queryParams, q, namespace, namedArgs, positionalArgs, nil, true)
}

func run(mockServer *MockServer, queryParams map[string]interface{}, q, namespace string, namedArgs map[string]value.Value,
	positionalArgs value.Values, userArgs map[string]string, prepare bool) *RunResult {

	var metrics value.Tristate
	consistency := &scanConfigImpl{scan_level: Consistency_parameter}

	mr := &MockResponse{
		results: []interface{}{}, warnings: []errors.Error{}, done: make(chan bool),
	}
	query := &MockQuery{
		response: mr,
	}

	server.NewBaseRequest(&query.BaseRequest)

	if prepare {
		prepared, err := PrepareStmt(mockServer, queryParams, namespace, q)
		if err != nil {
			return &RunResult{nil, nil, err, -1}
		}
		query.SetPrepared(prepared)
		query.SetType(prepared.Type())
	} else {
		query.SetStatement(q)
	}
	query.SetNamedArgs(namedArgs)
	query.SetPositionalArgs(positionalArgs)
	query.SetNamespace(namespace)
	query.SetReadonly(value.FALSE)
	query.SetMetrics(metrics)
	query.SetSignature(value.TRUE)
	query.SetPretty(value.TRUE)
	query.SetScanConfiguration(consistency)
	query.SetUseCBO(false)
	mockServer.server.SetAllowlist(curlAllowlist)

	var gv int
	if txGroup, txOk := queryParams["txgroup"]; txOk {
		gvf, _ := txGroup.(float64)
		gv = int(gvf)
	}
	query.SetTxId(mockServer.getTxId(gv))

	if s, ok := queryParams["durability_level"]; ok {
		if dl, ok := s.(string); ok && dl != "" {
			query.SetDurabilityLevel(datastore.DurabilityNameToLevel(dl))
		}
	}

	if s, ok := queryParams["kvtimeout"]; ok {
		if sk, ok := s.(string); ok && sk != "" {
			if kvTimeout, e := time.ParseDuration(sk); e != nil {
				query.SetKvTimeout(kvTimeout)
			}
		}
	}

	if s, ok := queryParams["preserve_expiry"]; ok {
		if b, ok := s.(bool); ok && b {
			query.SetPreserveExpiry(b)
		}
	}

	if txImplict, ok := queryParams["tximplicit"]; ok {
		if b, ok := txImplict.(bool); ok && b {
			query.SetTxImplicit(b)
		}
	}

	if s, ok := queryParams["query_context"]; ok {
		if qcs, ok := s.(string); ok && qcs != "" {
			query.SetQueryContext(qcs)
		}
	}

	if s, ok := queryParams["use_cbo"]; ok {
		if b, ok := s.(bool); ok && b {
			query.SetUseCBO(b)
		}
	}

	if userArgs == nil {
		query.SetCredentials(&_ALL_USERS)
	} else {
		users := auth.NewCredentials()
		for k, v := range _ALL_USERS.Users {
			users.Users[k] = v
		}
		for k, v := range userArgs {
			users.Users[k] = v
		}
		query.SetCredentials(users)
	}
	//	query.BaseRequest.SetIndexApiVersion(datastore.INDEX_API_3)
	//	query.BaseRequest.SetFeatureControls(util.N1QL_GROUPAGG_PUSHDOWN)
	defer mockServer.doStats(query)

	var ret bool
	if query.TxId() != "" {
		ret = mockServer.server.PlusServiceRequest(query)
	} else {
		ret = mockServer.server.ServiceRequest(query)
	}

	if !ret {
		mockServer.saveTxId(gv, query.Type(), nil)
		return &RunResult{nil, nil, errors.NewError(nil, "Query timed out"), -1}
	}

	// wait till all the results are ready
	<-mr.done
	mockServer.saveTxId(gv, query.Type(), mr.results)
	return &RunResult{mr.results, query.Warnings(), mr.err, mr.sortCount}
}

/*
Used to specify the N1QL nodes options using the method NewServer
as defined in server/server.go.

startHttpServer: Whether an HTTP Server must be created and started
*/
func Start(site, pool, namespace string, setGlobals, startHttpServer bool) *MockServer {

	nullSecurityConfig := &datastore.ConnectionSecurityConfig{}

	mockServer := &MockServer{}
	mockServer.prepDone = make(map[string]bool)
	mockServer.txgroups = make([]string, 16)
	ds, err := resolver.NewDatastore(site + pool)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}
	ds.SetConnectionSecurityConfig(nullSecurityConfig)

	acctstore, err := acct_resolver.NewAcctstore("stub:")
	if err != nil {
		logging.Errorf("Could not connect to acctstore: %v", err)
	}

	sys, err := system.NewDatastore(ds, acctstore)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}
	ds.SetConnectionSecurityConfig(nullSecurityConfig)

	if setGlobals {
		datastore.SetDatastore(ds)
		datastore.SetSystemstore(sys)
	}

	configstore, err := config_resolver.NewConfigstore("stub:", "")
	if err != nil {
		logging.Errorf("Could not connect to configstore: %v", err)
	}

	// Start the prepared statement cache
	prepareds.PreparedsInit(1024)

	// Start the UDF cache
	functions.FunctionsInit(1024)

	// Start the completed requests log - keep it small and busy
	server.RequestsInit(0, 8, 10)

	// Start the dictionary cache
	server.InitDictionaryCache(1024)

	srv, err := server.NewServer(ds, sys, configstore, acctstore, namespace,
		false, 10, 10, 1, 1, 16, 0, false, false, true, true,
		server.ProfOff, false)
	if err != nil {
		logging.Errorf("%v", err.Error())
		os.Exit(1)
	}

	server.SetActives(http.NewActiveRequests(srv))
	srv.SetAllowlist(curlAllowlist)

	prepareds.PreparedsReprepareInit(ds, sys)

	jsevalPath := ""
	if path := os.Getenv("JSEVALUATOR_PATH"); path != "" {
		jsevalPath = path
	}

	// create an HTTP server
	if startHttpServer {
		router := router.NewRouter()
		httpSrv := &go_http.Server{
			Handler:           router,
			Addr:              Query_CBS,
			ReadHeaderTimeout: 5 * time.Second,
		}

		constructor.Init(router, 6, jsevalPath)
		go httpSrv.ListenAndServe()

	} else {
		constructor.Init(nil, 6, jsevalPath)
	}

	srv.SetKeepAlive(1 << 10)

	storage.Migrate()

	mockServer.server = srv
	mockServer.acctstore = acctstore

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

func FtestCaseFile(fname string, prepared, explain bool, qc *MockServer, namespace string) (fin_stmt string, errstring error) {
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

		/* Handles all queries to be run against CBServer and Datastore */
		v, ok := c["statements"]
		if !ok || v == nil {
			errstring = go_er.New(fmt.Sprintf("missing statements for case file: %v, index: %v%s", ffname, i, findIndex(b, i)))
			return
		}
		statements := strings.TrimSpace(v.(string))
		// when statement starts with PREPARE or EXECUTE
		// just run the statement as is
		slen := len(statements)
		if slen > 8 {
			slen = 8
		}
		prefix := strings.ToLower(statements[0:slen])
		if strings.HasPrefix(prefix, "prepare") || strings.HasPrefix(prefix, "execute") {
			prepared = false
		}

		var ordered bool
		if o, ook := c["ordered"]; ook {
			ordered = o.(bool)
		}

		fin_stmt = strconv.Itoa(i) + ": " + statements
		var namedArgs map[string]value.Value
		var positionalArgs value.Values
		var userArgs map[string]string
		var queryParams map[string]interface{}
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
		if qp, ok := c["queryParams"]; ok {
			queryParams, _ = qp.(map[string]interface{})
		}
		if u, ok_u := c["userArgs"]; ok_u {
			uv, ok_uv := value.NewValue(u).Actual().(map[string]interface{})
			if ok_uv {
				userArgs = make(map[string]string, len(uv))
				for user, password := range uv {
					userArgs[user] = password.(string)
				}
			}
		}

		if explain {
			if errstring = checkExplain(qc, queryParams, namespace, statements, c, ordered, namedArgs, positionalArgs, fname, i, b); errstring != nil {
				return
			}
		}

		errCodeExpected := int(0)
		if v, ok = c["errorCode"]; ok {
			errCodeExpectedf, _ := v.(float64)
			errCodeExpected = int(errCodeExpectedf)
		}

		warnCodeExpected := int(0)
		if v, ok = c["warningCode"]; ok {
			warnCodeExpectedf, _ := v.(float64)
			warnCodeExpected = int(warnCodeExpectedf)
		}

		// no index, test infrastructure can't handle this.
		var rr *RunResult
		if prepared && errCodeExpected != 4000 {
			rr = RunPrepared(qc, queryParams, statements, namespace, namedArgs, positionalArgs)
		} else {
			rr = Run(qc, queryParams, statements, namespace, namedArgs, positionalArgs, userArgs)
		}

		errExpected := ""
		v, ok = c["error"]
		if ok {
			errExpected = v.(string)
		}

		warnExpected := ""
		v, ok = c["warning"]
		if ok {
			warnExpected = v.(string)
		}

		if rr.Err != nil {
			if errCodeExpected == int(rr.Err.Code()) {
				continue
			}

			if errExpected == "" {
				errstring = go_er.New(fmt.Sprintf("unexpected err: %v\nstatements: %v\n"+
					"      file: %v\n     index: %v%s\n\n", rr.Err, statements, ffname, i, findIndex(b, i)))
				return
			}

			if !rr.Err.ContainsText(errExpected) {
				errstring = go_er.New(fmt.Sprintf("Mismatched error:\nexpected: %s\n  actual: %s\n"+
					"      file: %v\n     index: %v%s\n\n", errExpected, rr.Err.Error(), ffname, i, findIndex(b, i)))
				return
			}

			continue
		}
		if errExpected != "" {
			errstring = go_er.New(fmt.Sprintf("did not see the expected err: %v\nstatements: %v\n"+
				"      file: %v\n     index: %v%s\n\n", errExpected, statements, ffname, i, findIndex(b, i)))
			return
		}
		if errCodeExpected != 0 {
			errstring = go_er.New(fmt.Sprintf("did not see the expected err: %v\nstatements: %v\n"+
				"      file: %v\n     index: %v%s\n\n", errCodeExpected, statements, ffname, i, findIndex(b, i)))
			return
		}

		if len(rr.Warnings) > 0 {
			if warnExpected == "" && warnCodeExpected == 0 {
				errstring = go_er.New(fmt.Sprintf("unexpected warning(s):\n%s\nstatements: %v\n"+
					"      file: %v\n     index: %v%s\n\n", prettyPrint(rr.Warnings), statements, ffname, i, findIndex(b, i)))
				return
			}

			found := false
			for _, w := range rr.Warnings {
				if int(w.Code()) == warnCodeExpected || (len(warnExpected) > 0 && !w.ContainsText(warnExpected)) {
					found = true
				}
			}

			if !found {
				if warnExpected != "" {
					errstring = go_er.New(fmt.Sprintf("Missing expected warning: %s\nstatements: %v\n"+
						"      file: %v\n     index: %v%s\n\n", warnExpected, statements, ffname, i, findIndex(b, i)))
				} else {
					errstring = go_er.New(fmt.Sprintf("Missing expected warning: %v\nstatements: %v\n"+
						"      file: %v\n     index: %v%s\n\n", warnCodeExpected, statements, ffname, i, findIndex(b, i)))
				}
				return
			}

			continue
		}
		if warnExpected != "" {
			errstring = go_er.New(fmt.Sprintf("did not see the expected warning: %v\nstatements: %v\n"+
				"      file: %v\n     index: %v%s\n\n", warnExpected, statements, ffname, i, findIndex(b, i)))
			return
		}
		if warnCodeExpected != 0 {
			errstring = go_er.New(fmt.Sprintf("did not see the expected warning: %v\nstatements: %v\n"+
				"      file: %v\n     index: %v%s\n\n", warnCodeExpected, statements, ffname, i, findIndex(b, i)))
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
						dropResultsEntry(rr.Results, v)
					}
				}
			case map[string]interface{}:
			default:
				dropResultsEntry(rr.Results, ignore)
			}
		}

		// opposite of ignore - only select certain fields
		// again, we handle scalars and the scalars in an array
		accept, ok := c["accept"]
		if ok {
			newResults := make([]interface{}, len(rr.Results))
			switch accept.(type) {
			case []interface{}:
				for j, _ := range rr.Results {
					newResults[j] = make(map[string]interface{}, len(accept.([]interface{})))
				}
				for _, v := range accept.([]interface{}) {
					switch v.(type) {
					case []interface{}:
					case map[string]interface{}:
					default:
						addResultsEntry(newResults, rr.Results, v)
					}
				}
			case map[string]interface{}:
			default:
				for j, _ := range rr.Results {
					newResults[j] = make(map[string]interface{}, 1)
				}
				addResultsEntry(newResults, rr.Results, accept)
			}
			rr.Results = newResults
		}
		v, ok = c["results"]
		if ok {
			if isAdvise, ok := c["advise"]; ok {
				if isAdvise, ok := isAdvise.(bool); ok && isAdvise {
					resultsExpected := v.([]interface{})
					for _, sub := range Subpath_advise {
						okres := doResultsMatch(getAdviseResults(sub, rr.Results), getAdviseResults(sub, resultsExpected), ordered, statements, fname, i, b)
						if okres != nil {
							errstring = okres
							return
						}
					}
				}
			} else {
				resultsExpected := v.([]interface{})
				okres := doResultsMatch(rr.Results, resultsExpected, ordered, statements, fname, i, b)
				if okres != nil {
					errstring = okres
					return
				}
			}
		}
		v, ok = c["rr.SortCount"]
		if ok {
			expectedSortCount := int(v.(float64))
			if rr.SortCount != expectedSortCount {
				errstring = go_er.New(fmt.Sprintf("rr.SortCount %v doesn't match expected %v\nstatement: %v\n"+
					"     file: %v\n    index: %v%s\n\n", rr.SortCount, expectedSortCount, statements, ffname, i, findIndex(b, i)))
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
func doResultsMatch(resultsActual, resultsExpected []interface{}, ordered bool, stmt, fname string, i int, content []byte) (errstring error) {
	ffname, e := filepath.Abs(fname)
	if e != nil {
		ffname = fname
	}
	if len(resultsActual) != len(resultsExpected) {
		return go_er.New(fmt.Sprintf("results length doesn't match; expected %v have %v\n%v\n"+
			"statement: %v\n     file: %v\n    index: %v%s\n\n",
			len(resultsExpected), len(resultsActual),
			diff(resultsExpected, resultsActual), stmt, ffname, i, findIndex(content, i)))
	}

	if ordered {
		if !reflect.DeepEqual(resultsActual, resultsExpected) {
			return go_er.New(fmt.Sprintf("results don't match\n%v\n"+
				"statement: %v\n     file: %v\n    index: %v%s\n\n",
				diff(resultsExpected, resultsActual), stmt, ffname, i, findIndex(content, i)))
		}
	} else {
	nextresult:
		for _, re := range resultsExpected {
			for j, ra := range resultsActual {
				if ra != nil && reflect.DeepEqual(ra, re) {
					if j < len(resultsActual)-1 {
						copy(resultsActual[j:], resultsActual[j+1:])
					}
					resultsActual = resultsActual[:len(resultsActual)-1]
					continue nextresult
				}
			}
			return go_er.New(fmt.Sprintf("results don't match\n%v\nis not present in remaining results:\n%v\n"+
				"statement: %v\n     file: %v\n    index: %v%s\n\n",
				prettyPrint(re), prettyPrint(resultsActual), stmt, ffname, i, findIndex(content, i)))
		}

	}

	return nil
}

func diff(a interface{}, b interface{}) string {
	return diffpkg.Diff(prettyPrint(a), prettyPrint(b))
}

func prettyPrint(what interface{}) string {
	res, err := json.MarshalIndent(what, "", "  ")
	if err != nil {
		res = []byte(fmt.Sprintf("%v", what))
	}
	return string(res)
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

func checkExplain(qc *MockServer, queryParams map[string]interface{}, namespace string, statement string, c map[string]interface{},
	ordered bool, namedArgs map[string]value.Value, positionalArgs value.Values, fname string, i int, content []byte) (errstring error) {
	var ev map[string]interface{}

	e, ok := c["explain"]
	if ok {
		ev, ok = e.(map[string]interface{})
	}

	if !ok {
		return
	}

	var eStmt string
	var erExpected []interface{}

	ed, dok := ev["disabled"]
	es, sok := ev["statement"]
	er, rok := ev["results"]
	ea, aok := ev["use_args"]

	if dok {
		if disabled := ed.(bool); disabled {
			return
		}
	}

	if sok {
		eStmt, sok = es.(string)
	}

	if !sok {
		return
	}

	if rok {
		erExpected, rok = er.([]interface{})
	}

	useArgs := false
	if aok {
		var bv bool
		if bv, aok = ea.(bool); aok {
			useArgs = bv
		}
	}
	if !useArgs {
		namedArgs = nil
		positionalArgs = nil
	}

	explainStmt := "EXPLAIN " + statement
	rr := Run(qc, queryParams, explainStmt, namespace, namedArgs, positionalArgs, nil)
	if rr.Err != nil {
		return go_er.New(fmt.Sprintf("(%v) error actual: code - %d, msg - %s\n"+
			"      file: %v\n     index: %v%s", explainStmt, rr.Err.Code(), rr.Err.Error(), fname, i, findIndex(content, i)))
	} else if len(rr.Results) != 1 {
		return go_er.New(fmt.Sprintf("(%v) unexpected number of results returned (%d)\n"+
			"      file: %v\n     index: %v%s", explainStmt, len(rr.Results), fname, i, findIndex(content, i)))
	}

	namedParams := make(map[string]value.Value, 1)
	namedParams["explan"] = value.NewValue(rr.Results[0])

	rr = Run(qc, queryParams, eStmt, namespace, namedParams, nil, nil)
	if rr.Err != nil {
		return go_er.New(fmt.Sprintf("unexpected err: code - %d, msg - %s\nstatement: %v\n"+
			"     file: %v\n    index: %v%s", rr.Err.Code(), rr.Err.Error(), eStmt, fname, i, findIndex(content, i)))
	}

	if rok {
		return doResultsMatch(rr.Results, erExpected, ordered, eStmt, fname, i, content)
	}

	return
}

func PrepareStmt(qc *MockServer, queryParams map[string]interface{}, namespace, statement string) (*plan.Prepared, errors.Error) {
	var queryContext string
	if s, ok := queryParams["query_context"]; ok {
		queryContext, _ = s.(string)
	}
	prepareStmt := "PREPARE " + statement
	rr := Run(qc, queryParams, prepareStmt, namespace, nil, nil, nil)
	if rr.Err != nil || len(rr.Results) != 1 {
		return nil, errors.NewError(nil, fmt.Sprintf("Error %#v FOR (%v)", prepareStmt, rr.Results))
	}
	ra := rr.Results[0].(map[string]interface{})

	// if already tried decodeing just get on with it
	qc.RLock()
	done := qc.prepDone[statement]
	qc.RUnlock()
	if done {
		return prepareds.GetPreparedWithContext(ra["name"].(string), queryContext, make(map[string]bool, 1), 0, nil,
			logging.NULL_LOG)
	}

	// we redecode the encoded plan to make sure that we can transmit it correctly across nodes
	rv, err := prepareds.DecodePreparedWithContext(ra["name"].(string), queryContext, ra["encoded_plan"].(string),
		false, nil, false, logging.NULL_LOG)
	if err != nil {
		return rv, err
	}
	qc.Lock()
	qc.prepDone[statement] = true
	qc.Unlock()
	return prepareds.GetPreparedWithContext(ra["name"].(string), queryContext, make(map[string]bool, 1), 0, nil, logging.NULL_LOG)
}

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.

Use Start_cs() if the test does not require an HTTP Server
*/

func Start_cs(setGlobals bool) *MockServer {
	ms := Start(Site_CBS, Auth_param+"@"+Pool_CBS, Namespace_CBS, setGlobals, false)

	return ms
}

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.

Use Start_cs_http() if the test requires an HTTP server to be started
*/
func Start_cs_http(setGlobals bool) *MockServer {
	ms := Start(Site_CBS, Auth_param+"@"+Pool_CBS, Namespace_CBS, setGlobals, true)

	return ms
}

func RunMatch(filename string, prepared, explain bool, qc *MockServer, t *testing.T) {
	util.SetN1qlFeatureControl(util.GetN1qlFeatureControl() & ^util.N1QL_ENCODED_PLAN)
	// Start the completed requests log - keep it small and busy

	util.SetN1qlFeatureControl(util.GetN1qlFeatureControl() & ^util.N1QL_ENCODED_PLAN)
	matches, err := filepath.Glob(filename)
	if err != nil {
		t.Errorf("glob failed: %v", err)
	}

	for _, m := range matches {
		t.Logf("TestCaseFile: %v\n", m)
		stmt, errcs := FtestCaseFile(m, prepared, explain, qc, Namespace_CBS)

		if errcs != nil {
			t.Errorf("Error : %s", errcs.Error())
			return
		}

		if stmt != "" {
			t.Logf(" %v\n", stmt)
		}

		fmt.Print("\nQuery : ", m, "\n\n")
	}

}

func RunStmt(mockServer *MockServer, q string) *RunResult {
	return Run(mockServer, nil, q, Namespace_CBS, nil, nil, nil)
}

func getAdviseResults(subpath string, result []interface{}) []interface{} {
	for _, v := range result {
		v, ok := value.NewValue(v).Actual().(map[string]interface{})
		if !ok {
			continue
		}
		v1, ok := v["advice"]
		if !ok {
			continue
		}
		v1a, ok := value.NewValue(v1).Actual().(map[string]interface{})
		if !ok {
			continue
		}
		v2, ok := v1a["adviseinfo"]
		if !ok {
			continue
		}
		v2a, ok := value.NewValue(v2).Actual().(map[string]interface{})
		if !ok {
			continue
		}
		v3, ok := v2a["recommended_indexes"]
		if !ok {
			continue
		}
		v3a, ok := value.NewValue(v3).Actual().(map[string]interface{})
		if !ok {
			continue
		}
		v4, ok := v3a[subpath]
		if ok {
			return value.NewValue(v4).Actual().([]interface{})
		}
		//for k4, v4 := range v3a {
		//	if k4 == subpath {
		//		return value.NewValue(v4).Actual().([]interface{})
		//	}
		//}

		//for _, v3 := range v2a {
		//	v3, ok := value.NewValue(v3).Actual().(map[string]interface{})
		//	if !ok {
		//		continue
		//	}
		//	for k4, v4 := range v3 {
		//		if k4 == "recommended_indexes" {
		//			v4, ok := value.NewValue(v4).Actual().(map[string]interface{})
		//			if !ok {
		//				continue
		//			}
		//			for k5, v5 := range v4 {
		//				if k5 == subpath {
		//					return value.NewValue(v5).Actual().([]interface{})
		//				}
		//			}
		//		}
		//	}
		//}
	}
	return nil
}
