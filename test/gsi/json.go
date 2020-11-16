//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package gsi

import (
	"encoding/json"
	go_er "errors"
	"fmt"
	"io/ioutil"
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
	"github.com/couchbase/query/functions/constructor"
	"github.com/couchbase/query/logging"
	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
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
var FTS_API_PATH = "api/index/"
var Namespace_CBS = "default"
var Consistency_parameter = datastore.SCAN_PLUS
var curlWhitelist = map[string]interface{}{"all_access": true}
var NodeServices = "pools/default/nodeServices"
var Subpath_advise = []string{"indexes", "covering_indexes"}

func init() {

	Pool_CBS = server.GetIP(true) + ":8091/"

	logger, _ := log_resolver.NewLogger("golog")
	logging.SetLogger(logger)
}

type MockQuery struct {
	server.BaseRequest
	response    *MockResponse
	resultCount int
}

type MockServer struct {
	sync.RWMutex
	prepDone  map[string]bool
	txgroups  []string
	server    *server.Server
	acctstore accounting.AccountingStore
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

func (this *MockQuery) Execute(srvr *server.Server, context *execution.Context, reqType string, signature value.Value) {
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

func (this *scanConfigImpl) SetScanConsistency(consistency datastore.ScanConsistency) interface{} {
	return this
}

func (this *MockServer) doStats(request *MockQuery) {
	request.CompleteRequest(0, 0, request.resultCount, 0, 0, nil, this.server)
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
	case "START":
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
	}, nil}

/*
This method is used to execute the N1QL query represented by
the input argument (q) string using the NewBaseRequest method
as defined in the server request.go.
*/
func Run(mockServer *MockServer, gv int, q, namespace string, namedArgs map[string]value.Value,
	positionalArgs value.Values, userArgs map[string]string) ([]interface{}, []errors.Error, errors.Error) {
	var metrics value.Tristate
	consistency := &scanConfigImpl{scan_level: Consistency_parameter}

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
	query.SetPretty(value.TRUE)
	query.SetScanConfiguration(consistency)
	mockServer.server.SetWhitelist(curlWhitelist)
	query.SetTxId(mockServer.getTxId(gv))

	if userArgs == nil {
		query.SetCredentials(&_ALL_USERS)
	} else {
		users := _ALL_USERS
		for k, v := range userArgs {
			users.Users[k] = v
		}
		query.SetCredentials(&users)
	}
	//	query.BaseRequest.SetIndexApiVersion(datastore.INDEX_API_3)
	//	query.BaseRequest.SetFeatureControls(util.N1QL_GROUPAGG_PUSHDOWN)
	defer mockServer.doStats(query)

	if !mockServer.server.ServiceRequest(query) {
		mockServer.saveTxId(gv, query.Type(), nil)
		return nil, nil, errors.NewError(nil, "Query timed out")
	}

	// wait till all the results are ready
	<-mr.done
	if mr.err != nil {
		mockServer.saveTxId(gv, query.Type(), mr.results)
	}
	return mr.results, mr.warnings, mr.err
}

func RunPrepared(mockServer *MockServer, gv int, q, namespace string, namedArgs map[string]value.Value,
	positionalArgs value.Values) ([]interface{}, []errors.Error, errors.Error) {
	var metrics value.Tristate
	consistency := &scanConfigImpl{scan_level: Consistency_parameter}

	mr := &MockResponse{
		results: []interface{}{}, warnings: []errors.Error{}, done: make(chan bool),
	}
	query := &MockQuery{
		response: mr,
	}

	prepared, err := PrepareStmt(mockServer, gv, namespace, q)
	if err != nil {
		return nil, nil, err
	}

	server.NewBaseRequest(&query.BaseRequest)
	query.SetPrepared(prepared)
	query.SetNamedArgs(namedArgs)
	query.SetPositionalArgs(positionalArgs)
	query.SetNamespace(namespace)
	query.SetReadonly(value.FALSE)
	query.SetMetrics(metrics)
	query.SetSignature(value.TRUE)
	query.SetPretty(value.TRUE)
	query.SetScanConfiguration(consistency)
	query.SetCredentials(&_ALL_USERS)
	query.SetTxId(mockServer.getTxId(gv))

	//	query.BaseRequest.SetIndexApiVersion(datastore.INDEX_API_3)
	//	query.BaseRequest.SetFeatureControls(util.N1QL_GROUPAGG_PUSHDOWN)
	defer mockServer.doStats(query)

	if !mockServer.server.ServiceRequest(query) {
		mockServer.saveTxId(gv, query.Type(), nil)
		return nil, nil, errors.NewError(nil, "Query timed out")
	}

	// wait till all the results are ready
	<-mr.done
	if mr.err != nil {
		mockServer.saveTxId(gv, query.Type(), mr.results)
	}
	return mr.results, mr.warnings, mr.err
}

/*
Used to specify the N1QL nodes options using the method NewServer
as defined in server/server.go.
*/
func Start(site, pool, namespace string, setGlobals bool) *MockServer {

	nullSecurityConfig := &datastore.ConnectionSecurityConfig{}

	mockServer := &MockServer{}
	mockServer.prepDone = make(map[string]bool)
	mockServer.txgroups = make([]string, 16)
	ds, err := resolver.NewDatastore(site + pool)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	ds.SetConnectionSecurityConfig(nullSecurityConfig)

	sys, err := system.NewDatastore(ds)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	ds.SetConnectionSecurityConfig(nullSecurityConfig)

	if setGlobals {
		datastore.SetDatastore(ds)
		datastore.SetSystemstore(sys)
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

	// Start the prepared statement cache
	prepareds.PreparedsInit(1024)

	// Start the completed requests log - keep it small and busy
	server.RequestsInit(0, 8)

	// Start the dictionary cache
	server.InitDictionaryCache(1024)

	srv, err := server.NewServer(ds, sys, configstore, acctstore, namespace,
		false, 10, 10, 1, 1, 1, 0, false, false, true, true,
		server.ProfOff, false)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	server.SetActives(http.NewActiveRequests(srv))
	srv.SetWhitelist(curlWhitelist)

	prepareds.PreparedsReprepareInit(ds, sys)
	constructor.Init(nil)
	srv.SetKeepAlive(1 << 10)

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
		statements := strings.TrimSpace(v.(string))
		// when statement starts with PREPARE or EXECUTE
		// just run the statement as is
		prefix := strings.ToLower(statements[0:8])
		if strings.HasPrefix(prefix, "prepare") || strings.HasPrefix(prefix, "execute") {
			prepared = false
		}

		var ordered bool
		if o, ook := c["ordered"]; ook {
			ordered = o.(bool)
		}

		var gv int
		if txGroup, txOk := c["txGroup"]; txOk {
			gv, _ = txGroup.(int)
		}

		fin_stmt = strconv.Itoa(i) + ": " + statements
		var resultsActual []interface{}
		var errActual errors.Error
		var namedArgs map[string]value.Value
		var positionalArgs value.Values
		var userArgs map[string]string
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
			if errstring = checkExplain(qc, gv, namespace, statements, c, ordered, namedArgs, positionalArgs, fname, i); errstring != nil {
				return
			}
		}

		if prepared {
			resultsActual, _, errActual = RunPrepared(qc, gv, statements, namespace, namedArgs, positionalArgs)
		} else {
			resultsActual, _, errActual = Run(qc, gv, statements, namespace, namedArgs, positionalArgs, userArgs)
		}

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

			if errExpected != errActual.Error() {
				errstring = go_er.New(fmt.Sprintf("Mismatched error - expected '%s' actual '%s'"+
					", for case file: %v, index: %v", errExpected, errActual.Error(), fname, i))
				return
			}

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
				for j, _ := range resultsActual {
					newResults[j] = make(map[string]interface{}, len(accept.([]interface{})))
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
				for j, _ := range resultsActual {
					newResults[j] = make(map[string]interface{}, 1)
				}
				addResultsEntry(newResults, resultsActual, accept)
			}
			resultsActual = newResults
		}
		v, ok = c["results"]
		if ok {
			if isAdvise, ok := c["advise"]; ok {
				if isAdvise, ok := isAdvise.(bool); ok && isAdvise {
					resultsExpected := v.([]interface{})
					for _, sub := range Subpath_advise {
						okres := doResultsMatch(getAdviseResults(sub, resultsActual), getAdviseResults(sub, resultsExpected), ordered, statements, fname, i)
						if okres != nil {
							errstring = okres
							return
						}
					}
				}
			} else {
				resultsExpected := v.([]interface{})
				okres := doResultsMatch(resultsActual, resultsExpected, ordered, statements, fname, i)
				if okres != nil {
					errstring = okres
					return
				}
			}
		}
	}
	return fin_stmt, nil
}

