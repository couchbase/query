//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/couchbase/gomemcached"
	memcached "github.com/couchbase/gomemcached/client"
	qerrors "github.com/couchbase/query/errors"
)

// build a document value with an XATTR section, as KV sends it
func xattrDoc(body string, pairs ...string) []byte {
	var sect []byte
	for i := 0; i+1 < len(pairs); i += 2 {
		p := append([]byte(pairs[i]), 0)
		p = append(p, pairs[i+1]...)
		p = append(p, 0)
		sect = binary.BigEndian.AppendUint32(sect, uint32(len(p)))
		sect = append(sect, p...)
	}
	rv := binary.BigEndian.AppendUint32(nil, uint32(len(sect)))
	rv = append(rv, sect...)
	return append(rv, body...)
}

func TestExtractXattrs(t *testing.T) {
	doc, xv, ok := ExtractXattrs(xattrDoc(`{"a":1}`, "_sync", `{"rev":2}`, "u", `"x"`))
	if !ok || string(doc) != `{"a":1}` || xv == nil {
		t.Fatalf("valid xattrs: doc %s, xattrs %v, ok %v", doc, xv, ok)
	}
	if v, _ := xv.Field("u"); v.Actual() != "x" {
		t.Errorf("xattr u = %v", v)
	}
	if v, _ := xv.Field("_sync"); v.ToString() != `{"rev":2}` {
		t.Errorf("xattr _sync = %v", v)
	}

	// an XATTR section and an array body
	if doc, _, ok := ExtractXattrs(xattrDoc(`[1,2]`, "k", `true`)); !ok || string(doc) != `[1,2]` {
		t.Errorf("array body: doc %s, ok %v", doc, ok)
	}

	// malformed input must not panic, and an invalid length leaves raw as it is
	for _, raw := range [][]byte{
		nil,
		{},
		[]byte(`[1`),
		[]byte(`"abc"`),
		[]byte(`[1,2,3]`),
		{0, 0, 0, 9, 0, 0, 0, 1},
	} {
		doc, xv, ok := ExtractXattrs(raw)
		if ok || xv != nil || string(doc) != string(raw) {
			t.Errorf("ExtractXattrs(%q) = %q, %v, %v", raw, doc, xv, ok)
		}
	}

	// a length that fits but pairs that don't: no panic, body after the section
	bad := []byte{0, 0, 0, 6, 0, 0, 0, 9, 'k', 'v', '{', '}'}
	if doc, _, ok := ExtractXattrs(bad); ok || string(doc) != `{}` {
		t.Errorf("truncated pair: doc %q, ok %v", doc, ok)
	}
	if doc, _, ok := ExtractXattrs([]byte{0, 0, 0, 2, 0, 0, '{', '}'}); ok || string(doc) != `{}` {
		t.Errorf("section shorter than a length field: doc %q, ok %v", doc, ok)
	}
	if doc, xv, ok := ExtractXattrs(xattrDoc(`{}`)); !ok || xv != nil || string(doc) != `{}` {
		t.Errorf("empty section: doc %q, xattrs %v, ok %v", doc, xv, ok)
	}
	noValue := xattrDoc(`{}`, "k", "")
	if _, _, ok := ExtractXattrs(noValue); ok {
		t.Errorf("empty xattr value accepted")
	}

	// a pair whose length field doesn't match its name and value
	for _, delta := range []int{-1, 1, 1 << 20} {
		raw := xattrDoc(`{}`, "k", `1`, "u", `2`)
		binary.BigEndian.PutUint32(raw[4:], uint32(int(binary.BigEndian.Uint32(raw[4:]))+delta))
		if doc, xv, ok := ExtractXattrs(raw); ok || xv != nil || string(doc) != `{}` {
			t.Errorf("pair length off by %d: doc %q, xattrs %v, ok %v", delta, doc, xv, ok)
		}
	}

	// a value without its 0 byte, and a partial length field after the last pair
	noNul := []byte{0, 0, 0, 7, 0, 0, 0, 3, 'k', 0, '1', '{', '}'}
	if doc, _, ok := ExtractXattrs(noNul); ok || string(doc) != `{}` {
		t.Errorf("value without 0 byte: doc %q, ok %v", doc, ok)
	}
	partial := xattrDoc(`{}`, "k", `1`)
	partial = append(append(append([]byte{}, partial[:len(partial)-2]...), 0, 0), '{', '}')
	binary.BigEndian.PutUint32(partial, binary.BigEndian.Uint32(partial)+2)
	if doc, _, ok := ExtractXattrs(partial); ok || string(doc) != `{}` {
		t.Errorf("partial length field: doc %q, ok %v", doc, ok)
	}
}

