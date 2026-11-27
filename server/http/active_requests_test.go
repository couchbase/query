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
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/primitives/couchbase"
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

// MB-74936: a cancel another node sends on behalf of a user is audited as that user, not as the internal user,
// and a cancel sent directly, rather than by another node, is audited as a cancelled request too
func TestActiveRequestsCancelAuditUser(t *testing.T) {
	m, err := mock.NewDatastore("mock:")
	if err != nil {
		t.Fatalf("mock datastore: %v", err)
	}
	prev := datastore.GetDatastore()
	datastore.SetDatastore(&mock.AuthStore{Datastore: m})
	defer datastore.SetDatastore(prev)

	var cancels []audit.RequestCancelFields
	prevAudit := auditRequestCancel
	auditRequestCancel = func(f *audit.RequestCancelFields) { cancels = append(cancels, *f) }
	defer func() { auditRequestCancel = prevAudit }()

	endpoint := &HttpEndpoint{actives: NewActiveRequests(nil)}
	cancel := func(user, userAgent, query string) (string, []string) {
		r := &httpRequest{}
		newHttpRequest(r, httptest.NewRecorder(), httptest.NewRequest("GET", "/query/service?statement=select+1", nil),
			NewSyncPool(1024), 1024, "default")
		r.SetCredentials(auth.NewCredentials("alice", "pw"))
		if err := endpoint.actives.Put(r); err != nil {
			t.Fatalf("Put: %v", err)
		}

		req := httptest.NewRequest("DELETE", "/admin/active_requests/"+r.Id().String()+query, nil)
		req.SetBasicAuth(user, "pw")
		req.Header.Set("User-Agent", userAgent)
		req = req.WithContext(context.WithValue(req.Context(), "request", r.Id().String()))
		af := &audit.ApiAuditFields{RemoteAddress: "10.0.0.1:5000"}
		cancels = nil
		if _, err := doActiveRequest(endpoint, httptest.NewRecorder(), req, af); err != nil {
			t.Fatalf("%v cancelling%v: %v", user, query, err)
		}
		return r.Id().String(), af.Users
	}

	if _, users := cancel("@cbq-engine", couchbase.USER_AGENT, "?impersonate=local:alice"); !slices.Equal(users,
		[]string{"local:alice"}) {
		t.Errorf("cancel fanned out for alice audited as %v", users)
	}
	if len(cancels) != 0 {
		t.Errorf("cancel fanned out for alice audited as a cancelled request %+v: the node that sent it audits it",
			cancels)
	}

	// an admin's cancel is sent without impersonate: the internal user is all the remote node knows
	if _, users := cancel("@cbq-engine", couchbase.USER_AGENT, ""); !slices.Equal(users,
		[]string{"local:@cbq-engine"}) {
		t.Errorf("cancel fanned out for an admin audited as %v", users)
	}
	if len(cancels) != 0 {
		t.Errorf("cancel fanned out for an admin audited as a cancelled request %+v", cancels)
	}

	// impersonate from a user who isn't the internal user doesn't change who is audited
	id, users := cancel("alice", "curl", "?impersonate=local:bob")
	if !slices.Equal(users, []string{"local:alice"}) {
		t.Errorf("alice's own cancel audited as %v", users)
	}
	expected := []audit.CancelledRequest{{RequestId: id, Node: distributed.RemoteAccess().WhoAmI()}}
	if len(cancels) != 1 || !slices.Equal(cancels[0].Users, []string{"local:alice"}) ||
		!slices.Equal(cancels[0].Cancelled, expected) || cancels[0].RequestId != "" ||
		cancels[0].UserAgent != "curl" || cancels[0].RemoteAddress != "10.0.0.1:5000" {
		t.Errorf("alice's own cancel audited as cancelled requests %+v, expected %v", cancels, expected)
	}
}
