//  Copyright (c) 2015 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/value"

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
	test_server = newTestServer()
}

func TestMakeSparseVector(t *testing.T) {
	d1 := restArg{float64(345), "AAUID"}
	d2 := restArg{float64(100001), "BAUID"}
	d3 := restArg{float64(999999), "CAUID"}

	vdata := map[string]*restArg{
		"3": &d1,
		"5": &d2,
		"7": &d3,
	}

	var actual timestamp.Vector // Verify expected return type.
	actual, err := makeSparseVector(vdata)

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
			if entry.Value() != 345 {
				t.Errorf("expected %d, actual %d", 345, entry.Value())
			}
			if entry.Guard() != "AAUID" {
				t.Errorf("expected %s, actual %s", "AAUID", entry.Guard())
			}
		case 5:
			if entry.Value() != 100001 {
				t.Errorf("expected %d, actual %d", 100001, entry.Value())
			}
			if entry.Guard() != "BAUID" {
				t.Errorf("expected %s, actual %s", "BAUID", entry.Guard())
			}
		case 7:
			if entry.Value() != 999999 {
				t.Errorf("expected %d, actual %d", 999999, entry.Value())
			}
			if entry.Guard() != "CAUID" {
				t.Errorf("expected %s, actual %s", "CAUID", entry.Guard())
			}
		default:
			t.Errorf("Unexpected position %d", entry.Position())
		}
	}
}

func TestRequestDefaults(t *testing.T) {
	statement := "select 1"
	logging.Infop("message", logging.Pair{"statement", statement})
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

func preparedSequence(t *testing.T, name string, stmt string) {
	// Verify a sequence of requests:

	// { "prepared": "name" }  fails with "no such name" error
	doNoSuchPrepared(t, name)

	// { "statement": "prepare name as <<N1QL statement>>" }  succeeds
	doPrepare(t, name, stmt)

	// { "prepared": "name" }  succeeds
	doJsonRequest(t, map[string]interface{}{
		"prepared": name,
	})

	prepared, _ := plan.GetPrepared(value.NewValue(name))
	if prepared == nil {
		t.Errorf("Expected to resolve prepared statement with name %v", name)
		return
	}

	// { "prepared": "name", "encoded_plan": "<<encoded plan>>" }  succeeds
	doJsonRequest(t, map[string]interface{}{
		"prepared":     name,
		"encoded_plan": prepared.EncodedPlan(),
	})

}

func doNoSuchPrepared(t *testing.T, name string) {
	payload := map[string]interface{}{
		"prepared": name,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	select {
	case err := <-test_server.request().Errors():
		if err.Code() != errors.NO_SUCH_PREPARED {
			t.Errorf("Expected error condition: no such prepared. Recieved: %v", err)
		}
	default:
		t.Errorf("Expected error: %v no such prepared", errors.NO_SUCH_PREPARED)
	}
}

func doPrepare(t *testing.T, name string, stmt string) {
	doJsonRequest(t, map[string]interface{}{
		"statement": "prepare " + name + " as " + stmt,
	})

	// Verify the name is in the prepared cache:
	prepared, err := plan.GetPrepared(value.NewValue(name))
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

func doJsonRequest(t *testing.T, payload map[string]interface{}) {
	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	select {
	case err = <-test_server.request().Errors():
		t.Errorf("Unexpected error: %v", err)
	default:
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

	select {
	case err = <-test_server.request().Errors():
		t.Errorf("Unexpected error: %v", err)
	default:
	}
}

func makeMockServer() *server.Server {
	store, err := resolver.NewDatastore("mock:")
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	datastore.SetDatastore(store)
	channel := make(server.RequestChannel, 10)
	plusChannel := make(server.RequestChannel, 10)
	server, err := server.NewServer(store, nil, nil, nil, "default",
		false, channel, plusChannel, 4, 4, 0, 0, false, false, false)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	server.SetKeepAlive(1 << 10)

	go server.Serve()
	return server
}

func (this *testServer) testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		this.query_request = newHttpRequest(w, r, NewSyncPool(1024), 1024)
		if this.query_request.State() == server.FATAL {
			return
		}
		select {
		case this.query_server.Channel() <- this.query_request:
			// Wait until the request exits.
			<-this.query_request.CloseNotify()
		default:
		}
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
