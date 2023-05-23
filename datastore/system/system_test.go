//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"net/http"
	"testing"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

type queryContextImpl struct {
	req   *http.Request
	creds *auth.Credentials
	t     *testing.T
}

func (ci *queryContextImpl) Credentials() *auth.Credentials {
	return ci.creds
}

func (ci *queryContextImpl) GetReqDeadline() time.Time {
	return time.Time{}
}

func (ci *queryContextImpl) GetTxContext() interface{} {
	return nil
}

func (ci *queryContextImpl) UseReplica() bool {
	return false
}

func (ci *queryContextImpl) Datastore() datastore.Datastore {
	return datastore.GetDatastore()
}

func (ci *queryContextImpl) SetTxContext(tc interface{}) {
}

func (ci *queryContextImpl) TxDataVal() value.Value {
	return nil
}

func (ci *queryContextImpl) Warning(warn errors.Error) {
	ci.t.Logf("datastore warning: %v", warn)
}

func (ci *queryContextImpl) Error(err errors.Error) {
	ci.t.Logf("datastore error: %v", err)
}

func (ci *queryContextImpl) Fatal(fatal errors.Error) {
	ci.t.Logf("scan fatal: %v", fatal)
}

func (ci *queryContextImpl) DurabilityLevel() datastore.DurabilityLevel {
	return datastore.DL_NONE
}

func (ci *queryContextImpl) KvTimeout() time.Duration {
	return datastore.DEF_KVTIMEOUT
}

func (ci *queryContextImpl) PreserveExpiry() bool {
	return false
}

func (ci *queryContextImpl) TenantCtx() tenant.Context {
	return nil
}

func (ci *queryContextImpl) FirstCreds() (string, bool) {
	return "", true
}

func (ci *queryContextImpl) SetFirstCreds(string) {
}

func (ci *queryContextImpl) GetScanCap() int64 {
	return 16
}

func (ci *queryContextImpl) MaxParallelism() int {
	return 1
}

func (ci *queryContextImpl) RecordFtsRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) RecordGsiRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) RecordKvRU(ru tenant.Unit) {
}

func (ci *queryContextImpl) RecordKvWU(wu tenant.Unit) {
}

func (ci *queryContextImpl) SkipKey(key string) bool {
	return false
}

func (ci *queryContextImpl) IsActive() bool {
	return true
}

func (ci *queryContextImpl) RequestId() string {
	return ""
}

