//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// Tests for nlquery.go - the pure prompt-construction and response-parsing
// helpers.
//
// Strategy:
//   - Template filling, keyspace listing, prompt builders, fence stripping and
//     the INFER schema/samples extraction are pure functions driven directly.
//   - INFER results are hand-built value.Values shaped like the inferencer's
//     output; algebra.Paths are built with NewPathLong. No datastore is needed.
//   - The samples privacy invariant (sample values never marshaled into the
//     persisted prompt/chat document) is asserted with sentinel values.
//   - ProcessRequest / ProcessConversationalRequest need a datastore, an
//     inferencer and a provider; they are covered by the integration harness
//     under test/gsi, not here.

package natural

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/natural/ai_gateway"
	"github.com/couchbase/query/value"
)

// ─── fillTemplate ─────────────────────────────────────────────────────────────

func TestFillTemplate(t *testing.T) {
	got := directFillTemplate("Q: {nl} on `{bucket_name}`", map[string]string{
		"{nl}":          "list hotels",
		"{bucket_name}": "travel-sample",
	})
	if got != "Q: list hotels on `travel-sample`" {
		t.Fatalf("got %q", got)
	}
}

func TestFillTemplate_UnknownTokensUntouched(t *testing.T) {
	got := directFillTemplate("{present} and {absent}", map[string]string{"{present}": "x"})
	if got != "x and {absent}" {
		t.Fatalf("got %q", got)
	}
}

func TestFillTemplate_EmptyVars(t *testing.T) {
	tmpl := "SELECT {field} FROM t"
	if got := directFillTemplate(tmpl, nil); got != tmpl {
		t.Fatalf("got %q", got)
	}
}

// ─── naturalOutput ────────────────────────────────────────────────────────────

func TestNewNaturalOutput(t *testing.T) {
	cases := map[string]naturalOutput{
		"sql": SQL, "SQL": SQL, "jsudf": JSUDF, "FTSSQL": FTSSQL, "bogus": UNDEFINED_NATURAL_OUTPUT,
	}
	for in, want := range cases {
		if got := NewNaturalOutput(in); got != want {
			t.Fatalf("NewNaturalOutput(%q): got %v, want %v", in, got, want)
		}
	}
}

// ─── SQL prompt builders ──────────────────────────────────────────────────────

func testKeyspaceInfo() map[string]interface{} {
	return map[string]interface{}{
		"hotel": map[string]interface{}{
			"schema":   map[string]string{"name": "\"string\""},
			"fullpath": ":`travel-sample`.`inventory`.`hotel`",
		},
	}
}

func testPaths() []*algebra.Path {
	return []*algebra.Path{algebra.NewPathLong("default", "travel-sample", "inventory", "hotel")}
}

// newSQLPrompt must route the slm provider to the slm variant templates.
func TestNewSQLPrompt_SLMVariant(t *testing.T) {
	p, err := newDirectSQLPrompt(testKeyspaceInfo(), testPaths(), "list hotels", "", "", false,
		ai_gateway.ProviderSLM, ai_gateway.SLMDefaultModel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.InitMessages) != 1 || p.InitMessages[0].Content != slmSystemTmpl {
		t.Fatal("slm prompt must carry the slm system template as its only init message")
	}
	if len(p.Messages) != 1 {
		t.Fatalf("messages: got %d", len(p.Messages))
	}
	user := p.Messages[0].Content
	// No summary was supplied: the {summary} slot must vanish entirely, leaving
	// the message starting at the schema section with no empty header.
	if !strings.HasPrefix(user, "Database Schema:") {
		t.Fatalf("user message must start with the schema section, got: %.60q", user)
	}
	if strings.Contains(user, "{summary}") || strings.Contains(user, "Summary of the conversation") {
		t.Fatal("no summary section expected")
	}
	// The schema must be keyed by the fully-qualified path, with each component
	// backtick-quoted and the field map under "properties", matching the shape the
	// slm was trained on.
	if !strings.Contains(user, "\"`travel-sample`.`inventory`.`hotel`\":{\"properties\":") {
		t.Fatal("schema must be keyed by the backtick-quoted fully-qualified path with a properties object")
	}
	if !strings.Contains(user, "list hotels") {
		t.Fatal("user message must carry the question")
	}
	if p.Size != _INIT_SIZE+len(slmSystemTmpl)+len(user) {
		t.Fatalf("size accounting: got %d", p.Size)
	}
}

