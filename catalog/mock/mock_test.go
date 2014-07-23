//  Copyright (c) 2013 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package mock

import (
	"strconv"
	"testing"

	"github.com/couchbaselabs/query/catalog"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/value"
)

func TestMock(t *testing.T) {
	s, err := NewDatastore("mock:")
	if err != nil {
		t.Errorf("failed to create datastore: %v", err)
	}
	if s.URL() != "mock:" {
		t.Errorf("expected datastore URL to be same")
	}

	n, err := s.NamespaceIds()
	if err != nil || len(n) != DEFAULT_NUM_NAMESPACES {
		t.Errorf("expected num namespaces to be same")
	}

	n, err = s.NamespaceNames()
	if err != nil || len(n) != DEFAULT_NUM_NAMESPACES {
		t.Errorf("expected num namespaces to be same")
	}

	p, err := s.NamespaceById("not-a-namespace")
	if err == nil || p != nil {
		t.Errorf("expected not-a-namespace")
	}

	p, err = s.NamespaceByName("not-a-namespace")
	if err == nil || p != nil {
		t.Errorf("expected not-a-namespace")
	}

	p, err = s.NamespaceById("p0")
	if err != nil || p == nil {
		t.Errorf("expected namespace p0")
	}

	if p.Id() != "p0" {
		t.Errorf("expected p0 id")
	}

	p, err = s.NamespaceByName("p0")
	if err != nil || p == nil {
		t.Errorf("expected namespace p0")
	}

	if p.Name() != "p0" {
		t.Errorf("expected p0 name")
	}

	n, err = p.KeyspaceIds()
	if err != nil || len(n) != DEFAULT_NUM_KEYSPACES {
		t.Errorf("expected num keyspaces to be same")
	}

	n, err = p.KeyspaceNames()
	if err != nil || len(n) != DEFAULT_NUM_KEYSPACES {
		t.Errorf("expected num keyspaces to be same")
	}

	b, err := p.KeyspaceById("not-a-keyspace")
	if err == nil || b != nil {
		t.Errorf("expected not-a-keyspace")
	}

	b, err = p.KeyspaceByName("not-a-keyspace")
	if err == nil || b != nil {
		t.Errorf("expected not-a-keyspace")
	}

	b, err = p.KeyspaceById("b0")
	if err != nil || b == nil {
		t.Errorf("expected keyspace b0")
	}

	if b.Id() != "b0" {
		t.Errorf("expected b0 id")
	}

	b, err = p.KeyspaceByName("b0")
	if err != nil || b == nil {
		t.Errorf("expected keyspace b0")
	}

	if b.Name() != "b0" {
		t.Errorf("expected b0 name")
	}

	c, err := b.Count()
	if err != nil || c != int64(DEFAULT_NUM_ITEMS) {
		t.Errorf("expected num items")
	}

	f := []string{"123"}
	vs, err := b.Fetch(f)
	if err != nil || vs == nil {
		t.Errorf("expected item 123")
	}

	v := vs[0].Value
	x, has_x := v.Field("id")
	if has_x != true || x == nil {
		t.Errorf("expected item.id")
	}

	x, has_x = v.Field("i")
	if has_x != true || x == nil {
		t.Errorf("expected item.i")
	}

	x, has_x = v.Field("not-a-valid-path")
	if has_x == true {
		t.Errorf("expected not-a-valid-path to err")
	}

	v, err = b.FetchOne("not-an-item")
	if err == nil || v != nil {
		t.Errorf("expected not-an-item")
	}

	v, err = b.FetchOne(strconv.Itoa(DEFAULT_NUM_ITEMS))
	if err == nil || v != nil {
		t.Errorf("expected not-an-item")
	}

}

func TestMockIndex(t *testing.T) {
	s, err := NewDatastore("mock:")
	if err != nil {
		t.Errorf("failed to create datastore: %v", err)
	}

	p, err := s.NamespaceById("p0")
	if err != nil || p == nil {
		t.Errorf("expected namespace p0")
	}

	b, err := p.KeyspaceById("b0")
	if err != nil || b == nil {
		t.Errorf("expected keyspace b0")
	}

	// Do a scan from keys 4 to 6 with Inclusion set to NEITHER - expect 1 result with key 5
	lo := []value.Value{value.NewValue("4")}
	hi := []value.Value{value.NewValue("6")}
	span := catalog.Span{Range: &catalog.Range{Inclusion: catalog.NEITHER, Low: lo, High: hi}}
	items, err := doIndexScan(t, b, span)

	if err != nil {
		t.Errorf("unexpected error in scan: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("unexpected number of items in scan: %d", len(items))
	}

	if items[0].PrimaryKey != "5" {
		t.Errorf("unexpected key in result: %v", items[0].PrimaryKey)
	}

	// Do a scan from keys 4 to 6 with Inclusion set to BOTH - expect 3 results
	span.Range.Inclusion = catalog.BOTH
	items, err = doIndexScan(t, b, span)

	if err != nil {
		t.Errorf("unexpected error in scan: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("unexpected number of items in scan: %d", len(items))
	}

	// Do a scan with incorrect range type - expect scan error
	span.Range.Low = []value.Value{value.NewValue(4.0)}
	items, err = doIndexScan(t, b, span)

	if err == nil {
		t.Errorf("Expected error in scan")
	}

	expected_error := "Invalid lower bound 4 of type float64."
	if errors.Error() != expected_error {
		t.Errorf("Unexpected error message %s (expected %s)", errors.Error(), expected_error)
	}

}

// Helper function to scan the all_docs index of given keyspace with given span
func doIndexScan(t *testing.T, b catalog.Keyspace, span catalog.Span) (e []*catalog.IndexEntry, excp errors.Error) {
	warnChan := make(errors.ErrorChannel)
	errChan := make(errors.ErrorChannel)
	defer close(warnChan)
	defer close(errChan)
	conn := catalog.NewIndexConnection(warnChan, errChan)

	e = []*catalog.IndexEntry{}

	nitems, excp := b.Count()
	if excp != nil {
		t.Errorf("failed to get keyspace count")
		return
	}

	idx, excp := b.IndexByName("all_docs")
	if excp != nil {
		t.Errorf("failed to retrieve all_docs index")
		return
	}

	// go Scan all_docs index with given span
	go idx.Scan(&span, nitems, conn)
	for {
		select {
		case v, conn_open := <-conn.EntryChannel():
			if !conn_open {
				// Channel closed => Scan complete
				return
			}
			e = append(e, v)
		case _excp, _ := <-errChan:
			excp = _excp
			return
		}
	}
}
