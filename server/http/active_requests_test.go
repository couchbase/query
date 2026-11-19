//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
)

func TestActiveRequestsDelete(t *testing.T) {
	r := httptest.NewRequest("GET", "/query/service?statement=select+1", nil)
	req := &httpRequest{}
	newHttpRequest(req, httptest.NewRecorder(), r, NewSyncPool(1024), 1024, "default")
	id := req.Id().String()

	actives := NewActiveRequests(nil)
	if err := actives.Put(req); err != nil {
		t.Fatalf("Put: %v", err)
	}

	if actives.Delete("no-such-id", true, nil) {
		t.Errorf("Delete of an unknown id reported success")
	}
	if actives.Delete(id, true, func(server.Request) bool { return false }) {
		t.Errorf("Delete rejected by the check reported success")
	}
	if n, _ := actives.Count(); n != 1 {
		t.Errorf("request removed although the check rejected it, count %v", n)
	}
	if !actives.Delete(id, true, nil) {
		t.Errorf("Delete of an active request reported failure")
	}
	if n, _ := actives.Count(); n != 0 {
		t.Errorf("request not removed, count %v", n)
	}
	if actives.Delete(id, true, nil) {
		t.Errorf("second Delete of the same id reported success")
	}
}

func TestAdminStmtNotFoundStatus(t *testing.T) {
	w := httptest.NewRecorder()
	if status := writeError(w, errors.NewSystemStmtNotFoundError(nil, "x")); status != http.StatusNotFound {
		t.Errorf("writeError status %v, expected %v", status, http.StatusNotFound)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("response code %v, expected %v", w.Code, http.StatusNotFound)
	}
}

// MB-74936: users without PRIV_SYSTEM_READ only see and cancel their own active requests
func TestActiveRequestsOwnership(t *testing.T) {
	m, err := mock.NewDatastore("mock:")
	if err != nil {
		t.Fatalf("mock datastore: %v", err)
	}
	prev := datastore.GetDatastore()
	datastore.SetDatastore(&mock.AuthStore{Datastore: m})
	defer datastore.SetDatastore(prev)

	endpoint := &HttpEndpoint{actives: NewActiveRequests(nil)}
	ids := map[string]string{}
	for _, user := range []string{"alice", "bob", "admin"} {
		req := &httpRequest{}
		newHttpRequest(req, httptest.NewRecorder(), httptest.NewRequest("GET", "/query/service?statement=select+1", nil),
			NewSyncPool(1024), 1024, "default")
		req.SetCredentials(auth.NewCredentials(user, "pw"))
		if err := endpoint.actives.Put(req); err != nil {
			t.Fatalf("Put: %v", err)
		}
		ids[user] = req.Id().String()
	}

	call := func(handler func(*HttpEndpoint, http.ResponseWriter, *http.Request, *audit.ApiAuditFields) (any, errors.Error),
		method, user, query, id string) (any, errors.Error) {

		req := httptest.NewRequest(method, "/admin/active_requests"+query, nil)
		if user != "" {
			req.SetBasicAuth(user, "pw")
		}
		if id != "" {
			req = req.WithContext(context.WithValue(req.Context(), "request", id))
		}
		return handler(endpoint, httptest.NewRecorder(), req, &audit.ApiAuditFields{})
	}
	index := func(user, query string) []string {
		res, err := call(doRequestIndex, "GET", user, query, "")
		if err != nil {
			t.Fatalf("index as %q%s: %v", user, query, err)
		}
		rv := res.([]string)
		slices.Sort(rv)
		return rv
	}
	sorted := func(users ...string) []string {
		rv := []string{}
		for _, u := range users {
			rv = append(rv, ids[u])
		}
		slices.Sort(rv)
		return rv
	}

	// listing
	if got, want := index("alice", ""), sorted("alice"); !slices.Equal(got, want) {
		t.Errorf("alice sees %v, expected %v", got, want)
	}
	if got, want := index("alice", "?impersonate=local:bob"), sorted("alice"); !slices.Equal(got, want) {
		t.Errorf("alice impersonating bob sees %v, expected %v", got, want)
	}
	if got, want := index("admin", ""), sorted("alice", "bob", "admin"); !slices.Equal(got, want) {
		t.Errorf("admin sees %v, expected %v", got, want)
	}
	if got, want := index("admin", "?impersonate=local:bob"), sorted("bob"); !slices.Equal(got, want) {
		t.Errorf("admin impersonating bob sees %v, expected %v", got, want)
	}
	if res, err := call(doActiveRequests, "GET", "alice", "", ""); err != nil {
		t.Errorf("list as alice: %v", err)
	} else if n := len(res.([]any)); n != 1 {
		t.Errorf("list as alice returned %v requests, expected 1", n)
	}

	// unauthenticated
	if _, err := call(doRequestIndex, "GET", "", "", ""); err == nil {
		t.Errorf("unauthenticated index succeeded")
	}
	if _, err := call(doActiveRequest, "DELETE", "", "", ids["alice"]); err == nil {
		t.Errorf("unauthenticated delete succeeded")
	}

	// single request
	if res, err := call(doActiveRequest, "GET", "alice", "", ids["bob"]); err != nil || res != nil {
		t.Errorf("alice fetching bob's request: %v, %v", res, err)
	}
	if res, err := call(doActiveRequest, "GET", "alice", "", ids["alice"]); err != nil || res == nil {
		t.Errorf("alice fetching her own request: %v, %v", res, err)
	}

	// cancellation
	if _, err := call(doActiveRequest, "DELETE", "alice", "?impersonate=local:bob", ids["bob"]); err == nil ||
		err.Code() != errors.E_SYSTEM_STMT_NOT_FOUND {
		t.Errorf("alice cancelling bob's request: %v", err)
	}
	if _, err := call(doActiveRequest, "DELETE", "alice", "", ids["alice"]); err != nil {
		t.Errorf("alice cancelling her own request: %v", err)
	}
	if _, err := call(doActiveRequest, "DELETE", "admin", "", ids["bob"]); err != nil {
		t.Errorf("admin cancelling bob's request: %v", err)
	}
	if got, want := index("admin", ""), sorted("admin"); !slices.Equal(got, want) {
		t.Errorf("after cancellations admin sees %v, expected %v", got, want)
	}
}
