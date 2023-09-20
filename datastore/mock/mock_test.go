//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package mock

import (
	"strconv"
	"testing"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

func TestMock(t *testing.T) {
	s, err := NewDatastore("mock:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	if s.URL() != "mock:" {
		t.Fatalf("expected store URL to be same")
	}

	n, err := s.NamespaceIds()
	if err != nil || len(n) != DEFAULT_NUM_NAMESPACES {
		t.Fatalf("expected num namespaces to be same")
	}

	n, err = s.NamespaceNames()
	if err != nil || len(n) != DEFAULT_NUM_NAMESPACES {
		t.Fatalf("expected num namespaces to be same")
	}

	p, err := s.NamespaceById("not-a-namespace")
	if err == nil || p != nil {
		t.Fatalf("expected not-a-namespace")
	}

	p, err = s.NamespaceByName("not-a-namespace")
	if err == nil || p != nil {
		t.Fatalf("expected not-a-namespace")
	}

	p, err = s.NamespaceById("p0")
	if err != nil || p == nil {
		t.Fatalf("expected namespace p0")
	}

	if p.Id() != "p0" {
		t.Fatalf("expected p0 id")
	}

	p, err = s.NamespaceByName("p0")
	if err != nil || p == nil {
		t.Fatalf("expected namespace p0")
	}

	if p.Name() != "p0" {
		t.Fatalf("expected p0 name")
	}

	n, err = p.KeyspaceIds()
	if err != nil || len(n) != DEFAULT_NUM_KEYSPACES {
		t.Fatalf("expected num keyspaces to be same")
	}

	n, err = p.KeyspaceNames()
	if err != nil || len(n) != DEFAULT_NUM_KEYSPACES {
		t.Fatalf("expected num keyspaces to be same")
	}

	b, err := p.KeyspaceById("not-a-keyspace")
	if err == nil || b != nil {
		t.Fatalf("expected not-a-keyspace")
	}

	b, err = p.KeyspaceByName("not-a-keyspace")
	if err == nil || b != nil {
		t.Fatalf("expected not-a-keyspace")
	}

	b, err = p.KeyspaceById("b0")
	if err != nil || b == nil {
		t.Fatalf("expected keyspace b0")
	}

	if b.Id() != "b0" {
		t.Fatalf("expected b0 id")
	}

	b, err = p.KeyspaceByName("b0")
	if err != nil || b == nil {
		t.Fatalf("expected keyspace b0")
	}

	if b.Name() != "b0" {
		t.Fatalf("expected b0 name")
	}

	c, err := b.Count(datastore.NULL_QUERY_CONTEXT)
	if err != nil || c != int64(DEFAULT_NUM_ITEMS) {
		t.Fatalf("expected num items")
	}

	f := []string{"123"}
	vs := make(map[string]value.AnnotatedValue, 1)
	errs := b.Fetch(f, vs, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	if errs != nil || len(vs) == 0 {
		t.Fatalf("expected item 123")
	}

	v := vs[f[0]]
	x, has_x := v.Field("id")
	if has_x != true || x == nil {
		t.Fatalf("expected item.id")
	}

	x, has_x = v.Field("i")
	if has_x != true || x == nil {
		t.Fatalf("expected item.i")
	}

	x, has_x = v.Field("not-a-valid-path")
	if has_x == true {
		t.Fatalf("expected not-a-valid-path to err")
	}

	f = []string{"not-an-item"}
	vs = make(map[string]value.AnnotatedValue, 1)
	errs = b.Fetch(f, vs, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	if errs == nil || len(vs) > 0 {
		t.Fatalf("expected not-an-item")
	}

	f = []string{strconv.Itoa(DEFAULT_NUM_ITEMS)}
	vs = make(map[string]value.AnnotatedValue, 1)
	errs = b.Fetch(f, vs, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	if errs == nil || len(vs) > 0 {
		t.Fatalf("expected not-an-item")
	}

}

func TestMockIndex(t *testing.T) {
	s, err := NewDatastore("mock:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	p, err := s.NamespaceById("p0")
	if err != nil || p == nil {
		t.Fatalf("expected namespace p0")
	}

	b, err := p.KeyspaceById("b0")
	if err != nil || b == nil {
		t.Fatalf("expected keyspace b0")
	}

	// Do a scan from keys 4 to 6 with Inclusion set to NEITHER - expect 1 result with key 5
	lo := []value.Value{value.NewValue("4")}
	hi := []value.Value{value.NewValue("6")}
	span := &datastore.Span{Range: datastore.Range{Inclusion: datastore.NEITHER, Low: lo, High: hi}}
	items, err := doIndexScan(t, b, span)

	if err != nil {
		t.Fatalf("unexpected error in scan: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("unexpected number of items in scan: %d", len(items))
	}

	if items[0].PrimaryKey != "5" {
		t.Fatalf("unexpected key in result: %v", items[0].PrimaryKey)
	}

	// Do a scan from keys 4 to 6 with Inclusion set to BOTH - expect 3 results
	span.Range.Inclusion = datastore.BOTH
	items, err = doIndexScan(t, b, span)

	if err != nil {
		t.Fatalf("unexpected error in scan: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("unexpected number of items in scan: %d", len(items))
	}

	// Do a scan with incorrect range type - expect scan error
	span.Range.Low = []value.Value{value.NewValue(4.0)}
	items, err = doIndexScan(t, b, span)
}

type testingContext struct {
	t *testing.T
}

func (this *testingContext) GetScanCap() int64 {
	return 16
}

func (this *testingContext) MaxParallelism() int {
	return 1
}

func (this *testingContext) Error(err errors.Error) {
	this.t.Logf("Scan error: %v", err)
}

func (this *testingContext) Warning(wrn errors.Error) {
	this.t.Logf("scan warning: %v", wrn)
}

func (this *testingContext) Fatal(fatal errors.Error) {
	this.t.Logf("scan fatal: %v", fatal)
}

func (this *testingContext) GetReqDeadline() time.Time {
	return time.Time{}
}

func (this *testingContext) TenantCtx() tenant.Context {
	return nil
}

func (this *testingContext) FirstCreds() (string, bool) {
	return "", true
}

func (this *testingContext) SetFirstCreds(string) {
}

func (this *testingContext) RecordFtsRU(ru tenant.Unit) {
}

func (this *testingContext) RecordGsiRU(ru tenant.Unit) {
}

func (this *testingContext) RecordKvRU(ru tenant.Unit) {
}

func (this *testingContext) RecordKvWU(wu tenant.Unit) {
}

func (this *testingContext) Credentials() *auth.Credentials {
	return nil
}

func (this *testingContext) SkipKey(key string) bool {
	return false
}

// Helper function to scan the primary index of given keyspace with given span
func doIndexScan(t *testing.T, b datastore.Keyspace, span *datastore.Span) (
	e []*datastore.IndexEntry, excp errors.Error) {
	conn := datastore.NewIndexConnection(&testingContext{t})
	e = []*datastore.IndexEntry{}

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

	idx, excp := indexers[0].IndexByName("#primary")
	if excp != nil {
		t.Fatalf("failed to retrieve primary index")
		return
	}

	go idx.Scan("", span, false, nitems, datastore.UNBOUNDED, nil, conn)

	for {
		entry, ok := conn.Sender().GetEntry()
		if entry == nil || !ok {
			return
		}

		e = append(e, entry)
	}

	return
}