/*
Matches expected results with the results obtained by
running the queries.
*/
func doResultsMatch(resultsActual, resultsExpected []interface{}, ordered bool, stmt, fname string, i int) (errstring error) {
	if len(resultsActual) != len(resultsExpected) {
		return go_er.New(fmt.Sprintf("results len don't match, %v vs %v, %v vs %v"+
			", (%v)for case file: %v, index: %v",
			len(resultsActual), len(resultsExpected),
			resultsActual, resultsExpected, stmt, fname, i))
	}

	if ordered {
		if !reflect.DeepEqual(resultsActual, resultsExpected) {
			return go_er.New(fmt.Sprintf("results don't match, actual: %#v, expected: %#v"+
				", (%v) for case file: %v, index: %v",
				resultsActual, resultsExpected, stmt, fname, i))
		}
	} else {
	nextresult:
		for _, re := range resultsExpected {
			for j, ra := range resultsActual {
				if ra != nil && reflect.DeepEqual(ra, re) {
					resultsActual[j] = nil
					continue nextresult
				}
			}
			return go_er.New(fmt.Sprintf("results don't match: %#v is not present in : %#v"+
				", (%v) for case file: %v, index: %v",
				re, resultsActual, stmt, fname, i))
		}

	}

	return nil
}

func checkExplain(qc *MockServer, gv int, namespace string, statement string, c map[string]interface{}, ordered bool,
	namedArgs map[string]value.Value, positionalArgs value.Values, fname string, i int) (errstring error) {
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

	explainStmt := "EXPLAIN " + statement
	resultsActual, _, errActual := Run(qc, gv, explainStmt, namespace, namedArgs, positionalArgs, nil)
	if errActual != nil || len(resultsActual) != 1 {
		return go_er.New(fmt.Sprintf("(%v) error actual: code - %d, msg - %s"+
			", for case file: %v, index: %v", explainStmt, errActual.Code(), errActual.Error(), fname, i))
	}

	namedParams := make(map[string]value.Value, 1)
	namedParams["explan"] = value.NewValue(resultsActual[0])

	resultsActual, _, errActual = Run(qc, gv, eStmt, namespace, namedParams, nil, nil)
	if errActual != nil {
		return go_er.New(fmt.Sprintf("unexpected err: code - %d, msg - %s, statement: %v"+
			", for case file: %v, index: %v", errActual.Code(), errActual.Error(), eStmt, fname, i))
	}

	if rok {
		return doResultsMatch(resultsActual, erExpected, ordered, eStmt, fname, i)
	}

	return
}