func TestNewSQLPrompt_SLMVariant_SummaryAndHintAndFTS(t *testing.T) {
	p, err := newDirectSQLPrompt(testKeyspaceInfo(), testPaths(), "list hotels", "prior context", "use city", true,
		ai_gateway.ProviderSLM, ai_gateway.SLMDefaultModel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	user := p.Messages[0].Content
	if !strings.HasPrefix(user, "Summary of the conversation so far:\nprior context\n\n") {
		t.Fatalf("summary section missing or malformed: %.80q", user)
	}
	if !strings.Contains(user, "Hint: \"use city\"") {
		t.Fatal("hint missing")
	}
	if !strings.Contains(user, "USE INDEX (USING FTS)") {
		t.Fatal("FTS instruction missing")
	}
}

func TestNewSQLPrompt_NonSLM(t *testing.T) {
	p, err := newDirectSQLPrompt(testKeyspaceInfo(), testPaths(), "list hotels", "prior context", "", false,
		ai_gateway.ProviderOpenAI, "gpt-4o-2024-05-13")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.InitMessages[0].Content == slmSystemTmpl {
		t.Fatal("non-slm prompt must not use the slm system template")
	}
	user := p.Messages[0].Content
	if !strings.Contains(user, "Summary of the conversation so far: prior context") {
		t.Fatal("summary missing")
	}
	if !strings.Contains(user, "`travel-sample`") || !strings.Contains(user, "Prompt: \"list hotels\"") {
		t.Fatal("schema or prompt missing from user message")
	}
	if p.Provider != ai_gateway.ProviderOpenAI || p.CompletionSettings.Model != "gpt-4o-2024-05-13" {
		t.Fatalf("provider/model: got %q/%q", p.Provider, p.CompletionSettings.Model)
	}
}

// ─── JSUDF prompt builder ─────────────────────────────────────────────────────

// The slm finetune was trained on the slm system template, so the JSUDF prompt
// must prepend it (in the same, single system message) and account for it in
// the prompt size.
func TestNewJSUDFPrompt_SLMSystemTemplatePrepended(t *testing.T) {
	slm, err := newDirectJSUDFPrompt(testKeyspaceInfo(), "add two numbers", "", "",
		ai_gateway.ProviderSLM, ai_gateway.SLMDefaultModel)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hosted, err := newDirectJSUDFPrompt(testKeyspaceInfo(), "add two numbers", "", "",
		ai_gateway.ProviderOpenAI, "gpt-4o-2024-05-13")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(slm.InitMessages) != 1 {
		t.Fatalf("slm prompt must have a single system turn, got %d", len(slm.InitMessages))
	}
	sys := slm.InitMessages[0].Content
	if !strings.HasPrefix(sys, slmSystemTmpl) {
		t.Fatal("slm JSUDF system message must start with the slm system template")
	}
	if !strings.Contains(sys, "Javascript user defined functions") {
		t.Fatal("slm JSUDF system message must retain the JSUDF task instructions")
	}
	if strings.HasPrefix(hosted.InitMessages[0].Content, slmSystemTmpl) {
		t.Fatal("hosted JSUDF system message must not carry the slm system template")
	}
	if slm.Size != hosted.Size+len(slmSystemTmpl) {
		t.Fatalf("size accounting: slm %d, hosted %d", slm.Size, hosted.Size)
	}
}

// ─── getTemperatureForModel ───────────────────────────────────────────────────

func TestGetTemperatureForModel(t *testing.T) {
	cases := []struct {
		provider, model string
		want            float64
	}{
		{ai_gateway.ProviderOpenAI, "gpt-5-turbo", 1},
		// Model ids are passed to providers verbatim (case preserved), so the
		// gpt-5 check must be case-insensitive.
		{ai_gateway.ProviderOpenAI, "GPT-5-TURBO", 1},
		{ai_gateway.ProviderOpenAI, "gpt-4o-2024-05-13", 0},
		{ai_gateway.ProviderSLM, "any-model", 0},
		{ai_gateway.ProviderBedrock, "gpt-5-lookalike", 0},
	}
	for _, c := range cases {
		if got := directGetTemperatureForModel(c.provider, c.model); got != c.want {
			t.Fatalf("(%s, %s): got %v, want %v", c.provider, c.model, got, c.want)
		}
	}
}

// ─── response content extraction ──────────────────────────────────────────────

func TestCheckAndReturnErrorResponse(t *testing.T) {
	if err := CheckAndReturnErrorResponse("SELECT 1"); err != nil {
		t.Fatalf("no #ERR marker: got %v", err)
	}
	err := CheckAndReturnErrorResponse("#ERR:\" cannot generate a query for that")
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.TrimSpace(err.Error()) != "cannot generate a query for that" {
		t.Fatalf("got %q", err.Error())
	}
}

func TestGetSQLContent(t *testing.T) {
	cases := []struct{ in, want string }{
		{"```sql\nSELECT 1;\n```", "SELECT 1"},
		{"```\nSELECT 1\n```", "SELECT 1"},
		{"SELECT 1;", "SELECT 1"},
		{"  SELECT 1  ", "SELECT 1"},
		{"", ""},
	}
	for _, c := range cases {
		if got := getSQLContent(c.in); got != c.want {
			t.Fatalf("getSQLContent(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetJsContent(t *testing.T) {
	stmt := "CREATE FUNCTION add(a,b) LANGUAGE JAVASCRIPT AS 'function add(a,b) { return(a+b);}'"
	cases := []struct{ in, want string }{
		// Hosted providers are instructed to fence with ```javascript.
		{"```javascript\n" + stmt + "\n```", stmt},
		{"```js\n" + stmt + "\n```", stmt},
		// Models occasionally mislabel the fence.
		{"```sql\n" + stmt + "\n```", stmt},
		// The slm system template instructs a plain fence.
		{"```\n" + stmt + "\n```", stmt},
		// And no fence at all.
		{stmt, stmt},
		{"  " + stmt + "  ", stmt},
	}
	for _, c := range cases {
		if got := getJsContent(c.in); got != c.want {
			t.Fatalf("getJsContent(%.20q...): got %q", c.in, got)
		}
	}
}

func TestGetStatement_EmptyContent(t *testing.T) {
	_, err := getStatement("", SQL)
	if err == nil || err.Code() != errors.E_NL_FAIL_GENERATED_STMT {
		t.Fatalf("expected E_NL_FAIL_GENERATED_STMT, got %v", err)
	}
}

// ─── INFER schema and sample extraction ───────────────────────────────────────

// inferValue builds a value shaped like a single-keyspace INFER result.
func inferValue(properties map[string]interface{}) value.Value {
	return value.NewValue([]interface{}{
		map[string]interface{}{"properties": properties},
	})
}

func TestCollectSchemaForPromptFromInfer_Types(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"name":  map[string]interface{}{"type": "string"},
		"score": map[string]interface{}{"type": []interface{}{"number", "string"}},
		"~meta": map[string]interface{}{"type": "object"},
	})
	schema, samples := collectSchemaFromInfer(map[string]string{}, infer, false)
	if samples != nil {
		t.Fatalf("samples must be nil when not requested, got %v", samples)
	}
	if schema["name"] != "\"string\"" {
		t.Fatalf("name type: got %q", schema["name"])
	}
	if schema["score"] != "number or string" {
		t.Fatalf("score type: got %q", schema["score"])
	}
	if _, ok := schema["~meta"]; ok {
		t.Fatal("~meta must be skipped")
	}
}

func TestCollectSchemaForPromptFromInfer_Samples(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"type":    map[string]interface{}{"type": "string", "samples": []interface{}{"hotel", "airline"}},
		"ratings": map[string]interface{}{"type": "number"}, // no samples reported
	})
	_, samples := collectSchemaFromInfer(map[string]string{}, infer, true)
	if len(samples) != 1 {
		t.Fatalf("samples: got %v", samples)
	}
	got, ok := samples["type"]
	if !ok || len(got) != 2 || got[0] != "hotel" || got[1] != "airline" {
		t.Fatalf("type samples: got %v", got)
	}
}

