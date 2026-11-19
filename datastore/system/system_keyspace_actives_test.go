//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"strings"
	"testing"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// userContext supplies credentials, and caches the first credentials like a real request does
type userContext struct {
	datastore.QueryContext
	creds         *auth.Credentials
	firstCreds    string
	firstCredsSet bool
}

func (this *userContext) Credentials() *auth.Credentials {
	return this.creds
}

func (this *userContext) FirstCreds() (string, bool) {
	return this.firstCreds, this.firstCredsSet
}

func (this *userContext) SetFirstCreds(userName string) {
	this.firstCreds = userName
	this.firstCredsSet = true
}

func newUserContext(user string) *userContext {
	return &userContext{QueryContext: datastore.NULL_QUERY_CONTEXT, creds: auth.NewCredentials(user, "pw")}
}

// testRequest only implements what the active_requests keyspace uses
type testRequest struct {
	server.Request
	creds *auth.Credentials
}

func (this *testRequest) Credentials() *auth.Credentials {
	return this.creds
}

func (this *testRequest) Controls() value.Tristate {
	return value.FALSE
}

func (this *testRequest) Sensitive() bool {
	return false
}

func (this *testRequest) Format(util.DurationStyle, bool, bool, bool) map[string]any {
	return map[string]any{"users": datastore.CredsString(this.creds)}
}

type testActives map[string]server.Request

func (this testActives) Put(r server.Request) errors.Error { return nil }
func (this testActives) Count() (int, errors.Error)        { return len(this), nil }
func (this testActives) Load() int                         { return 0 }

func (this testActives) Get(id string, f func(server.Request)) errors.Error {
	if r, ok := this[id]; ok {
		f(r)
	}
	return nil
}

func (this testActives) Delete(id string, stop bool, f func(server.Request) bool) bool {
	r, ok := this[id]
	if !ok || (f != nil && !f(r)) {
		return false
	}
	delete(this, id)
	return true
}

func (this testActives) ForEach(nonBlocking func(string, server.Request) bool, blocking func() bool) {
	for id, r := range this {
		if !nonBlocking(id, r) || (blocking != nil && !blocking()) {
			return
		}
	}
}

// twoNodes is a two node cluster, "local" and "remote", that records the keys sent to the remote node
type twoNodes struct {
	distributed.SystemRemoteAccess
	keys []string
}

// newTwoNodes makes the local node part of a twoNodes cluster until the test ends
func newTwoNodes(t *testing.T) *twoNodes {
	prevRa := distributed.RemoteAccess()
	cluster := &twoNodes{SystemRemoteAccess: prevRa}
	distributed.SetRemoteAccess(cluster)
	t.Cleanup(func() { distributed.SetRemoteAccess(prevRa) })
	return cluster
}

func (this *twoNodes) WhoAmI() string {
	return "local"
}

func (this *twoNodes) SplitKey(key string) (string, string) {
	if strings.HasPrefix(key, "[remote]") {
		return "remote", key[len("[remote]"):]
	}
	return "", key
}

func (this *twoNodes) GetRemoteDoc(node string, key string, endpoint string, command string,
	docFn func(map[string]any), warnFn func(errors.Error), creds distributed.Creds, authToken string,
	formData map[string]any) {
	this.keys = append(this.keys, key)
}

// newTestKeyspace sets up a mock datastore that authorizes users as mock.AuthStore does, and returns its system
// keyspace name; actives, if not nil, replace the active requests. The test restores both when it ends: no
// active requests are set in this package's tests otherwise.
func newTestKeyspace(t *testing.T, name string, actives server.ActiveRequests) datastore.Keyspace {
	m, err := mock.NewDatastore("mock:")
	if err != nil {
		t.Fatalf("mock datastore: %v", err)
	}
	prevDs := datastore.GetDatastore()
	datastore.SetDatastore(&mock.AuthStore{Datastore: m})
	t.Cleanup(func() { datastore.SetDatastore(prevDs) })

	if actives != nil {
		server.SetActives(actives)
		t.Cleanup(func() { server.SetActives(nil) })
	}

	s, err := NewDatastore(m, nil, true)
	if err != nil {
		t.Fatalf("system datastore: %v", err)
	}
	p, err := s.NamespaceById(datastore.SYSTEM_NAMESPACE)
	if err != nil {
		t.Fatalf("system namespace: %v", err)
	}
	ks, err := p.KeyspaceByName(name)
	if err != nil {
		t.Fatalf("%v keyspace: %v", name, err)
	}
	return ks
}