func TestOwnContext(t *testing.T) {
	dti := &memcached.DataTransferInfo{}
	shared := &memcached.ClientContext{CollId: 8, User: "u", DataTransferInfo: dti}
	other := &memcached.ClientContext{CollId: 9}
	in := []*memcached.ClientContext{shared, other}

	ctx, cc := ownContext(in)
	cc.Compressed = true
	cc.IncludeXATTRs = true
	if shared.Compressed || shared.IncludeXATTRs {
		t.Errorf("caller's context modified: %+v", shared)
	}
	if in[0] != shared || in[1] != other {
		t.Errorf("caller's slice modified")
	}
	if len(ctx) != 2 || ctx[0] != cc || ctx[1] != other {
		t.Errorf("returned context = %v", ctx)
	}
	if cc.CollId != 8 || cc.User != "u" || cc.DataTransferInfo != dti {
		t.Errorf("copy lost fields: %+v", cc)
	}

	ctx, cc = ownContext(nil)
	if len(ctx) != 1 || ctx[0] != cc || cc == nil {
		t.Errorf("ownContext(nil) = %v, %v", ctx, cc)
	}
}

// a real error must come first in the combined error, as callers ignore it when its text says KEY_ENOENT
func TestErrorCollectorOrder(t *testing.T) {
	notFound := &gomemcached.MCResponse{Status: gomemcached.KEY_ENOENT}
	failed := fmt.Errorf("bulk get of v-bucket 3 failed: boom")
	wrapped := fmt.Errorf("get failed: KEY_ENOENT") // not found, by its text only
	for _, in := range [][]error{{notFound, failed}, {failed, notFound}, {notFound, notFound, failed, notFound},
		{wrapped, failed}, {wrapped, notFound, failed}} {
		ech := make(chan error)
		eout := make(chan error, 2)
		status := &errorStatus{}
		go errorCollector(ech, eout, status)
		for _, e := range in {
			ech <- e
		}
		close(ech)
		err := <-eout
		if err == nil || strings.Contains(err.Error(), "KEY_ENOENT") || !strings.Contains(err.Error(), "boom") {
			t.Errorf("%v: got %v", in, err)
		}
		if m, ok := err.(multiError); !ok || len(m) != len(in) || len(m.Unwrap()) != len(in) {
			t.Errorf("%v: errors lost: %v", in, err)
		}
		if !status.errStatus.Load() {
			t.Errorf("%v: error status not set", in)
		}
		// isNotFoundError() in datastore/couchbase looks at every error through this
		if _, ok := err.(interface{ Unwrap() []error }); !ok {
			t.Errorf("%v: combined error can't be unwrapped", in)
		}
	}

	// not found keys alone don't stop the bulk get; a real error does
	for _, c := range []struct {
		in   []error
		stop bool
	}{{[]error{notFound, wrapped}, false}, {[]error{notFound, failed}, true}} {
		ech := make(chan error)
		eout := make(chan error, 2)
		status := &errorStatus{}
		go errorCollector(ech, eout, status)
		for _, e := range c.in {
			ech <- e
		}
		close(ech)
		err := <-eout
		if status.errStatus.Load() != c.stop {
			t.Errorf("%v: error status %v, expected %v", c.in, status.errStatus.Load(), c.stop)
		}
		if IsNotFoundError(err) == c.stop {
			t.Errorf("%v: IsNotFoundError = %v", c.in, !c.stop)
		}
	}
}

// the one not found check, used by bulk get to decide whether to stop and by fetch() to ignore errors
func TestIsNotFoundError(t *testing.T) {
	notFound := &gomemcached.MCResponse{Status: gomemcached.KEY_ENOENT}
	wrapped := fmt.Errorf("get failed: KEY_ENOENT")
	// only its cause says KEY_ENOENT, not its message (any query error with a cause will do)
	cause := qerrors.NewAWRInvalidSettingError("s", 1, nil, nil, fmt.Errorf("vb 3: KEY_ENOENT"))
	failed := fmt.Errorf("bulk get of v-bucket 3 failed: worker panic (see the log)")
	if strings.Contains(cause.Error(), "KEY_ENOENT") {
		t.Fatalf("test error's message mentions KEY_ENOENT: %v", cause)
	}
	for _, c := range []struct {
		err error
		exp bool
	}{
		{nil, false},
		{notFound, true},
		{wrapped, true},
		{cause, true},
		{failed, false},
		{multiError{notFound, wrapped, cause}, true},
		{multiError{notFound, failed}, false},
		{multiError{failed, notFound}, false},
		{multiError{}, false},
	} {
		if got := IsNotFoundError(c.err); got != c.exp {
			t.Errorf("IsNotFoundError(%v) = %v, expected %v", c.err, got, c.exp)
		}
	}

	// the combined error starts with a missing bucket (which makes it not retryable), then another real error, then a
	// not found key, whatever the arrival order
	bucketGone := fmt.Errorf("vb 3: %s", BUCKET_NOT_FOUND)
	retries := fmt.Errorf("bulkget exceeded MaxBulkRetries for b1(vbid:3)")
	for _, in := range [][]error{{notFound, failed, bucketGone}, {failed, bucketGone}, {notFound, failed},
		{cause, bucketGone}, {retries, notFound, bucketGone}, {notFound, retries}} {
		ech := make(chan error)
		eout := make(chan error, 2)
		go errorCollector(ech, eout, &errorStatus{})
		for _, e := range in {
			ech <- e
		}
		close(ech)
		m := (<-eout).(multiError)
		exp := in[len(in)-1]
		if m[0] != exp {
			t.Errorf("%v: first error %v, expected %v", in, m[0], exp)
		}
	}
}
