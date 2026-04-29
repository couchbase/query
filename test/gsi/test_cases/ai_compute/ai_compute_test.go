//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ai_compute

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	gsi "github.com/couchbase/query/test/gsi"
	"github.com/couchbase/query/value"
)

/*
Tests for the AI window aggregate functions: AI_COMPUTE and AI_RERANK.

AI_COMPUTE(docs, options, query?) OVER (PARTITION BY ...)

  - docs    (operand 0): expression whose values are collected across the partition.
  - options (operand 1): static object { uri, action, model, cred_id, header, batchSize }.
    action "rerank" is the only supported value today.
  - query   (operand 2): required when action is "rerank" — the rerank query text.

AI_RERANK(docs, options, query) OVER (PARTITION BY ...)

  - docs    (operand 0): expression whose values are collected across the partition.
  - options (operand 1): static object { uri, model, cred_id, header }.
  - query   (operand 2): string – the rerank query text.

ORDER BY is not allowed in the OVER clause for either function.
*/
func TestAiCompute(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	fmt.Println("Inserting values into Bucket for AI window aggregate tests")
	runMatch("insert.json", false, false, qc, t)

	runStmt(qc, "CREATE PRIMARY INDEX ON orders")

	// Error cases: mode-agnostic (same error regardless of prepared/explain),
	// so run once to avoid redundant triple execution.
	runMatch("case_ai_compute_error.json", false, false, qc, t)

	// Explain/plan cases: verify WindowAggregate appears in the query plan.
	// Run in all 3 modes to cover non-prepared, prepared, and explain paths.
	runMatch("case_ai_compute.json", false, false, qc, t) // non-prepared, no explain
	runMatch("case_ai_compute.json", true, false, qc, t)  // prepared, no explain
	runMatch("case_ai_compute.json", false, true, qc, t)  // non-prepared, with explain

	// Exercises the full HTTP execution path using an in-process mock server.
	testAiComputeSuccess(qc, t)
	testAiComputeMockErrors(qc, t)

	rr := runStmt(qc, "DELETE FROM orders WHERE test_id IN [\"ai_compute\", \"ai_rerank\"]")
	if rr.Err != nil {
		t.Errorf("did not expect err %s", rr.Err.Error())
	}

	runStmt(qc, "DROP PRIMARY INDEX ON orders")
}

