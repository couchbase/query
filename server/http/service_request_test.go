//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/go_json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/datastore/system"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/timestamp"

	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/server"
)

type testServer struct {
	http_server   *httptest.Server
	query_server  *server.Server
	query_request *httpRequest
}

func newTestServer() *testServer {
	var rv testServer
	rv.query_server = makeMockServer()
	rv.http_server = httptest.NewServer(rv.testHandler())
	return &rv
}

func (this *testServer) URL() string {
	return this.http_server.URL
}

func (this *testServer) request() *httpRequest {
	return this.query_request
}

var test_server *testServer

func init() {
	logger, _ := log_resolver.NewLogger("golog")
	if logger == nil {
		fmt.Printf("Unable to create logger")
		os.Exit(1)
	}
	logging.SetLogger(logger)
	server.RequestsInit(1000, 4000, 10000)
	prepareds.PreparedsInit(1024)
	test_server = newTestServer()
	prepareds.PreparedsReprepareInit(test_server.query_server.Datastore(), test_server.query_server.Systemstore())
}

func verifyEntry(t *testing.T, e timestamp.Entry, position uint32, guard string, value uint64) {
	if e.Position() != position {
		t.Errorf("Bad position, expected %d actual %d", position, e.Position())
	}
	if e.Guard() != guard {
		t.Errorf("Bad guard, expected %s actual %s", guard, e.Guard())
	}
	if e.Value() != value {
		t.Errorf("Bad value, expecte %d actual %d", value, e.Value())
	}
}

func TestGetScanVectors(t *testing.T) {
	jsonText := ` { 
		"bucketb": [FULL_SCAN_VECTOR],
		"default": {
			"23": [9012344, "AUUID"],
			"45": [7455623, "BUUID"]
		},
		"bucketa": {
			"1000": [1234567, "CUUID"]
		}
	}`
	fullScanElement := "[90909, \"DUUID\"],"
	replacement := strings.TrimRight(strings.Repeat(fullScanElement, 1024), ",")
	jsonText = strings.Replace(jsonText, "FULL_SCAN_VECTOR", replacement, 1)

	decoder := json.NewDecoder(strings.NewReader(jsonText))
	var target interface{}
	e := decoder.Decode(&target)
	if e != nil {
		t.Errorf("Unexpected JSON parsing error %v", e)
	}
	var bucketVectorMap map[string]timestamp.Vector
	bucketVectorMap, err := getScanVectorsFromJSON(target)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if len(bucketVectorMap) != 3 {
		t.Errorf("Expected map size 3, actual %d", len(bucketVectorMap))
	}

	// Verify bucket "bucketb".
	vector, ok := bucketVectorMap["bucketb"]
	if !ok {
		t.Errorf("Could not find element bucketb")
	}
	entries := vector.Entries()
	if len(entries) != 1024 {
		t.Errorf("Expected bucketb entries lenth 1024, actual %d", len(entries))
	}
	for i, entry := range entries {
		verifyEntry(t, entry, uint32(i), "DUUID", 90909)
	}

	// Verify bucket "default".
	vector, ok = bucketVectorMap["default"]
	if !ok {
		t.Errorf("Could not find element default.")
	}
	entries = vector.Entries()
	if len(entries) != 2 {
		t.Errorf("Expected default entries length 2, actual %d", len(entries))
	}
	var entry23 timestamp.Entry
	var entry45 timestamp.Entry

	if entries[0].Position() == 23 {
		entry23 = entries[0]
		entry45 = entries[1]
	} else {
		entry23 = entries[1]
		entry45 = entries[0]
	}

	verifyEntry(t, entry23, 23, "AUUID", 9012344)
	verifyEntry(t, entry45, 45, "BUUID", 7455623)

	// Verify bucket "bucketa".
	vector, ok = bucketVectorMap["bucketa"]
	if !ok {
		t.Errorf("Could not find element bucketa.")
	}
	entries = vector.Entries()
	if len(entries) != 1 {
		t.Errorf("Expected bucketa entries length 1, actual %d", len(entries))
	}
	entry := entries[0]
	verifyEntry(t, entry, 1000, "CUUID", 1234567)
}

