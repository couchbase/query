//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"strconv"
	"testing"
)

type testCache struct {
	value int
}

func TestCache(t *testing.T) {
	var names []string

	names = make([]string, 50)
	c := NewGenCache(100)

	// Add and Get
	for i := 1; i <= 50; i++ {
		v := testCache{value: i}
		id := strconv.Itoa(i)
		names[i-1] = id

		c.Add(v, id, nil)
		s := c.Size()
		if s != i {
			t.Errorf("Add test: expected %v elements, got %v", i, s)
		}
		vi := c.Get(id, nil)
		if vi == nil {
			t.Errorf("Add test: expected to find %v, got nothing", id)
		}
		v1, ok := vi.(testCache)
		if !ok {
			t.Errorf("Add test: invalid element type for %v", id)
		}
		if v1.value != i {
			t.Errorf("Add test: was expecting %v, read back %v", i, v1.value)
		}
	}

	// Delete
	sz := 49
	tgt := "25"
	r := c.Delete(tgt, nil)
	if !r {
		t.Errorf("Delete test: element not deleted %v", tgt)
	}
	s := c.Size()
	if s != sz {
		t.Errorf("Delete test: expected %v elements, got %v", sz, s)
	}
	r = c.Delete(tgt, nil)
	if r {
		t.Errorf("Delete test: deleted element deleted again %v", tgt)
	}
	s = c.Size()
	if s != sz {
		t.Errorf("Delete test: expected %v elements, got %v", sz, s)
	}

	// Update
	id := "50"
	v := testCache{value: 51}
	c.Add(v, id, nil)
	s = c.Size()
	if s != sz {
		t.Errorf("Update test: expected %v elements, got %v", sz, s)
	}
	vi := c.Get(id, nil)
	if vi == nil {
		t.Errorf("Update test: expected to find %v, got nothing", id)
	}
	v1, ok := vi.(testCache)
	if !ok {
		t.Errorf("Update test: invalid element type for %v", id)
	}
	if v1.value != 51 {
		t.Errorf("Update test: was expecting %v, read back %v", 51, v1.value)
	}

	// Names, Foreach
	n := c.Names()
	s = len(n)
	if s != sz {
		t.Errorf("Foreach test: expected %v elements, got %v", sz, s)
	}

	for i := 0; i < sz; i++ {
		iName, err := strconv.Atoi(n[i])

		if err != nil || iName < 1 || iName > 50 {
			t.Errorf("Update test: element name %v is not valid %v %v", n[i], i, n)
		} else if names[iName-1] == "" {
			t.Errorf("Update test: element name %v is duplicate", n[i])
		} else {
			names[iName-1] = ""
		}
		v := c.Get(n[i], nil)
		if v == nil {
			t.Errorf("Foreach test: expected to find %v, got nothing", n[i])
		}
	}

	c.SetLimit(sz)
}