func TestCollectSchemaForPromptFromInfer_NoSamplesReported(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"name": map[string]interface{}{"type": "string"},
	})
	_, samples := collectSchemaFromInfer(map[string]string{}, infer, true)
	if samples != nil {
		t.Fatalf("expected nil samples map, got %v", samples)
	}
}

// ─── slm sample injection and the samples privacy invariant ───────────────────

func TestSlmSamplesBlock(t *testing.T) {
	if got := directSLMSamplesBlock(nil); got != "" {
		t.Fatalf("nil samples: got %q", got)
	}
	block := directSLMSamplesBlock(map[string]map[string][]interface{}{
		"hotel": {"type": {"hotel", "airline"}},
	})
	if !strings.Contains(block, "Representative sample values") ||
		!strings.Contains(block, `"hotel"`) || !strings.Contains(block, `"airline"`) {
		t.Fatalf("block malformed: %q", block)
	}
}

// Sample values are provider-gated context, not conversation history: they must
// never appear in the marshaled prompt (which is what gets persisted in the
// chat document on pause).
func TestPromptMarshal_ExcludesSamples(t *testing.T) {
	const sentinel = "SAMPLE-VALUE-MUST-NOT-PERSIST"
	p := &prompt{
		Provider: ai_gateway.ProviderSLM,
		Messages: []message{{Role: "user", Content: "list hotels"}},
		samples:  map[string]map[string][]interface{}{"hotel": {"name": {sentinel}}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), sentinel) {
		t.Fatalf("sample values leaked into the marshaled prompt: %s", b)
	}
	if strings.Contains(string(b), "samples") {
		t.Fatalf("a samples field leaked into the marshaled prompt: %s", b)
	}
}