func TestFullScanVector(t *testing.T) {
	jsonText := "[FULL_SCAN_VECTOR]"
	fullScanElement := "[777777, \"VUUID\"],"
	replacement := strings.TrimRight(strings.Repeat(fullScanElement, 1024), ",")
	jsonText = strings.Replace(jsonText, "FULL_SCAN_VECTOR", replacement, 1)

	decoder := json.NewDecoder(strings.NewReader(jsonText))
	var target interface{}
	e := decoder.Decode(&target)
	if e != nil {
		t.Errorf("Unexpected JSON parsing error %v", e)
	}
	var vector timestamp.Vector
	vector, err := getScanVectorFromJSON(target)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	entries := vector.Entries()
	if len(entries) != 1024 {
		t.Errorf("Expected entries lenth 1024, actual %d", len(entries))
	}
	for i, entry := range entries {
		verifyEntry(t, entry, uint32(i), "VUUID", 777777)
	}
}

func TestSparseScanVector(t *testing.T) {
	jsonText := `{ 
		"3": [ 345, "AAUID" ],
		"5": [ 100001, "BAUID" ],
		"7": [ 999999, "CAUID" ]
	}`
	var target interface{}
	decoder := json.NewDecoder(strings.NewReader(jsonText))
	e := decoder.Decode(&target)
	if e != nil {
		t.Errorf("Unexpected error %v", e)
	}

	var actual timestamp.Vector // Verify expected return type.
	actual, err := getScanVectorFromJSON(target)

	if err != nil {
		t.Errorf("expected %v, actual %v", nil, err)
	}

	// Results may appear in any order in entry array.
	if len(actual.Entries()) != 3 {
		t.Errorf("expected length 3, actual %d", len(actual.Entries()))
	}
	for _, entry := range actual.Entries() {
		switch entry.Position() {
		case 3:
			verifyEntry(t, entry, 3, "AAUID", 345)
		case 5:
			verifyEntry(t, entry, 5, "BAUID", 100001)
		case 7:
			verifyEntry(t, entry, 7, "CAUID", 999999)
		default:
			t.Errorf("Unexpected position %d", entry.Position())
		}
	}
}

func TestRequestDefaults(t *testing.T) {
	statement := "select 1"
	logging.Infof("statement : %v", statement)
	doUrlRequest(t, map[string]string{
		"statement": statement,
	})
	if test_server.request().State() != server.COMPLETED {
		t.Errorf("Expected request state: %v, actual: %v\n", server.COMPLETED, test_server.request().State())
	}
	if test_server.request().Statement() != statement {
		t.Errorf("Expected request statement: %v, actual: %v\n", statement, test_server.request().Statement())
	}
}