// testAiComputeSuccess verifies that AI_COMPUTE and AI_RERANK successfully make
// an HTTP call and return numeric relevance scores for each row.
//
// It spins up an in-process mock HTTP server (net/http/httptest) that mimics a
// rerank API endpoint, so no external service, credentials, or network access is
// required.  The mock:
//
//   - Accepts POST requests with a JSON body containing a "documents" array.
//   - Returns one { "index": i, "relevance_score": <score> } entry per document,
//     with scores descending from fixedScores (0.9, 0.5, 0.1).
//
// To run against a real rerank endpoint instead of the mock, set AI_RERANK_URL
// before launching the test:
//
//	AI_RERANK_URL=https://your-host/rerank GSI_TEST=true go test ./...
//
// When AI_RERANK_URL is set the mock server is still started (so the defer close
// is safe), but the real URL is used for every N1QL statement.
func testAiComputeSuccess(qc *gsi.MockServer, t *testing.T) {
	t.Helper()

	// fixedScores maps document position (0-based within a partition) to a
	// relevance score returned by the mock.  Extend this slice if the test
	// data ever grows beyond 3 docs per partition.
	fixedScores := []float64{0.9, 0.5, 0.1}

	// Start the mock rerank HTTP server.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Guard: the window executor must set Content-Type: application/json.
		// If this check fails, a real API would return HTTP 415 — exactly the
		// bug this test was written to catch.
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w,
				fmt.Sprintf("mock: expected Content-Type: application/json, got %q", ct),
				http.StatusUnsupportedMediaType)
			return
		}

		// Decode the request body to find out how many documents were sent.
		// We must return exactly that many results; returning more would cause
		// an index-out-of-bounds panic inside the window execution code.
		var req struct {
			Documents []interface{} `json:"documents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "mock: bad request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Build one result per document with a fixed relevance score.
		results := make([]map[string]interface{}, len(req.Documents))
		for i := range req.Documents {
			score := 0.0
			if i < len(fixedScores) {
				score = fixedScores[i]
			}
			results[i] = map[string]interface{}{
				"index":           i,
				"relevance_score": score,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
	defer ts.Close()

	// Allow developers to target a real endpoint without changing code.
	rerankURL := ts.URL + "/rerank"
	if envURL := os.Getenv("AI_RERANK_URL"); envURL != "" {
		t.Logf("testAiComputeSuccess: using real endpoint AI_RERANK_URL=%s", envURL)
		rerankURL = envURL
	}

	// checkRows asserts that rr has exactly wantCount rows and that every row
	// contains a float64 "score" field.  label identifies the calling test in
	// failure messages.
	checkRows := func(label string, rr *gsi.RunResult, wantCount int) {
		t.Helper()
		if rr.Err != nil {
			t.Fatalf("%s: unexpected error: %v", label, rr.Err)
		}
		if len(rr.Results) != wantCount {
			t.Errorf("%s: expected %d results, got %d", label, wantCount, len(rr.Results))
		}
		for i, row := range rr.Results {
			m, ok := row.(map[string]interface{})
			if !ok {
				t.Errorf("%s result %d: expected map[string]interface{}, got %T", label, i, row)
				continue
			}
			score, exists := m["score"]
			if !exists {
				t.Errorf("%s result %d: missing 'score' field", label, i)
				continue
			}
			if _, isNum := score.(float64); !isNum {
				t.Errorf("%s result %d: expected float64 score, got %T (%v)", label, i, score, score)
			}
		}
	}

	// --- AI_RERANK success (literal config) ---------------------------------
	// Test data: 6 ai_rerank documents split across 2 PARTITION BY categories
	// (A: 3 docs, B: 3 docs).  Each partition triggers one HTTP call to the
	// mock, so the server receives 2 requests total.
	aiRerankQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER(PARTITION BY d.category) AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_rerank'`,
		rerankURL,
	)
	checkRows("AI_RERANK literal", runStmt(qc, aiRerankQ), 6)

	// --- AI_COMPUTE with action:'rerank' success (literal config) -----------
	// Exercises the two-hop code path: ai_compute → aiEvaluate → aiRerankEvaluate.
	// Same 2-partition layout applies (categories X and Y, 3 docs each).
	aiComputeQ := fmt.Sprintf(
		`SELECT AI_COMPUTE(d.text, {'action':'rerank', 'uri':'%s'}, 'database technology') `+
			`OVER(PARTITION BY d.category) AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_compute'`,
		rerankURL,
	)
	checkRows("AI_COMPUTE literal", runStmt(qc, aiComputeQ), 6)

	// --- Named parameter success tests ---------------------------------------
	// In practice, users pass the config as a named parameter ($cfg) rather than
	// embedding a literal object in SQL — it's cleaner and allows the URI/model
	// to be set per-request without recompiling the statement.
	//
	// Named parameters bypass the plan-time "static object" semantic check
	// (op.Value() returns nil for runtime-resolved params, so the check trivially
	// passes), making runtime type validation in setAiCompute/setAiRerank the
	// only gate.  These tests confirm the full execution path works end-to-end
	// when the config arrives via a named parameter.

	// AI_RERANK with named $cfg and $query
	aiRerankNamedRR := gsi.Run(qc, nil,
		`SELECT AI_RERANK(d.text, $cfg, $query) OVER(PARTITION BY d.category) AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_rerank'`,
		"default",
		map[string]value.Value{
			"query": value.NewValue("database technology"),
			"cfg":   value.NewValue(map[string]interface{}{"uri": rerankURL}),
		},
		nil, nil)
	checkRows("AI_RERANK named-param $cfg", aiRerankNamedRR, 6)

	// AI_COMPUTE with named $cfg (uri and action in config) and $query as third arg
	aiComputeNamedRR := gsi.Run(qc, nil,
		`SELECT AI_COMPUTE(d.text, $cfg, $query) OVER(PARTITION BY d.category) AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_compute'`,
		"default",
		map[string]value.Value{
			"cfg": value.NewValue(map[string]interface{}{
				"action": "rerank",
				"uri":    rerankURL,
			}),
			"query": value.NewValue("database technology"),
		},
		nil, nil)
	checkRows("AI_COMPUTE named-param $cfg", aiComputeNamedRR, 6)

	// --- NULL / MISSING document handling -----------------------------------
	// When d.text is NULL (explicit null stored in the document) or MISSING
	// (the field simply does not exist), CumulateInitial converts the value to
	// NULL_VALUE so that the array position is preserved.  The API therefore
	// receives a null element in the "documents" array and returns a score for
	// it just like any other position.  The row should still appear in the
	// result set with a numeric score — not an error and not skipped.
	//
	// We insert 3 temporary docs under test_id='ai_null_test':
	//   ai_null000 – text field is explicitly null
	//   ai_null001 – text field is absent (MISSING at evaluation time)
	//   ai_null002 – text field has a real string value
	// All 3 live in OVER() (no PARTITION BY), so one HTTP call is made with
	// docs=[null, null, "real text..."] and all 3 rows receive a score.
	runStmt(qc,
		`INSERT INTO orders VALUES
			('ai_null000', {'test_id':'ai_null_test','text':null}),
			('ai_null001', {'test_id':'ai_null_test'}),
			('ai_null002', {'test_id':'ai_null_test','text':'couchbase distributed database'})`)

	aiRerankNullQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER() AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_null_test'`,
		rerankURL,
	)
	checkRows("AI_RERANK NULL/MISSING docs", runStmt(qc, aiRerankNullQ), 3)

	// Clean up the null-test documents so they don't interfere with other tests.
	if rr := runStmt(qc, `DELETE FROM orders WHERE test_id = 'ai_null_test'`); rr.Err != nil {
		t.Errorf("cleanup ai_null_test: unexpected error: %v", rr.Err)
	}

	// --- Empty result set (zero rows matching WHERE) ------------------------
	// When the WHERE clause matches no documents, the window aggregate is never
	// entered and no HTTP call is made (the docs array is never populated).
	// The expected outcome is: 0 rows returned, no error.
	//
	// This test does not require the mock server to receive any request; it
	// exercises the short-circuit path inside the query engine itself.
	aiRerankEmptyQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER(PARTITION BY d.category) AS score `+
			`FROM orders AS d WHERE d.test_id = 'this_test_id_does_not_exist'`,
		rerankURL,
	)
	rrEmpty := runStmt(qc, aiRerankEmptyQ)
	if rrEmpty.Err != nil {
		t.Fatalf("AI_RERANK empty WHERE: unexpected error: %v", rrEmpty.Err)
	}
	if len(rrEmpty.Results) != 0 {
		t.Errorf("AI_RERANK empty WHERE: expected 0 results, got %d", len(rrEmpty.Results))
	}
}

// testAiComputeMockErrors exercises error paths in aiRerankEvaluate that require
// an HTTP server: out-of-bounds index in the API response and malformed JSON.
func testAiComputeMockErrors(qc *gsi.MockServer, t *testing.T) {
	t.Helper()

	// --- Out-of-bounds index in API response --------------------------------
	// The mock returns index == len(docs), which is one past the end of the
	// aiValues slice.  aiRerankEvaluate must detect this and return an error.
	oobServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Documents []interface{} `json:"documents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Return an index that is equal to len(docs) — just past the end.
		results := []map[string]interface{}{
			{"index": len(req.Documents), "relevance_score": 0.9},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
	defer oobServer.Close()

	oobQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER() AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_rerank'`,
		oobServer.URL+"/rerank",
	)
	rrOob := runStmt(qc, oobQ)
	if rrOob.Err == nil {
		t.Errorf("AI_RERANK out-of-bounds index: expected an error, got none")
	}

	// --- Malformed JSON response ---------------------------------------------
	// The mock returns a response body that is not valid JSON.  aiRerankEvaluate
	// must propagate the unmarshal error rather than silently producing nil scores.
	badJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`this is not valid json`))
	}))
	defer badJSONServer.Close()

	badJSONQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER() AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_rerank'`,
		badJSONServer.URL+"/rerank",
	)
	rrBad := runStmt(qc, badJSONQ)
	if rrBad.Err == nil {
		t.Errorf("AI_RERANK malformed JSON response: expected an error, got none")
	}

	// --- Missing relevance_score in API response ----------------------------
	// The mock returns results entries where the relevance_score field is absent.
	// aiRerankEvaluate must detect this and return an error rather than storing
	// a silently nil/NULL score.
	missingScoreServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Documents []interface{} `json:"documents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Return entries with index but no relevance_score field.
		results := make([]map[string]interface{}, len(req.Documents))
		for i := range req.Documents {
			results[i] = map[string]interface{}{"index": i}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": results})
	}))
	defer missingScoreServer.Close()

	missingScoreQ := fmt.Sprintf(
		`SELECT AI_RERANK(d.text, {'uri':'%s'}, 'database technology') `+
			`OVER() AS score `+
			`FROM orders AS d WHERE d.test_id = 'ai_rerank'`,
		missingScoreServer.URL+"/rerank",
	)
	rrMissing := runStmt(qc, missingScoreQ)
	if rrMissing.Err == nil {
		t.Errorf("AI_RERANK missing relevance_score: expected an error, got none")
	}
}
