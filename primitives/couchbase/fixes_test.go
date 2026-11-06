//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"sort"
	"testing"
	"unsafe"
)

func TestScanCancelSliceSort(t *testing.T) {
	b1 := &Bucket{Name: "b1"}
	b2 := &Bucket{Name: "b2"}
	cr := scanCancelSlice{
		{vbucket: 7, b: b2},
		{vbucket: 3, b: b1},
		{vbucket: 9, b: b1},
		{vbucket: 1, b: b2},
		{vbucket: 3, b: b1},
	}
	sort.Sort(cr)
	expected := []struct {
		name string
		vb   uint16
	}{{"b1", 3}, {"b1", 3}, {"b1", 9}, {"b2", 1}, {"b2", 7}}
	for i, e := range expected {
		if cr[i].b.Name != e.name || cr[i].vbucket != e.vb {
			t.Errorf("position %d: got %s/%d, expected %s/%d", i, cr[i].b.Name, cr[i].vbucket, e.name, e.vb)
		}
	}
}

func TestGroupsToParamFormat(t *testing.T) {
	cases := []struct {
		groups   []string
		expected string
	}{
		{[]string{"g1"}, "g1"},
		{[]string{"g1", "g2", "g3"}, "g1,g2,g3"},
		{[]string{"", "g1", "g2"}, "g1,g2"},
		{[]string{"g1", "", "g2", ""}, "g1,g2"},
		{[]string{"", ""}, ""},
	}
	for _, c := range cases {
		if got := groupsToParamFormat(c.groups); got != c.expected {
			t.Errorf("%q: got %q, expected %q", c.groups, got, c.expected)
		}
	}
}

func TestGetConnectionToVBucketBounds(t *testing.T) {
	b := &Bucket{vBucketServerMap: unsafe.Pointer(&VBucketServerMap{
		VBucketMap: [][]int{{-1}, {-1}, {-1}, {-1}},
	})}
	desc := &doDescriptor{}
	// vbucket == len(VBucketMap) must be rejected, not index past the end
	for _, vb := range []uint32{4, 5, 1024} {
		conn, pool, err := b.getConnectionToVBucket(vb, desc)
		if err == nil || conn != nil || pool != nil {
			t.Errorf("vb %d: expected an error, got conn %v pool %v err %v", vb, conn, pool, err)
		}
	}
	// last valid vbucket is looked up (and fails here only because it has no master)
	_, _, err := b.getConnectionToVBucket(3, desc)
	if err == nil || err.Error() != "primitives/couchbase: No master for vbucket 3" {
		t.Errorf("vb 3: unexpected error %v", err)
	}
}

func TestReadCurrentReusesHead(t *testing.T) {
	// keys "aaaa", "b", "cccccc" packed in the buffer; keys holds the cumulative end offsets
	this := &vbRangeScan{
		buffer: []byte("aaaabcccccc"),
		keys:   []uint32{4, 5, 11},
		head:   make([]byte, 0, 8),
	}
	head := unsafe.SliceData(this.head)
	for i, e := range []string{"aaaa", "b", "cccccc"} {
		this.currentKey = i
		if !this.readCurrent() {
			t.Fatalf("key %d: readCurrent failed", i)
		}
		if string(this.current()) != e {
			t.Errorf("key %d: got %q, expected %q", i, this.current(), e)
		}
		// a key that fits the capacity must not reallocate, even if longer than the previous key
		if unsafe.SliceData(this.head) != head {
			t.Errorf("key %d: head reallocated", i)
		}
	}
	this.currentKey = 3
	if this.readCurrent() {
		t.Errorf("readCurrent past the last key succeeded")
	}
}
