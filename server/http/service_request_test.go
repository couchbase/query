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

	"github.com/couchbase/query/datastore/resolver"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"

	log_resolver "github.com/couchbase/query/logging/resolver"
	"github.com/couchbase/query/server"
)

var test_server *httptest.Server
var query_server *server.Server
var query_request *httpRequest

func init() {
	logger, _ := log_resolver.NewLogger("golog")
	if logger == nil {
		fmt.Printf("Unable to create logger")
		os.Exit(1)
	}
	logging.SetLogger(logger)
	query_server = makeMockServer()
	test_server = httptest.NewServer(testHandler())
}

func TestRequestDefaults(t *testing.T) {
	statement := "select 1"
	payload := url.Values{}
	payload.Set("statement", statement)

	_, err := doUrlEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	if query_request.State() != server.COMPLETED {
		t.Errorf("Expected request state: %v, actual: %v\n", server.COMPLETED, query_request.State())
	}

	if query_request.Statement() != statement {
		t.Errorf("Expected request statement: %v, actual: %v\n", statement, query_request.Statement())
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

	if query_request.Timeout() != expected_timeout {
		t.Errorf("Expected timeout: %v, actual: %v\n", expected_timeout, query_request.Timeout())
	}
}

func TestPrepared(t *testing.T) {
	name := "name"
	stmt1 := "SELECT 1"

	// Verify the following sequence of requests:
	// { "prepared": "name" }  fails with "no such name" error
	// { "statement": "prepare name as SELECT 1" succeeds
	// { "prepared": "name" }  succeeds
	// { "prepared": "name", "encoded_plan": "<<encoded plan>>" }  succeeds

	doNoSuchPrepared(t, name)
	doPrepare(t, name, stmt1)
	doPreparedNameOnly(t, name)
	prepared, _ := plan.GetPrepared(value.NewValue(name))
	doPreparedWithPlan(t, name, prepared.EncodedPlan())
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
	case err := <-query_request.Errors():
		if err.Code() != errors.NO_SUCH_PREPARED {
			t.Errorf("Expected error condition: no such prepared. Recieved: %v", err)
		}
	default:
		t.Errorf("Expected error: %v no such prepared", errors.NO_SUCH_PREPARED)
	}
}

func doPrepare(t *testing.T, name string, stmt string) {
	payload := map[string]interface{}{
		"statement": "prepare " + name + " as " + stmt,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	if query_request.State() != server.COMPLETED {
		t.Errorf("Expected request state: %v, actual: %v\n", server.COMPLETED, query_request.State())
	}

	// Verify the name is in the prepared cache:
	prepared, err := plan.GetPrepared(value.NewValue(name))
	if err != nil {
		t.Errorf("Unexpected error looking up prepared: %v", err)
	}

	if prepared == nil {
		t.Errorf("Expected to resolve prepared statement")
	}
}

func doPreparedWithPlan(t *testing.T, name string, encoded_plan string) {
	payload := map[string]interface{}{
		"prepared":     name,
		"encoded_plan": encoded_plan,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	select {
	case err = <-query_request.Errors():
		t.Errorf("Unexpected error: %v", err)
	default:
	}
}

func doPreparedNameOnly(t *testing.T, name string) {
	payload := map[string]interface{}{
		"prepared": name,
	}

	_, err := doJsonEncodedPost(payload)
	if err != nil {
		t.Errorf("Unexpected error in HTTP request: %v", err)
	}

	select {
	case err = <-query_request.Errors():
		t.Errorf("Unexpected error: %v", err)
	default:
	}
}

func makeMockServer() *server.Server {
	datastore, err := resolver.NewDatastore("http://localhost:8091")
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}

	channel := make(server.RequestChannel, 10)
	plusChannel := make(server.RequestChannel, 10)
	server, err := server.NewServer(datastore, nil, nil, "default",
		false, channel, plusChannel, 4, 4, 0, 0, false, false, false)
	if err != nil {
		logging.Errorp(err.Error())
		os.Exit(1)
	}
	server.SetKeepAlive(1 << 10)

	go server.Serve()
	return server
}

func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query_request = newHttpRequest(w, r, NewSyncPool(1024), 1024)
		if query_request.State() == server.FATAL {
			return
		}
		select {
		case query_server.Channel() <- query_request:
			// Wait until the request exits.
			<-query_request.CloseNotify()
		default:
		}
	})
}

func doUrlEncodedPost(payload url.Values) (*http.Response, error) {
	u, err := url.ParseRequestURI(test_server.URL)
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

	u, err := url.ParseRequestURI(test_server.URL)
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
