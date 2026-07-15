//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"net/http/httptest"
	"strings"
	"testing"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/server"
)

// completedNaturalResponse builds a minimal natural-language httpRequest whose
// generated statement is `statement`, drives it through CompletedNaturalRequest,
// and returns the raw JSON response body that a client would receive.
func completedNaturalResponse(natural, statement string) string {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", strings.NewReader(""))

	rv := &httpRequest{}
	rv.resp = rec
	rv.req = req
	rv.format = JSON
	server.NewBaseRequest(&rv.BaseRequest)

	// A generated statement is only emitted when both the user's natural
	// language input and the generated N1QL statement are present.
	rv.SetNatural(natural)
	rv.SetStatement(statement)

	// Use a large buffer so the whole response stays in memory (no mid-stream
	// flush) and is written out to the recorder by noMoreData().
	NewBufferedWriter(&rv.writer, rv, NewSyncPool(1<<20))

	rv.CompletedNaturalRequest(test_server.query_server)

	return rec.Body.String()
}

// TestGeneratedStatementNoUnicodeEscape is the regression test for MB-72778:
// characters such as '<', '>' and '&' in the generated_statement field must be
// emitted literally so the statement can be copy-pasted and run, rather than as
// their HTML-safe Unicode escapes ("<", ">", "&").
func TestGeneratedStatementNoUnicodeEscape(t *testing.T) {
	const natural = "show me test docs priced under 100 tagged a and b"
	const statement = "SELECT t.* FROM `test` AS t " +
		"WHERE t.price < 100 AND t.qty > 5 AND t.tag = \"a&b\""

	body := completedNaturalResponse(natural, statement)

	// The bug emits '<', '>' and '&' as their HTML-safe Unicode escapes. Assert
	// on the raw wire bytes: json.Unmarshal would decode the escapes back to the
	// literal characters, so a round-trip comparison alone cannot catch this.
	for _, esc := range []string{"\\u003c", "\\u003e", "\\u0026"} {
		if strings.Contains(body, esc) {
			t.Errorf("response contains Unicode escape %q; generated_statement "+
				"should use literal characters.\nbody: %s", esc, body)
		}
	}

	// Decode and verify the round-tripped value matches the statement exactly.
	var resp struct {
		GeneratedStatement string `json:"generated_statement"`
	}
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v\nbody: %s", err, body)
	}
	if resp.GeneratedStatement != statement {
		t.Errorf("generated_statement mismatch\n  expected: %s\n  actual:   %s",
			statement, resp.GeneratedStatement)
	}
}