func TestChatEntryMarshal_ExcludesSamples(t *testing.T) {
	const sentinel = "SAMPLE-VALUE-MUST-NOT-PERSIST"
	samples := map[string]map[string][]interface{}{"hotel": {"name": {sentinel}}}
	ce := &ChatEntry{
		users:     []string{"local:tester"},
		Keyspaces: testPaths(),
		Summary:   "prior context",
		prompt: &prompt{
			Provider: ai_gateway.ProviderSLM,
			Messages: []message{{Role: "user", Content: "list hotels"}},
			samples:  samples,
		},
		samples:           samples,
		inactivityTimeout: 5 * time.Minute,
	}
	b, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), sentinel) {
		t.Fatalf("sample values leaked into the persisted chat document: %s", b)
	}
}

func TestChatEntryMarshal_RoundTrip(t *testing.T) {
	ce := &ChatEntry{
		users:     []string{"local:tester"},
		Keyspaces: testPaths(),
		Summary:   "prior context",
		prompt: &prompt{
			Provider: ai_gateway.ProviderOpenAI,
			Messages: []message{{Role: "user", Content: "list hotels"}},
			Size:     300,
		},
		inactivityTimeout: 5 * time.Minute,
	}
	b, err := json.Marshal(ce)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got ChatEntry
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !slices.Equal(got.users, ce.users) || got.Summary != ce.Summary || got.inactivityTimeout != ce.inactivityTimeout {
		t.Fatalf("fields not restored: users=%q summary=%q timeout=%v",
			got.users, got.Summary, got.inactivityTimeout)
	}
	if got.prompt == nil || got.prompt.Provider != ai_gateway.ProviderOpenAI ||
		len(got.prompt.Messages) != 1 || got.prompt.Messages[0].Content != "list hotels" {
		t.Fatalf("prompt not restored: %+v", got.prompt)
	}
	if len(got.Keyspaces) != 1 ||
		got.Keyspaces[0].ProtectedString() != ce.Keyspaces[0].ProtectedString() {
		t.Fatalf("keyspaces not restored: %v", got.Keyspaces)
	}
}

func TestChatEntryUnmarshal_InvalidTimeout(t *testing.T) {
	var ce ChatEntry
	if err := json.Unmarshal([]byte(`{"inactivity_timeout":"not-a-duration"}`), &ce); err == nil {
		t.Fatal("expected an error for an invalid timeout")
	}
}

// ─── gateway bridge ───────────────────────────────────────────────────────────

func TestToGatewayRequest_MappingAndCopy(t *testing.T) {
	p := &prompt{
		Provider:     ai_gateway.ProviderSLM,
		InitMessages: []message{{Role: "system", Content: "sys"}},
		Messages:     []message{{Role: "user", Content: "original"}},
		CompletionSettings: completionSettings{
			Model:       "m",
			Temperature: 0.5,
			Seed:        1,
		},
	}
	req := p.toDirectGatewayRequest(&NaturalConfig{OutputTokenLimit: 100})
	if req.Model != "m" ||
		req.Temperature != 0.5 || req.Seed != 1 || req.MaxTokens != 100 {
		t.Fatalf("mapping: got %+v", req)
	}
	if len(req.InitMessages) != 1 || len(req.Messages) != 1 {
		t.Fatalf("messages: got %+v", req)
	}

	// doChatCompletion appends the slm samples block to req.Messages assuming
	// they are a fresh copy; mutating the request must not touch the prompt.
	req.Messages[0].Content = "mutated"
	if p.Messages[0].Content != "original" {
		t.Fatal("toGatewayRequest must return copied messages, not aliases of the prompt's")
	}
}