func TestRequestWithTimeout(t *testing.T) {
	request_timeout := "100ms"
	expected_timeout := time.Millisecond * 100

	payload := map[string]interface{}{
		"statement": "select 1",
		"timeout":   request_timeout,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	if test_server.request().Timeout() != expected_timeout {
		t.Errorf("Expected timeout: %v, actual: %v\n", expected_timeout, test_server.request().Timeout())
	}
}

func TestPrepareStatements(t *testing.T) {
	preparedSequence(t, "doSelect", "SELECT b FROM p0:b0 LIMIT 5")
	preparedSequence(t, "doInsert", "INSERT INTO p0:b0 VALUES ($1, $2)")
}

// insert requires parameters
var insertArgs []interface{} = []interface{}{"aaa", "bbb"}

func preparedSequence(t *testing.T, name string, stmt string) {
	// Verify a sequence of requests:

	// { "prepared": "name" }  fails with "no such name" error
	doNoSuchPrepared(t, name)

	// { "statement": "prepare name as <<N1QL statement>>" }  succeeds
	doPrepare(t, name, stmt)

	// { "prepared": "name" }  succeeds
	doJsonRequest(t, map[string]interface{}{
		"prepared": name,
		"args":     insertArgs,
	})

	prepared, _ := prepareds.GetPrepared(name, nil)
	if prepared == nil {
		t.Errorf("Expected to resolve prepared statement with name %v", name)
		return
	}

	// { "prepared": "name", "encoded_plan": "<<encoded plan>>" }  succeeds
	doJsonRequest(t, map[string]interface{}{
		"prepared":     name,
		"encoded_plan": prepared.EncodedPlan(),
		"args":         insertArgs,
	})

	// { "prepared": "name", "statement": "statement text" }  fails with "multiple values" error
	doPreparedAndStatement(t, name, stmt)

}

func doNoSuchPrepared(t *testing.T, name string) {
	payload := map[string]interface{}{
		"prepared": name,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	errs := test_server.request().Errors()
	if len(errs) == 0 {
		t.Errorf("Expected error: %v no such prepared, got nothing", errors.E_NO_SUCH_PREPARED)
	} else if len(errs) > 1 {
		t.Errorf("Expected error: %v no such prepared, got %v", errors.E_NO_SUCH_PREPARED, errs)
	} else if errs[0].Code() != errors.E_NO_SUCH_PREPARED {
		t.Errorf("Expected error condition: no such prepared. Received: %v", errs[0])
	}
}

func doPrepare(t *testing.T, name string, stmt string) {
	doJsonRequest(t, map[string]interface{}{
		"statement": "prepare " + name + " as " + stmt,
	})

	// Verify the name is in the prepared cache:
	prepared, err := prepareds.GetPrepared(name, nil)
	if err != nil {
		t.Errorf("Unexpected error looking up prepared: %v", err)
	}

	if prepared == nil {
		t.Errorf("Expected to resolve prepared statement with name %s", name)
	}
}

func doPreparedWithPlan(t *testing.T, name string, encoded_plan string) {
	doJsonRequest(t, map[string]interface{}{
		"prepared":     name,
		"encoded_plan": encoded_plan,
	})
}

func doPreparedNameOnly(t *testing.T, name string) {
	doJsonRequest(t, map[string]interface{}{
		"prepared": name,
	})
}

func doPreparedAndStatement(t *testing.T, name string, stmt string) {
	payload := map[string]interface{}{
		"prepared":  name,
		"statement": stmt,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	errs := test_server.request().Errors()
	if len(errs) == 0 {
		t.Errorf("Expected error: %v multiple values, got nothing", errors.E_SERVICE_MULTIPLE_VALUES)
	} else if len(errs) > 1 {
		t.Errorf("Expected error: %v multiple values, got %v", errors.E_SERVICE_MULTIPLE_VALUES, errs)
	} else if errs[0].Code() != errors.E_SERVICE_MULTIPLE_VALUES {
		t.Errorf("Expected error condition: multiple values. Received: %v", errs[0])
	}
}

func doJsonRequest(t *testing.T, payload map[string]interface{}) {
	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	errs := test_server.request().Errors()
	if len(errs) > 0 {

		// insert is not implemented in mock
		if errs[0].Code() != 16003 {
			t.Errorf("Unexpected error: %v payload: %v", errs, payload)
		}
	}
}

func doUrlRequest(t *testing.T, params map[string]string) {
	payload := url.Values{}
	for param, value := range params {
		payload.Set(param, value)
	}

	_, err := doUrlEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	errs := test_server.request().Errors()
	if len(errs) > 0 {
		t.Errorf("Unexpected error: %v", errs)
	}
}

func makeMockServer() *server.Server {
	store, err := resolver.NewDatastore("mock:")
	if err != nil {
		logging.Errorf(err.Error())
		os.Exit(1)
	}

	datastore.SetDatastore(store)
	sys, err := system.NewDatastore(store, nil)
	server, err := server.NewServer(store, sys, nil, nil, "default",
		false, 10, 10, 4, 4, 0, 0, false, false, false, true, server.ProfOff, false, nil)
	if err != nil {
		logging.Errorf(err.Error())
		os.Exit(1)
	}
	server.SetKeepAlive(1 << 10)

	return server
}

var _ALL_USERS = auth.NewCredentials("dummy", "dummy")

func (this *testServer) testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		this.query_request = &httpRequest{}
		newHttpRequest(this.query_request, w, r, NewSyncPool(1024), 1024, "default")
		if this.query_request.State() == server.FATAL {
			return
		}
		this.query_request.SetCredentials(_ALL_USERS)
		this.query_server.ServiceRequest(this.query_request)
	})
}

func doUrlEncodedPost(payload url.Values) (*http.Response, error) {
	u, err := url.ParseRequestURI(test_server.URL())
	if err != nil {
		return nil, err
	}

	u.Path = "/"
	urlStr := fmt.Sprintf("%v", u)
	req, err := http.NewRequest("POST", urlStr, bytes.NewBufferString(payload.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func doJsonEncodedPost(payload map[string]interface{}) (*http.Response, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	u, err := url.ParseRequestURI(test_server.URL())
	if err != nil {
		return nil, err
	}

	u.Path = "/"
	urlStr := fmt.Sprintf("%v", u)
	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
