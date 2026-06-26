/*
Copyright 2026-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

// Hermetic end-to-end tests for the natural-language path that exercise the
// ai_gateway against an in-process fake LLM instead of a live provider. The
// "slm" provider is used because it speaks the OpenAI wire format, performs no
// moderation call, takes a caller-supplied endpoint and needs no api_key - so a
// single POST reaches the fake server and the whole round-trip is deterministic.
// The mock query server allows every outbound URL (all_access allowlist), so
// the gateway's cluster-allowlist gate passes for the httptest URL.

package natural

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

// writeChatCompletion writes an OpenAI-compatible chat-completions success body
// carrying the given assistant content and canned token usage.
func writeChatCompletion(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"choices": []map[string]interface{}{
			{"message": map[string]interface{}{"content": content}},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     11,
			"completion_tokens": 7,
			"total_tokens":      18,
		},
	})
}

// slmConfig builds a natural_config object targeting the fake server via the slm
// provider.
func slmConfig(endpoint string) map[string]interface{} {
	return map[string]interface{}{
		"provider": "slm",
		"endpoint": endpoint,
		"model":    "test-model",
	}
}

// runNatural runs a natural-language statement with the given natural_config.
func runNatural(qc *gsi.MockServer, stmt string, cfg map[string]interface{}) *gsi.RunResult {
	qp := map[string]interface{}{"natural_config": cfg}
	return gsi.Run(qc, qp, stmt, gsi.Namespace_CBS, nil, nil, nil)
}

// beginChat opens a conversation over the given keyspace context. The
// server-minted conversation id is returned in the result's ChatId field.
func beginChat(qc *gsi.MockServer, keyspace string) *gsi.RunResult {
	return beginChatAs(qc, keyspace, nil)
}

// beginChatAs opens a conversation as a specific user (nil uses the default
// all-users creds). Used to exercise per-user chat ownership: the chat is
// populated with the opening user's authenticated users list.
func beginChatAs(qc *gsi.MockServer, keyspace string, userArgs map[string]string) *gsi.RunResult {
	stmt := fmt.Sprintf(`BEGIN CHAT WITH {"keyspaces":[%q]}`, keyspace)
	return gsi.Run(qc, nil, stmt, gsi.Namespace_CBS, nil, nil, userArgs)
}

// runNaturalChat runs a conversational natural-language turn against an existing
// chat id, optionally as a specific user (nil uses the default all-users creds).
func runNaturalChat(qc *gsi.MockServer, stmt string, cfg map[string]interface{}, chatId string,
	userArgs map[string]string) *gsi.RunResult {
	qp := map[string]interface{}{"natural_config": cfg, "natural_chatid": chatId}
	return gsi.Run(qc, qp, stmt, gsi.Namespace_CBS, nil, nil, userArgs)
}

// testNaturalFakeProvider drives the gateway end-to-end against a fake LLM.
func testNaturalFakeProvider(qc *gsi.MockServer, t *testing.T) {
	testNaturalHappyPath(qc, t)
	testNaturalTokenReporting(qc, t)
	testNaturalModelRefusal(qc, t)
	testNaturalGatewayErrorNotRetried(qc, t)
	testNaturalConversationalHappyPath(qc, t)
	testNaturalConversationalNoSuchChat(qc, t)
	testNaturalConversationalWrongUser(qc, t)
	testNaturalConfigInWithClause(qc, t)
	testNaturalWithOptionNotAllowed(qc, t)
}

// testNaturalHappyPath verifies the full round-trip: the gateway posts to the
// fake server, the completion is unwrapped from its markdown fence, and the
// generated statement is produced. "execute":false keeps the assertion on the
// generation path (the gateway) rather than on query execution.
func testNaturalHappyPath(qc *gsi.MockServer, t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(w, "```sql\nSELECT name FROM orders\n```")
	}))
	defer srv.Close()

	rr := runNatural(qc, `USING AI WITH {"keyspaces":"orders", "execute":false} list the character names`,
		slmConfig(srv.URL))
	if rr.Err != nil {
		t.Fatalf("happy path: unexpected error: %v", rr.Err)
	}
	if !rr.GeneratedStatement {
		t.Errorf("happy path: expected a generated statement")
	}
}

// testNaturalTokenReporting verifies that the LLM token usage returned by the
// provider is propagated into the response as requestTokens. The fake server
// returns a fixed usage block (prompt 11 / completion 7 / total 18); a
// single-shot generation makes exactly one completion call, so the reported
// counts must equal that block. This is an independent (non-chat) request, so
// the usage surfaces as requestTokens and chatTokens is absent. Regression guard
// for the direct path's per-request token reporting.
func testNaturalTokenReporting(qc *gsi.MockServer, t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(w, "```sql\nSELECT name FROM orders\n```")
	}))
	defer srv.Close()

	rr := runNatural(qc, `USING AI WITH {"keyspaces":"orders", "execute":false} list the character names`,
		slmConfig(srv.URL))
	if rr.Err != nil {
		t.Fatalf("token reporting: unexpected error: %v", rr.Err)
	}
	if rr.RequestTokens == nil {
		t.Fatalf("token reporting: expected requestTokens to be reported, got nil")
	}
	if rr.ChatTokens != nil {
		t.Fatalf("token reporting: expected no chatTokens on an independent request, got %v", rr.ChatTokens)
	}
	for _, want := range []struct {
		field string
		count int
	}{{"promptTokens", 11}, {"completionTokens", 7}, {"totalTokens", 18}} {
		if got := rr.RequestTokens[want.field]; got != want.count {
			t.Errorf("token reporting: %s = %v, want %d", want.field, got, want.count)
		}
	}
}

// testNaturalModelRefusal verifies that a #ERR response (the model declaring it
// cannot answer) surfaces as E_NL_ERR_CHATCOMPLETIONS_RESP rather than being
// retried or reported as a generic failure.
func testNaturalModelRefusal(qc *gsi.MockServer, t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(w, "#ERR: the request cannot be answered from this schema")
	}))
	defer srv.Close()

	rr := runNatural(qc, `USING AI WITH {"keyspaces":"orders"} do something impossible`,
		slmConfig(srv.URL))
	if rr.Err == nil {
		t.Fatalf("model refusal: expected an error, got none")
	}
	if rr.Err.Code() != errors.E_NL_ERR_CHATCOMPLETIONS_RESP {
		t.Errorf("model refusal: got code %d, want %d (%v)",
			rr.Err.Code(), errors.E_NL_ERR_CHATCOMPLETIONS_RESP, rr.Err)
	}
}

// testNaturalGatewayErrorNotRetried is the regression test for the correction
// loop: a gateway/transport failure that occurs *during* a correction retry must
// be surfaced immediately, not fed back to the model as if it were a bad
// statement and re-sent until the retry budget is exhausted.
//
// The fake server returns an unparseable statement on the first call (which
// drives the request into the correction loop) and HTTP 401 on every call
// after. The request must fail with the provider's request-failed error
// (E_NL_CHATCOMPLETIONS_REQ_FAILED), NOT the generic E_NL_FAIL_GENERATED_STMT
// that results from exhausting the correction retries.
func testNaturalGatewayErrorNotRetried(qc *gsi.MockServer, t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			// Syntactically invalid SQL: passes content extraction but fails the
			// parse, sending the request into the correction loop.
			writeChatCompletion(w, "SELECT ((( FROM orders")
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	rr := runNatural(qc, `USING AI WITH {"keyspaces":"orders"} list the character names`,
		slmConfig(srv.URL))
	if rr.Err == nil {
		t.Fatalf("gateway-error-in-retry: expected an error, got none")
	}
	if rr.Err.Code() != errors.E_NL_CHATCOMPLETIONS_REQ_FAILED {
		t.Errorf("gateway-error-in-retry: got code %d, want %d (a gateway error must abort the "+
			"correction loop immediately rather than surfacing as E_NL_FAIL_GENERATED_STMT); err: %v",
			rr.Err.Code(), errors.E_NL_CHATCOMPLETIONS_REQ_FAILED, rr.Err)
	}
}

// testNaturalConversationalHappyPath is the multi-turn round-trip on the direct
// path: BEGIN CHAT opens a conversation and mints an id, then a follow-up USING
// AI request carrying that natural_chatid is routed through the conversational
// path (ProcessDirectConversationalRequest) and generates a statement against
// the fake LLM. "execute":false keeps the assertion on generation, not query
// execution.
func testNaturalConversationalHappyPath(qc *gsi.MockServer, t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(w, "```sql\nSELECT name FROM orders\n```")
	}))
	defer srv.Close()

	begin := beginChat(qc, "orders")
	if begin.Err != nil {
		t.Fatalf("conversational: BEGIN CHAT failed: %v", begin.Err)
	}
	if begin.ChatId == "" {
		t.Fatalf("conversational: BEGIN CHAT returned an empty chat id")
	}

	rr := runNaturalChat(qc, `USING AI WITH {"keyspaces":"orders", "execute":false} list the character names`,
		slmConfig(srv.URL), begin.ChatId, nil)
	if rr.Err != nil {
		t.Fatalf("conversational: follow-up turn failed: %v", rr.Err)
	}
	if !rr.GeneratedStatement {
		t.Errorf("conversational: expected a generated statement on the follow-up turn")
	}
}

// testNaturalConversationalNoSuchChat verifies that a follow-up turn naming a
// chat id that was never opened fails with E_NL_NO_SUCH_CHAT. The lookup happens
// before provider resolution or any LLM call, so the endpoint is never hit.
func testNaturalConversationalNoSuchChat(qc *gsi.MockServer, t *testing.T) {
	rr := runNaturalChat(qc, `USING AI WITH {"keyspaces":"orders"} list the character names`,
		slmConfig("http://127.0.0.1:1"), "no-such-chat-id", nil)
	if rr.Err == nil {
		t.Fatalf("no-such-chat: expected an error, got none")
	}
	if rr.Err.Code() != errors.E_NL_NO_SUCH_CHAT {
		t.Errorf("no-such-chat: got code %d, want %d (%v)",
			rr.Err.Code(), errors.E_NL_NO_SUCH_CHAT, rr.Err)
	}
}

// testNaturalConversationalWrongUser verifies that a follow-up turn issued by a
// user who is not among the chat's owners is rejected with E_NL_CHAT_WRONG_USER.
// The ownership check happens before any LLM call. The chat is opened by a
// single owner (ordersowner); the follow-up runs as a different, non-member
// user (reviewowner), so the multi-user CheckUser rejects it.
func testNaturalConversationalWrongUser(qc *gsi.MockServer, t *testing.T) {
	begin := beginChatAs(qc, "orders", map[string]string{"ordersowner": "orderspass"})
	if begin.Err != nil || begin.ChatId == "" {
		t.Fatalf("wrong-user: BEGIN CHAT failed: %v", begin.Err)
	}

	rr := runNaturalChat(qc, `USING AI WITH {"keyspaces":"orders"} list the character names`,
		slmConfig("http://127.0.0.1:1"), begin.ChatId, map[string]string{"reviewowner": "reviewpass"})
	if rr.Err == nil {
		t.Fatalf("wrong-user: expected an error, got none")
	}
	if rr.Err.Code() != errors.E_NL_CHAT_WRONG_USER {
		t.Errorf("wrong-user: got code %d, want %d (%v)",
			rr.Err.Code(), errors.E_NL_CHAT_WRONG_USER, rr.Err)
	}
}

// testNaturalConfigInWithClause verifies natural_config supplied inside the
// USING AI WITH clause (rather than as the natural_config request parameter)
// drives the direct provider path end-to-end. Mirrors the happy path but with
// config carried in the statement; no natural_config query parameter is set.
func testNaturalConfigInWithClause(qc *gsi.MockServer, t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeChatCompletion(w, "```sql\nSELECT name FROM orders\n```")
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(slmConfig(srv.URL))
	stmt := fmt.Sprintf(`USING AI WITH {"keyspaces":"orders","execute":false,"config":%s} list the character names`, cfg)
	rr := gsi.Run(qc, nil, stmt, gsi.Namespace_CBS, nil, nil, nil)
	if rr.Err != nil {
		t.Fatalf("config-in-with: unexpected error: %v", rr.Err)
	}
	if !rr.GeneratedStatement {
		t.Errorf("config-in-with: expected a generated statement")
	}
}

// testNaturalWithOptionNotAllowed verifies the shared WITH-clause option parser
// rejects a key a statement does not accept with E_NL_OPTION_NOT_ALLOWED.
// "config" is valid on USING AI/PAUSE CHAT but not on END CHAT; the rejection
// happens during statement parsing, before any chat lookup or LLM call, so a
// bogus chat id never matters.
func testNaturalWithOptionNotAllowed(qc *gsi.MockServer, t *testing.T) {
	stmt := `END CHAT WITH {"chatId":"some-chat","config":{"provider":"slm","endpoint":"http://127.0.0.1:1","model":"m"}}`
	rr := gsi.Run(qc, nil, stmt, gsi.Namespace_CBS, nil, nil, nil)
	if rr.Err == nil {
		t.Fatalf("option-not-allowed: expected an error, got none")
	}
	if rr.Err.Code() != errors.E_NL_OPTION_NOT_ALLOWED {
		t.Errorf("option-not-allowed: got code %d, want %d (%v)",
			rr.Err.Code(), errors.E_NL_OPTION_NOT_ALLOWED, rr.Err)
	}
}