func (ci *queryContextImpl) Loga(l logging.Level, f func() string)                   {}
func (ci *queryContextImpl) Debuga(f func() string)                                  {}
func (ci *queryContextImpl) Tracea(f func() string)                                  {}
func (ci *queryContextImpl) Infoa(f func() string)                                   {}
func (ci *queryContextImpl) Warna(f func() string)                                   {}
func (ci *queryContextImpl) Errora(f func() string)                                  {}
func (ci *queryContextImpl) Severea(f func() string)                                 {}
func (ci *queryContextImpl) Fatala(f func() string)                                  {}
func (ci *queryContextImpl) Logf(level logging.Level, f string, args ...interface{}) {}
func (ci *queryContextImpl) Debugf(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Tracef(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Infof(f string, args ...interface{})                     {}
func (ci *queryContextImpl) Warnf(f string, args ...interface{})                     {}
func (ci *queryContextImpl) Errorf(f string, args ...interface{})                    {}
func (ci *queryContextImpl) Severef(f string, args ...interface{})                   {}
func (ci *queryContextImpl) Fatalf(f string, args ...interface{})                    {}

func (ci *queryContextImpl) ErrorLimit() int {
	return errors.DEFAULT_REQUEST_ERROR_LIMIT
}

func (ci *queryContextImpl) ErrorCount() int {
	return 0
}

func TestSystem(t *testing.T) {
	// Use mock to test system; 2 namespaces with 5 keyspaces per namespace
	m, err := mock.NewDatastore("mock:namespaces=2,keyspaces=5,items=5000")
	if err != nil {
		t.Fatalf("failed to create mock store: %v", err)
	}
	datastore.SetDatastore(m)

	// Create systems store with mock m as the ActualStore
	s, err := NewDatastore(m, nil)
	if err != nil {
		t.Fatalf("failed to create system store: %v", err)
	}

	p, err := s.NamespaceById(datastore.SYSTEM_NAMESPACE)
	if err != nil {
		t.Fatalf("failed to get system namespace: %v", err)
	}

	pb, err := p.KeyspaceByName("namespaces")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	bb, err := p.KeyspaceByName("keyspaces")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	ib, err := p.KeyspaceByName("indexes")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	ui, err := p.KeyspaceByName("user_info")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	mui, err := p.KeyspaceByName("my_user_info")
	if err != nil {
		t.Fatalf("failed to get keyspace by name %v", err)
	}

	// Should be able to get a Value for UserInfo.
	v, err := s.UserInfo()
	if err != nil {
		t.Fatalf("failed to get stub user info: %v", err)
	}
	if v == nil {
		t.Fatalf("failed to get value for user info: nil")
	}

	// Expect count of 2 namespaces for the namespaces keyspace
	pb_c, err := pb.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || pb_c != 2 {
		t.Fatalf("failed to get expected namespaces keyspace count %v", err)
	}

	// Expect count of 10 for the keyspaces keyspace
	bb_c, err := bb.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || bb_c != 10 {
		t.Fatalf("failed to get expected keyspaces keyspace count %v", err)
	}

	// Expect count of 2 for the user_info keyspace
	ui_c, err := ui.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || ui_c != 2 {
		t.Fatalf("faied to get expect user_info keyspace count %v", err)

	}

	// Expect count of 2 for the my_user_info keyspace
	mui_c, err := mui.Count(&queryContextImpl{
		t: t,
		creds: &auth.Credentials{
			AuthenticatedUsers: auth.AuthenticatedUsers{"local:ivanivanov", "local:petrpetrov"},
		},
	})
	if err != nil || mui_c != 2 {
		t.Fatalf("failed to get expect my_user_info keyspace count %v", err)
	}

	// Expect count of 10 for the indexes keyspace (all the primary indexes)
	ib_c, err := ib.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || ib_c != 10 {
		t.Fatalf("failed to get expected indexes keyspace count %v %v", ib_c, err)
	}

	// Scan all Primary Index entries of the keyspaces keyspace
	bb_e, err := doPrimaryIndexScan(t, bb)

	// Check for expected and unexpected names:
	if !bb_e["p0/b1"] {
		t.Fatalf("failed to get expected keyspace name from index scan: p0/b1")
	}

	if bb_e["not a name"] {
		t.Fatalf("found unexpected name in index scan")
	}

	// Scan all Primary Index Entries of the user_info keyspace
	ui_e, err := doPrimaryIndexScan(t, ui)
	if err != nil {
		t.Fatalf("unable to scan index of system:user_info: %v", err)
	}
	if !ui_e["local:ivanivanov"] || !ui_e["local:petrpetrov"] {
		t.Fatalf("unexpected results from scan of syste:user_info: %v", ui_e)
	}

	// Scan Primary Index Entries of the my_user_info keyspace
	/*
		au := &datastore.AuthenticatedUsers{"ivanivanov"}
		mui_e, err := doPrimaryIndexScanForUsers(t, mui, *au)
		if err != nil {
			t.Fatalf("unable to scan index of system:my_user_info: %v", err)
		}
		if !mui_e["ivanivanov"] || mui_e["petrpetrov"] {
			t.Fatalf("unexpected results from scan of system:my_user_info: %v", ui_e)
		}
	*/

	// Scan all Primary Index entries of the indexes keyspace
	ib_e, err := doPrimaryIndexScan(t, ib)

	// Check for expected and unexpected names:
	if !ib_e["p1/b4/#primary"] {
		t.Fatalf("failed to get expected keyspace name from index scan: p1/b4/#primary")
	}

	if ib_e["p0/b4"] {
		t.Fatalf("found unexpected name in index scan")
	}

	// Fetch on the keyspaces keyspace - expect to find a value for this key:
	vals := make(map[string]value.AnnotatedValue, 1)
	key := "p0/b1"

	errs := bb.Fetch([]string{key}, vals, datastore.NULL_QUERY_CONTEXT, nil)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[key] == nil) {
		t.Fatalf("failed to fetch expected key from keyspaces keyspace")
	}

	// Fetch on the user_info keyspace - expect to find a value for this key:
	vals = make(map[string]value.AnnotatedValue, 1)
	key = "ivanivanov"
	errs = ui.Fetch([]string{key}, vals, datastore.NULL_QUERY_CONTEXT, nil)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[key] == nil) {
		t.Fatalf("failed to fetch expected key from keyspaces keyspace")
	}

	// Fetch on the indexes keyspace - expect to find a value for this key:
	vals = make(map[string]value.AnnotatedValue, 1)
	key = "p0/b1/#primary"
	errs = ib.Fetch([]string{key}, vals, datastore.NULL_QUERY_CONTEXT, nil)
	if errs != nil {
		t.Fatalf("errors in key fetch %v", errs)
	}

	if vals == nil || (len(vals) == 1 && vals[key] == nil) {
		t.Fatalf("failed to fetch expected key from indexes keyspace")
	}

	// Fetch on the keyspaces keyspace - expect to not find a value for this key:
	vals = make(map[string]value.AnnotatedValue, 1)
	key = "p0/b5"
	errs = bb.Fetch([]string{key}, vals, datastore.NULL_QUERY_CONTEXT, nil)
	if errs == nil {
		t.Fatalf("Expected not found error for key fetch on %s", "p0/b5")
	}

	if vals == nil || (len(vals) == 1 && vals[key] == nil) {
		t.Fatalf("Found unexpected key in keyspaces keyspace")
	}

}

// Helper function to perform a primary index scan on the given keyspace. Returns a map of
// all primary key names.
func doPrimaryIndexScan(t *testing.T, b datastore.Keyspace) (m map[string]bool, excp errors.Error) {
	//	conn := datastore.NewIndexConnection(&testingContext{t})
	conn := datastore.NewIndexConnection(&queryContextImpl{
		t: t,
		creds: &auth.Credentials{
			AuthenticatedUsers: auth.AuthenticatedUsers{"local:Administrator"},
		},
	})

	m = map[string]bool{}

	nitems, excp := b.Count(datastore.NULL_QUERY_CONTEXT)
	if excp != nil {
		t.Fatalf("failed to get keyspace count")
		return
	}

	indexers, excp := b.Indexers()
	if excp != nil {
		t.Fatalf("failed to retrieve indexers")
		return
	}

	pindexes, excp := indexers[0].PrimaryIndexes()
	if excp != nil || len(pindexes) < 1 {
		t.Fatalf("failed to retrieve primary indexes")
		return
	}

	idx := pindexes[0]
	go idx.ScanEntries("", nitems, datastore.UNBOUNDED, nil, conn)
	for {
		v, ok := conn.Sender().GetEntry()
		if !ok || v == nil {

			// Closed or Stopped
			return
		}

		m[v.PrimaryKey] = true
	}
}