func PrepareStmt(qc *MockServer, gv int, namespace, statement string) (*plan.Prepared, errors.Error) {
	prepareStmt := "PREPARE " + statement
	resultsActual, _, errActual := Run(qc, gv, prepareStmt, namespace, nil, nil, nil)
	if errActual != nil || len(resultsActual) != 1 {
		return nil, errors.NewError(nil, fmt.Sprintf("Error %#v FOR (%v)", prepareStmt, resultsActual))
	}
	ra := resultsActual[0].(map[string]interface{})

	// if already tried decodeing just get on with it
	qc.RLock()
	done := qc.prepDone[statement]
	qc.RUnlock()
	if done {
		return prepareds.GetPrepared(ra["name"].(string), nil)
	}

	// we redecode the encoded plan to make sure that we can transmit it correctly across nodes
	rv, err := prepareds.DecodePrepared(ra["name"].(string), ra["encoded_plan"].(string))
	if err != nil {
		return rv, err
	}
	qc.Lock()
	qc.prepDone[statement] = true
	qc.Unlock()
	return rv, err
}

/*
Method to pass in parameters for site, pool and
namespace to Start method for Couchbase Server.
*/

func Start_cs(setGlobals bool) *MockServer {
	ms := Start(Site_CBS, Auth_param+"@"+Pool_CBS, Namespace_CBS, setGlobals)

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

func RunStmt(mockServer *MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return Run(mockServer, 0, q, Namespace_CBS, nil, nil, nil)
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