// MB-74936: on prem, users without PRIV_SYSTEM_READ only see and cancel their own active requests
func TestActiveRequestsOwnership(t *testing.T) {
	actives := testActives{}
	for _, user := range []string{"alice", "bob", "admin"} {
		actives[user+"-id"] = &testRequest{creds: auth.NewCredentials(user, "pw")}
	}
	ks := newTestKeyspace(t, KEYSPACE_NAME_ACTIVE, actives)

	for user, expected := range map[string]int64{"alice": 1, "bob": 1, "admin": 3} {
		if n, err := ks.Count(newUserContext(user)); err != nil || n != expected {
			t.Errorf("count as %v: %v %v, expected %v", user, n, err, expected)
		}
	}

	vals := map[string]value.AnnotatedValue{}
	if errs := ks.Fetch([]string{"alice-id", "bob-id"}, vals, newUserContext("alice"), nil, nil, false); errs != nil {
		t.Errorf("fetch as alice: %v", errs)
	}
	if len(vals) != 1 || vals["alice-id"] == nil {
		t.Errorf("alice fetched %v, expected only alice-id", vals)
	}

	deletes := value.Pairs{{Name: "bob-id"}, {Name: "alice-id"}}
	if _, _, errs := ks.Delete(deletes, newUserContext("alice"), false); errs != nil {
		t.Errorf("delete as alice: %v", errs)
	}
	if _, ok := actives["bob-id"]; !ok {
		t.Errorf("alice cancelled bob's request")
	}
	if _, ok := actives["alice-id"]; ok {
		t.Errorf("alice could not cancel her own request")
	}

	if _, _, errs := ks.Delete(value.Pairs{{Name: "bob-id"}}, newUserContext("admin"), false); errs != nil {
		t.Errorf("delete as admin: %v", errs)
	}
	if _, ok := actives["bob-id"]; ok {
		t.Errorf("admin could not cancel bob's request")
	}
}

// authErrorStore fails every privilege check with an error other than a denial, such as cbauth unavailable
type authErrorStore struct {
	mock.AuthStore
}

func (s *authErrorStore) AuthorizeInternal(privs *auth.Privileges, creds *auth.Credentials) errors.Error {
	if err := s.Authorize(nil, creds); err != nil || privs == nil || len(privs.List) == 0 {
		return err
	}
	return errors.NewDatastoreAuthorizationError(nil)
}

// MB-74936: a context with a user's credentials restricts to that user, and a failure to check PRIV_SYSTEM_READ
// fails the operation rather than restricting an administrator to their own requests
func TestActiveRequestsUserCheck(t *testing.T) {
	actives := testActives{}
	for _, user := range []string{"alice", "bob", "admin"} {
		actives[user+"-id"] = &testRequest{creds: auth.NewCredentials(user, "pw")}
	}
	ks := newTestKeyspace(t, KEYSPACE_NAME_ACTIVE, actives)

	ctx := datastore.NewQueryContextWithCredentials(auth.NewCredentials("alice", "pw"))
	if n, err := ks.Count(ctx); err != nil || n != 1 {
		t.Errorf("count with alice's credentials: %v %v, expected 1", n, err)
	}

	datastore.SetDatastore(&authErrorStore{AuthStore: *datastore.GetDatastore().(*mock.AuthStore)})
	admin := newUserContext("admin")
	if _, err := ks.Count(admin); err == nil || err.Code() != errors.E_DATASTORE_AUTHORIZATION {
		t.Errorf("count with failing authorization: %v, expected an authorization error", err)
	}
	if _, set := admin.FirstCreds(); set {
		t.Errorf("the user was cached although PRIV_SYSTEM_READ couldn't be checked")
	}
	vals := map[string]value.AnnotatedValue{}
	if errs := ks.Fetch([]string{"bob-id"}, vals, admin, nil, nil, false); len(errs) != 1 || len(vals) != 0 {
		t.Errorf("fetch with failing authorization: %v %v, expected an error and nothing", errs, vals)
	}
	if _, _, errs := ks.Delete(value.Pairs{{Name: "bob-id"}}, admin, false); len(errs) != 1 {
		t.Errorf("delete with failing authorization: %v, expected an error", errs)
	}
	if _, ok := actives["bob-id"]; !ok {
		t.Errorf("bob's request was cancelled although PRIV_SYSTEM_READ couldn't be checked")
	}
}
