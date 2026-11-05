//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
