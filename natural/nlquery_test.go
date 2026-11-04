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
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/natural/ai_gateway"
	"github.com/couchbase/query/parser/n1ql"
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

// testSLMModel is the couchmind served-model id the slm templates were built for.
const testSLMModel = "jastorj/couchmind-v5.7.8.1_arctic_stage_3-cw-24K-16bit"

// newSQLPrompt must route the slm provider to the slm variant templates.
func TestNewSQLPrompt_SLMVariant(t *testing.T) {
	p, err := newDirectSQLPrompt(testKeyspaceInfo(), testPaths(), "list hotels", "", "", false,
		ai_gateway.ProviderSLM, testSLMModel)
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
		ai_gateway.ProviderSLM, testSLMModel)
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
		ai_gateway.ProviderSLM, testSLMModel)
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
	tests := []struct {
		name    string
		content string
		wantErr bool
		wantMsg string
	}{
		{"no error marker", "SELECT * FROM foo", false, ""},
		{"error with explanation", "#ERR: bad request", true, "bad request"},
		{"error marker alone at end", "#ERR", true, "unexpected empty error response from LLM"},
		{"error marker with no trailing content", "#ERR: ", true, "unexpected empty error response from LLM"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckAndReturnErrorResponse(tc.content)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.wantErr && err.Error() != tc.wantMsg {
				t.Errorf("expected message %q, got %q", tc.wantMsg, err.Error())
			}
		})
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
	// Locks in that the call site supplies args matching E_NL_FAIL_GENERATED_STMT's
	// "...failed after %d retries: %v" format (a mismatched-arity Sprintf call here
	// would silently produce a garbled "%!d(string=...)" message instead of an error).
	if want := "Statement generation failed after 0 retries: empty response"; err.Error() != want {
		t.Fatalf("got %q, want %q", err.Error(), want)
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
	schema, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, false)
	if samples != nil {
		t.Fatalf("samples must be nil when not requested, got %v", samples)
	}
	if schema["name"].Type != "string" {
		t.Fatalf("name type: got %q", schema["name"].Type)
	}
	if schema["score"].Type != "number or string" {
		t.Fatalf("score type: got %q", schema["score"].Type)
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
	_, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, true)
	if len(samples) != 1 {
		t.Fatalf("samples: got %v", samples)
	}
	got, ok := samples["type"]
	if !ok {
		t.Fatalf("type samples: missing")
	}
	strSamples := got.Samples["string"]
	if len(strSamples) != 2 || strSamples[0] != "hotel" || strSamples[1] != "airline" {
		t.Fatalf("type samples: got %v", got.Samples)
	}
	// schemaField has no Samples field at all, so the type tree is samples-free
	// by construction -- nothing further to assert here.
}

func TestCollectSchemaForPromptFromInfer_NoSamplesReported(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"name": map[string]interface{}{"type": "string"},
	})
	_, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, true)
	if samples != nil {
		t.Fatalf("expected nil samples map, got %v", samples)
	}
}

// A field that can be more than one shape (e.g. "null or object") must still
// recurse into its object shape's properties -- losing nested structure just
// because a field is nullable would silently hide most optional nested
// objects, which are common in flexible-schema documents.
func TestCollectFields_UnionObjectRecursesIntoProperties(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"address": map[string]interface{}{
			"type": []interface{}{"null", "object"},
			"properties": map[string]interface{}{
				"city": map[string]interface{}{"type": "string"},
			},
		},
	})
	schema, _ := collectSchemaFromInfer(map[string]*schemaField{}, infer, false)
	addr, ok := schema["address"]
	if !ok {
		t.Fatal("address field missing")
	}
	if addr.Type != "null or object" {
		t.Fatalf("address type: got %q", addr.Type)
	}
	city, ok := addr.Properties["city"]
	if !ok || city.Type != "string" {
		t.Fatalf("union-typed object field must still recurse into properties, got %+v", addr.Properties)
	}
}

// INFER never attaches samples inside "items" itself -- only whole example
// arrays on the array field -- so an array field's own sample bucket must
// carry those whole instances as-is, keyed by the "array" shape, rather than
// leaving them unreachable or split apart into item fields (which would lose
// which values co-occurred in the same sampled record).
func TestCollectFields_ArraySamplesAttachToFieldItself(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"reviews": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"author": map[string]interface{}{"type": "string"},
				},
			},
			"samples": []interface{}{
				[]interface{}{
					map[string]interface{}{"author": "Alice", "rating": 5},
					map[string]interface{}{"author": "Bob", "rating": 3},
				},
			},
		},
	})
	_, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, true)
	reviews, ok := samples["reviews"]
	if !ok {
		t.Fatal("reviews samples missing")
	}
	if len(reviews.Items) != 0 {
		t.Fatalf("array field samples must not be decomposed into item properties, got Items=%+v", reviews.Items)
	}
	got := reviews.Samples["array"]
	if len(got) != 1 {
		t.Fatalf("reviews array samples: got %+v", got)
	}
	instance, ok := got[0].([]interface{})
	if !ok || len(instance) != 2 {
		t.Fatalf("expected the whole sampled instance intact, got %+v", got[0])
	}
	// The instance must survive as one JSON-shaped value, so author/rating pairs
	// from the same record are still readable together.
	first, ok := instance[0].(map[string]interface{})
	if !ok || first["author"] != "Alice" || first["rating"] != 5 {
		t.Fatalf("expected co-occurring fields preserved on one element, got %+v", instance[0])
	}
}

// A heterogeneous array (items reported as more than one shape) must still
// carry its whole sampled instances intact -- there is no per-shape routing
// to do since nothing is being decomposed.
func TestCollectFields_HeterogeneousArraySamplesAttachToFieldItself(t *testing.T) {
	infer := inferValue(map[string]interface{}{
		"notes": map[string]interface{}{
			"type": "array",
			"items": []interface{}{
				map[string]interface{}{"type": "string"},
				map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"author": map[string]interface{}{"type": "string"},
					},
				},
			},
			"samples": []interface{}{
				[]interface{}{
					"hello",
					map[string]interface{}{"author": "Alice"},
				},
			},
		},
	})
	_, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, true)
	notes, ok := samples["notes"]
	if !ok {
		t.Fatal("notes samples missing")
	}
	got := notes.Samples["array"]
	if len(got) != 1 {
		t.Fatalf("notes array samples: got %+v", got)
	}
	instance, ok := got[0].([]interface{})
	if !ok || len(instance) != 2 || instance[0] != "hello" {
		t.Fatalf("expected the whole heterogeneous instance intact, got %+v", got[0])
	}
}

// A single array sample is a whole field value copied verbatim from a real
// document, unlike a scalar sample -- so unlike scalar buckets (already
// bounded by INFER's own per-field sample count), it needs its own size cap
// to keep a large array field (or a long string nested inside one) from
// dumping unabridged into the prompt.
func TestCollectFields_ArraySamplesAreCapped(t *testing.T) {
	longString := strings.Repeat("x", _MAX_SAMPLE_STRING_LEN+10)
	manyElems := []interface{}{}
	for i := 0; i < _MAX_SAMPLE_ARRAY_LEN+3; i++ {
		manyElems = append(manyElems, map[string]interface{}{"comment": longString})
	}
	infer := inferValue(map[string]interface{}{
		"reviews": map[string]interface{}{
			"type":    "array",
			"items":   map[string]interface{}{"type": "object"},
			"samples": []interface{}{manyElems},
		},
	})
	_, samples := collectSchemaFromInfer(map[string]*schemaField{}, infer, true)
	instance, ok := samples["reviews"].Samples["array"][0].([]interface{})
	if !ok {
		t.Fatalf("reviews array sample missing: got %+v", samples["reviews"])
	}
	if len(instance) != _MAX_SAMPLE_ARRAY_LEN {
		t.Fatalf("expected array truncated to %d elements, got %d", _MAX_SAMPLE_ARRAY_LEN, len(instance))
	}
	elem, ok := instance[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected element to remain an object, got %+v", instance[0])
	}
	if s := elem["comment"].(string); len(s) != _MAX_SAMPLE_STRING_LEN {
		t.Fatalf("expected nested string truncated to %d bytes, got %d", _MAX_SAMPLE_STRING_LEN, len(s))
	}
}

// ─── slm sample injection and the samples privacy invariant ───────────────────

func TestSlmSamplesBlock(t *testing.T) {
	if got := directSLMSamplesBlock(nil); got != "" {
		t.Fatalf("nil samples: got %q", got)
	}
	block := directSLMSamplesBlock(map[string]map[string]*sampleField{
		"hotel": {"type": {Samples: map[string][]interface{}{"string": {"hotel", "airline"}}}},
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
		samples:  map[string]map[string]*sampleField{"hotel": {"name": {Samples: map[string][]interface{}{"string": {sentinel}}}}},
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
	samples := map[string]map[string]*sampleField{"hotel": {"name": {Samples: map[string][]interface{}{"string": {sentinel}}}}}
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

// TestChatEntryMarshal_KnowledgeRoundTrip guards that the BEGIN CHAT "knowledge" choice survives a
// PAUSE/RESUME: the prompt is rebuilt from scratch after a resume, so a lost flag would silently drop
// (or add) knowledge for the rest of the conversation.
func TestChatEntryMarshal_KnowledgeRoundTrip(t *testing.T) {
	for _, useKnowledge := range []bool{true, false} {
		ce := &ChatEntry{
			users:        []string{"local:tester"},
			Keyspaces:    testPaths(),
			useKnowledge: useKnowledge,
		}
		b, err := json.Marshal(ce)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !useKnowledge && strings.Contains(string(b), "knowledge") {
			t.Fatalf("knowledge field persisted for a chat without knowledge: %s", b)
		}

		var got ChatEntry
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.useKnowledge != useKnowledge {
			t.Fatalf("useKnowledge not restored: got %v, want %v", got.useKnowledge, useKnowledge)
		}
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

func TestVectorSearchInstructions(t *testing.T) {
	sql := vectorSearchInstructions(false)
	if !strings.Contains(sql, "ORDER BY APPROX_VECTOR_DISTANCE") {
		t.Error("SQL variant missing APPROX_VECTOR_DISTANCE guidance")
	}
	if strings.Contains(sql, "parameter of the generated function") {
		t.Error("SQL variant should not contain JSUDF-specific guidance")
	}

	js := vectorSearchInstructions(true)
	if !strings.Contains(js, "parameter of the generated function") {
		t.Error("JSUDF variant missing function-parameter guidance")
	}
	if !strings.Contains(js, "ORDER BY APPROX_VECTOR_DISTANCE") {
		t.Error("JSUDF variant missing APPROX_VECTOR_DISTANCE guidance")
	}
}

func TestNewDirectSQLPromptIncludesVectorSearchInstructions(t *testing.T) {
	p, err := newDirectSQLPrompt(testKeyspaceInfo(), testPaths(), "find similar docs", "", "", false,
		ai_gateway.ProviderOpenAI, "gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(p.Messages))
	}
	if !strings.Contains(p.Messages[0].Content, vectorSearchInstructions(false)) {
		t.Error("SQL prompt does not contain vector-search instructions")
	}
}

func TestNewDirectJSUDFPromptIncludesVectorSearchInstructions(t *testing.T) {
	p, err := newDirectJSUDFPrompt(testKeyspaceInfo(), "find similar docs", "", "",
		ai_gateway.ProviderOpenAI, "gpt-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Messages) != 1 {
		t.Fatalf("expected 1 user message, got %d", len(p.Messages))
	}
	if !strings.Contains(p.Messages[0].Content, vectorSearchInstructions(true)) {
		t.Error("JSUDF prompt does not contain vector-search instructions")
	}
}

func TestVectorSearchInstructionsAsksForClarificationOnAmbiguousField(t *testing.T) {
	for _, forJSUDF := range []bool{false, true} {
		instr := vectorSearchInstructions(forJSUDF)
		if !strings.Contains(instr, "#ERR") || !strings.Contains(instr, "more than one") {
			t.Errorf("forJSUDF=%v: expected guidance to ask for clarification when multiple vector"+
				" fields/indexes are candidates, got: %s", forJSUDF, instr)
		}
	}
}

type fakeVectorIndex struct {
	datastore.Index
	name         string
	rangeKey     datastore.IndexKeys
	isVector     bool
	distanceType datastore.IndexDistanceType
	dimension    int
	probes       int
}

func (f *fakeVectorIndex) Name() string                                    { return f.name }
func (f *fakeVectorIndex) RangeKey2() datastore.IndexKeys                  { return f.rangeKey }
func (f *fakeVectorIndex) IsVector() bool                                  { return f.isVector }
func (f *fakeVectorIndex) VectorDistanceType() datastore.IndexDistanceType { return f.distanceType }
func (f *fakeVectorIndex) VectorDimension() int                            { return f.dimension }
func (f *fakeVectorIndex) VectorProbes() int                               { return f.probes }

type notAVectorIndex struct {
	datastore.Index
	name string
}

func (f *notAVectorIndex) Name() string { return f.name }

func TestCollectVectorIndexes(t *testing.T) {
	vecKey := &datastore.IndexKey{Expr: expression.NewIdentifier("embedding"), Attributes: datastore.IK_DENSE_VECTOR}
	plainKey := &datastore.IndexKey{Expr: expression.NewIdentifier("name"), Attributes: datastore.IK_NONE}

	vecIdx := &fakeVectorIndex{
		name:         "idx_vector",
		rangeKey:     datastore.IndexKeys{plainKey, vecKey},
		isVector:     true,
		distanceType: datastore.IX_DIST_COSINE,
		dimension:    128,
		probes:       1,
	}
	nonVecIdx := &fakeVectorIndex{name: "idx_plain", isVector: false}
	otherIdx := &notAVectorIndex{name: "idx_other"}
	nilExprIdx := &fakeVectorIndex{
		name:         "idx_nil_expr",
		rangeKey:     datastore.IndexKeys{{Expr: nil, Attributes: datastore.IK_DENSE_VECTOR}},
		isVector:     true,
		distanceType: datastore.IX_DIST_COSINE,
	}

	rv := collectVectorIndexes([]datastore.Index{vecIdx, nonVecIdx, otherIdx, nilExprIdx})

	if len(rv) != 1 {
		t.Fatalf("expected 1 vector index entry, got %d: %+v", len(rv), rv)
	}
	info := rv[0]
	if info["field"] != "`embedding`" {
		t.Errorf("expected field '`embedding`', got %v", info["field"])
	}
	if info["similarity"] != string(datastore.IX_DIST_COSINE) {
		t.Errorf("expected similarity 'cosine', got %v", info["similarity"])
	}
	if info["indexName"] != "idx_vector" {
		t.Errorf("expected indexName 'idx_vector', got %v", info["indexName"])
	}
	if info["dimension"] != 128 {
		t.Errorf("expected dimension 128, got %v", info["dimension"])
	}
	if info["type"] != datastore.IK_DENSE_VECTOR_NAME {
		t.Errorf("expected type 'dense', got %v", info["type"])
	}
}

func TestCollectVectorIndexesSparse(t *testing.T) {
	sparseKey := &datastore.IndexKey{Expr: expression.NewIdentifier("sparseVec"), Attributes: datastore.IK_SPARSE_VECTOR}

	vecIdx := &fakeVectorIndex{
		name:         "idx_sparse_vec",
		rangeKey:     datastore.IndexKeys{sparseKey},
		isVector:     true,
		distanceType: datastore.IX_DIST_DOT,
		dimension:    128, // should be ignored/omitted for sparse
		probes:       1,
	}

	rv := collectVectorIndexes([]datastore.Index{vecIdx})
	if len(rv) != 1 {
		t.Fatalf("expected 1 vector index entry, got %d: %+v", len(rv), rv)
	}
	info := rv[0]
	if info["type"] != datastore.IK_SPARSE_VECTOR_NAME {
		t.Errorf("expected type 'sparse', got %v", info["type"])
	}
	if _, ok := info["dimension"]; ok {
		t.Errorf("expected no 'dimension' entry for a sparse vector index, got %v", info["dimension"])
	}
}

func TestVectorFieldInfoNilExpressionDoesNotPanic(t *testing.T) {
	vi := &fakeVectorIndex{
		name:         "idx_nil_expr",
		rangeKey:     datastore.IndexKeys{{Expr: nil, Attributes: datastore.IK_DENSE_VECTOR}},
		isVector:     true,
		distanceType: datastore.IX_DIST_COSINE,
	}
	if info := vectorFieldInfo(vi); info != nil {
		t.Errorf("expected nil info when the vector key has a nil expression, got %+v", info)
	}
}

type fakeStatement struct {
	algebra.Statement
	paramsCount int
	stmtType    string
}

func (f *fakeStatement) ParamsCount() int { return f.paramsCount }
func (f *fakeStatement) Type() string     { return f.stmtType }

func TestCanServerExecuteGeneratedStatement(t *testing.T) {
	tests := []struct {
		name        string
		stmtType    string
		paramsCount int
		want        bool
	}{
		{"SELECT no params", "SELECT", 0, true},
		{"ADVISE no params", "ADVISE", 0, true},
		{"EXPLAIN no params", "EXPLAIN", 0, true},
		{"SELECT with named/positional params", "SELECT", 1, false},
		{"ADVISE with params", "ADVISE", 2, true},
		{"EXPLAIN with params", "EXPLAIN", 3, true},
		{"INSERT no params", "INSERT", 0, false},
		{"UPDATE no params", "UPDATE", 0, false},
		{"DELETE with params", "DELETE", 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stmt := &fakeStatement{stmtType: tc.stmtType, paramsCount: tc.paramsCount}
			if got := CanServerExecuteGeneratedStatement(stmt); got != tc.want {
				t.Errorf("CanServerExecuteGeneratedStatement(%s, params=%d) = %v, want %v",
					tc.stmtType, tc.paramsCount, got, tc.want)
			}
		})
	}
}

func TestVectorIndexesForPathReturnsNilWithoutPanicking(t *testing.T) {
	p := algebra.NewPathShort("default", "no-such-keyspace")
	if rv := vectorIndexesForPath(p); rv != nil {
		t.Errorf("expected nil for a non-existent keyspace, got %+v", rv)
	}
}

// ─── ambiguous-term / anti-hallucination instruction (MB-72780) ───────────────
func TestAppendSQLUserMessage_IncludesAmbiguousTermInstruction(t *testing.T) {
	p := &prompt{}
	if err := appendSQLUserMessage(p, testKeyspaceInfo(), "list hotels", "", "", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(p.Messages[0].Content, _AMBIGUOUS_TERM_INSTRUCTION) {
		t.Fatal("SQL user message missing the ambiguous-term instruction")
	}
}

// ─── validateGeneratedStatement (rewrite + semantic-check retry coverage) ─────

// mustParse parses a raw N1QL statement for use as validateGeneratedStatement
// input. Parsing alone (no datastore) is enough since neither rewrite nor the
// semantic checker touches the datastore.
func mustParse(t *testing.T, stmt string) algebra.Statement {
	t.Helper()
	s, err := n1ql.ParseStatement2(stmt, "default", "")
	if err != nil {
		t.Fatalf("parse %q: %v", stmt, err)
	}
	return s
}

func TestValidateGeneratedStatement_Valid(t *testing.T) {
	if err := validateGeneratedStatement(mustParse(t, "SELECT * FROM default"), nil); err != nil {
		t.Fatalf("unexpected error for a valid statement: %v", err)
	}
}

// A generated statement referencing an undeclared window name parses fine but
// fails rewrite.NewRewrite(REWRITE_PHASE1)'s window-term validation.
func TestValidateGeneratedStatement_RewriteError(t *testing.T) {
	stmt := mustParse(t, "SELECT RANK() OVER w FROM default")
	if err := validateGeneratedStatement(stmt, nil); err == nil {
		t.Fatal("expected a rewrite error for an undeclared window reference")
	}
}

// A sequence operation inside a WHERE clause parses fine but is rejected by
// semantics.GetSemChecker.
func TestValidateGeneratedStatement_SemanticError(t *testing.T) {
	stmt := mustParse(t, "SELECT * FROM default WHERE NEXTVAL FOR `b`.`s`.`seq1` > 0")
	if err := validateGeneratedStatement(stmt, nil); err == nil {
		t.Fatal("expected a semantic error for a sequence operation in a WHERE clause")
	}
}

// A nil context (as used by callers that have none available) must not panic
// and must behave as inTx=false.
func TestValidateGeneratedStatement_NilContext(t *testing.T) {
	if err := validateGeneratedStatement(mustParse(t, "SELECT * FROM default"), nil); err != nil {
		t.Fatalf("unexpected error with nil context: %v", err)
	}
}

// ─── retry-feedback prompts fold in _SQLPP_TASK_INSTRUCTIONS once ────────────

func TestDirectBuildRetryPrompt_SLM_NeverIncludesInstructions(t *testing.T) {
	// The slm system prompt already carries _SQLPP_TASK_INSTRUCTIONS, so the slm
	// feedback branch must never duplicate it into the feedback turn regardless
	// of includeInstructions.
	for _, include := range []bool{true, false} {
		p := &prompt{Provider: ai_gateway.ProviderSLM}
		directBuildRetryPrompt(p, "SELECT bogus", fmt.Errorf("parse error"), include)
		feedback := p.Messages[len(p.Messages)-1].Content
		if strings.Contains(feedback, _SQLPP_TASK_INSTRUCTIONS) {
			t.Fatalf("include=%v: slm feedback must not carry _SQLPP_TASK_INSTRUCTIONS, got: %.80q", include, feedback)
		}
	}
}

func TestDirectBuildRetryPrompt_NonSLM_IncludesInstructionsOnlyWhenAsked(t *testing.T) {
	p := &prompt{Provider: ai_gateway.ProviderOpenAI}
	directBuildRetryPrompt(p, "SELECT bogus", fmt.Errorf("parse error"), true)
	if got := p.Messages[len(p.Messages)-1].Content; !strings.Contains(got, _SQLPP_TASK_INSTRUCTIONS) {
		t.Fatalf("includeInstructions=true: expected _SQLPP_TASK_INSTRUCTIONS in feedback, got: %.80q", got)
	}

	p2 := &prompt{Provider: ai_gateway.ProviderOpenAI}
	directBuildRetryPrompt(p2, "SELECT bogus", fmt.Errorf("parse error"), false)
	if got := p2.Messages[len(p2.Messages)-1].Content; strings.Contains(got, _SQLPP_TASK_INSTRUCTIONS) {
		t.Fatalf("includeInstructions=false: expected no _SQLPP_TASK_INSTRUCTIONS in feedback, got: %.80q", got)
	}
}

// directBuildRetryPrompt must append exactly one assistant turn (the prior,
// failing response) followed by one user turn (the correction feedback), and
// keep pmt.Size in sync with the appended feedback text.
func TestDirectBuildRetryPrompt_AppendsTurnsAndUpdatesSize(t *testing.T) {
	p := &prompt{Provider: ai_gateway.ProviderOpenAI, Size: 10}
	directBuildRetryPrompt(p, "SELECT bogus", fmt.Errorf("parse error"), false)
	if len(p.Messages) != 2 {
		t.Fatalf("expected 2 messages appended, got %d", len(p.Messages))
	}
	if p.Messages[0].Role != "assistant" || p.Messages[0].Content != "SELECT bogus" {
		t.Fatalf("assistant turn not as expected: %+v", p.Messages[0])
	}
	if p.Messages[1].Role != "user" || !strings.Contains(p.Messages[1].Content, "parse error") {
		t.Fatalf("user feedback turn not as expected: %+v", p.Messages[1])
	}
	if want := 10 + len(p.Messages[1].Content); p.Size != want {
		t.Fatalf("size accounting: got %d, want %d", p.Size, want)
	}
}

func TestCapellaBuildRetryPrompt_IncludesInstructionsOnlyWhenAsked(t *testing.T) {
	p := &prompt{}
	capellaBuildRetryPrompt(p, "SELECT bogus", fmt.Errorf("parse error"), true)
	if got := p.Messages[len(p.Messages)-1].Content; !strings.Contains(got, _SQLPP_TASK_INSTRUCTIONS) {
		t.Fatalf("includeInstructions=true: expected _SQLPP_TASK_INSTRUCTIONS in feedback, got: %.80q", got)
	}

	p2 := &prompt{}
	capellaBuildRetryPrompt(p2, "SELECT bogus", fmt.Errorf("parse error"), false)
	if got := p2.Messages[len(p2.Messages)-1].Content; strings.Contains(got, _SQLPP_TASK_INSTRUCTIONS) {
		t.Fatalf("includeInstructions=false: expected no _SQLPP_TASK_INSTRUCTIONS in feedback, got: %.80q", got)
	}
}

// ─── E_NL_FAIL_GENERATED_STMT retry-count reporting ───────────────────────────

func TestFailGeneratedStmtError_IncludesRetryCount(t *testing.T) {
	e := errors.NewNaturalLanguageRequestError(errors.E_NL_FAIL_GENERATED_STMT,
		maxCorrectionRetries, "SELECT bogus", fmt.Errorf("parse error"))
	want := fmt.Sprintf("Statement generation failed after %d retries: SELECT bogus", maxCorrectionRetries)
	if e.Error() != want {
		t.Fatalf("got %q, want %q", e.Error(), want)
	}
}

// ─── persistPausedChat: MB-73909 ambiguous-timeout / duplicate-key disambiguation ──

// fakeQueryMetadataKeyspace embeds a nil datastore.Keyspace and overrides only Insert
// and Fetch, since persistPausedChat (directly, or via confirmChatDocPersisted's
// read-back fallback) never calls any other Keyspace method.
type fakeQueryMetadataKeyspace struct {
	datastore.Keyspace
	attempt        int
	behavior       func(attempt int) (int, value.Pairs, errors.Errors)
	fetchAttempt   int
	fetchBehavior  func(attempt int) (value.AnnotatedValue, errors.Errors) // nil value means "not found"
	updateAttempt  int
	updateBehavior func(attempt int) (int, value.Pairs, errors.Errors)
}

func (f *fakeQueryMetadataKeyspace) Insert(inserts value.Pairs, context datastore.QueryContext,
	preserveMutations bool) (int, value.Pairs, errors.Errors) {
	f.attempt++
	return f.behavior(f.attempt)
}

// Delete shares Insert's attempt/behavior fields -- a given fake instance is only
// ever exercised through one of the two methods per test (persistPausedChat uses
// Insert; completeChatClaim uses Delete), so reusing the same counters keeps the
// fake simple rather than duplicating them.
func (f *fakeQueryMetadataKeyspace) Delete(deletes value.Pairs, context datastore.QueryContext,
	preserveMutations bool) (int, value.Pairs, errors.Errors) {
	f.attempt++
	return f.behavior(f.attempt)
}

func (f *fakeQueryMetadataKeyspace) Fetch(keys []string, keysMap map[string]value.AnnotatedValue,
	context datastore.QueryContext, subPath []string, projection []string, useSubDoc bool) errors.Errors {
	f.fetchAttempt++
	if f.fetchBehavior == nil {
		return errors.Errors{errors.NewOtherDatastoreError(nil, "test: unexpected Fetch call, no fetchBehavior set")}
	}
	av, errs := f.fetchBehavior(f.fetchAttempt)
	if av != nil {
		keysMap[keys[0]] = av
	}
	return errs
}

func (f *fakeQueryMetadataKeyspace) Update(updates value.Pairs, context datastore.QueryContext,
	preserveMutations bool) (int, value.Pairs, errors.Errors) {
	f.updateAttempt++
	if f.updateBehavior == nil {
		return 0, nil, errors.Errors{errors.NewOtherDatastoreError(nil, "test: unexpected Update call, no updateBehavior set")}
	}
	return f.updateBehavior(f.updateAttempt)
}

func timeoutError() errors.Errors {
	return errors.Errors{errors.NewOtherDatastoreError(nil, "read tcp 1.2.3.4:1->5.6.7.8:11210: i/o timeout")}
}

func syncWriteAmbiguousError() errors.Errors {
	return errors.Errors{errors.NewOtherDatastoreError(nil, "SYNC_WRITE_AMBIGUOUS")}
}

func syncWriteInProgressError() errors.Errors {
	return errors.Errors{errors.NewOtherDatastoreError(nil, "SYNC_WRITE_IN_PROGRESS")}
}

func notMyVbucketError() errors.Errors {
	return errors.Errors{errors.NewOtherDatastoreError(nil, "NOT_MY_VBUCKET")}
}

// casMismatchError mirrors what a real CAS-mismatched Update returns: an E_CB_DML
// error whose cause chain carries E_CAS_MISMATCH (see errors.NewCbDMLError).
func casMismatchError() errors.Errors {
	return errors.Errors{errors.NewCbDMLError(nil, "test cas mismatch", true, errors.FALSE, "aichat::chat1", "QUERY_METADATA")}
}

// withFastRetry temporarily shrinks the package's retry budget (maxRetry/interval)
// for exhaustion-path test cases, which otherwise sleep out the full real backoff
// (~6.3s each) for no added coverage. Restored via t.Cleanup once the whole calling
// Test function (including all its t.Run subtests) finishes.
func withFastRetry(t *testing.T) {
	t.Helper()
	origMaxRetry, origInterval, origConfirmMaxRetry := maxRetry, interval, confirmMaxRetry
	maxRetry, interval, confirmMaxRetry = 3, time.Millisecond, 3
	t.Cleanup(func() { maxRetry, interval, confirmMaxRetry = origMaxRetry, origInterval, origConfirmMaxRetry })
}

func TestPersistPausedChat(t *testing.T) {
	withFastRetry(t)
	successResult := func(attempt int) (int, value.Pairs, errors.Errors) { return 1, nil, nil }

	tests := []struct {
		name          string
		behavior      func(attempt int) (int, value.Pairs, errors.Errors)
		fetchBehavior func(attempt int) (value.AnnotatedValue, errors.Errors)
		wantErrCode   errors.ErrorCode // 0 means expect nil error
		wantPaused    bool
		wantAttempts  int
	}{
		{
			name:         "immediate success",
			behavior:     successResult,
			wantPaused:   true,
			wantAttempts: 1,
		},
		{
			name: "ambiguous timeout then success",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				if attempt == 1 {
					return 0, nil, timeoutError()
				}
				return successResult(attempt)
			},
			wantPaused:   true,
			wantAttempts: 2,
		},
		{
			name: "ambiguous timeout then duplicate key -- the write already landed",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				if attempt == 1 {
					return 0, nil, timeoutError()
				}
				return 0, nil, errors.Errors{errors.NewDuplicateKeyError("aichat::chat1", "", nil)}
			},
			wantPaused:   true,
			wantAttempts: 2,
		},
		{
			name: "non-retryable failure fails immediately, chat stays live",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				return 0, nil, errors.Errors{errors.NewOtherDatastoreError(nil, "some fatal error")}
			},
			wantErrCode:  errors.E_NL_CHAT_PAUSE_FAILED,
			wantPaused:   false,
			wantAttempts: 1,
		},
		{
			name:     "exhausted retries without confirmation fails, chat stays live",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, timeoutError() },
			// The write genuinely never landed -- read-back confirms the key absent.
			fetchBehavior: func(attempt int) (value.AnnotatedValue, errors.Errors) { return nil, nil },
			wantErrCode:   errors.E_NL_CHAT_PAUSE_FAILED,
			wantPaused:    false,
			wantAttempts:  maxRetry,
		},
		{
			// MB-73909: the KV-reported (not just client-side-timeout) ambiguity.
			// Every insert attempt comes back SYNC_WRITE_AMBIGUOUS, exhausting the
			// retry budget without ever seeing a duplicate-key confirmation -- exactly
			// the case confirmChatDocPersisted's read-back fallback exists for. The
			// direct read finds the document, confirming the write actually landed, so
			// PAUSE must still report success rather than the old "exhausted retries ==
			// failure" inference.
			name: "SYNC_WRITE_AMBIGUOUS exhausts insert retries, read-back confirms persisted",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				return 0, nil, syncWriteAmbiguousError()
			},
			fetchBehavior: func(attempt int) (value.AnnotatedValue, errors.Errors) {
				return value.NewAnnotatedValue(map[string]interface{}{"chat": "b64"}), nil
			},
			wantPaused:   true,
			wantAttempts: maxRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chatId := "chat1-" + tt.name
			ce := &ChatEntry{Id: chatId, users: []string{"local:tester"}}
			AddConversation(ce, chatId)

			fake := &fakeQueryMetadataKeyspace{behavior: tt.behavior, fetchBehavior: tt.fetchBehavior}
			err := persistPausedChat(chatId, ce, fake, datastore.NULL_QUERY_CONTEXT)

			if tt.wantErrCode == 0 {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error code %v, got nil", tt.wantErrCode)
				} else if err.Code() != tt.wantErrCode {
					t.Fatalf("expected error code %v, got %v (%v)", tt.wantErrCode, err.Code(), err)
				}
			}

			if ce.Paused != tt.wantPaused {
				t.Errorf("ce.Paused = %v, want %v", ce.Paused, tt.wantPaused)
			}
			stillLive := GetConversation(chatId) != nil
			if stillLive == tt.wantPaused {
				t.Errorf("chat live-cache presence = %v, want live iff !wantPaused (wantPaused=%v)", stillLive, tt.wantPaused)
			}
			if fake.attempt != tt.wantAttempts {
				t.Errorf("Insert call count = %d, want %d", fake.attempt, tt.wantAttempts)
			}

			DeleteConversation(chatId)
		})
	}
}

// ─── claimChatDocument / completeChatClaim: RESUME's claim-Update and claim- ──────
// ─── completion-Delete retry/disambiguation (mirrors persistPausedChat's Insert) ──

func TestClaimChatDocument(t *testing.T) {
	withFastRetry(t)
	updateSuccess := func(attempt int) (int, value.Pairs, errors.Errors) { return 1, nil, nil }

	tests := []struct {
		name         string
		behavior     func(attempt int) (int, value.Pairs, errors.Errors)
		wantClaimed  bool
		wantRetry    bool
		wantErrCode  errors.ErrorCode // 0 means expect nil error
		wantAttempts int
	}{
		{
			name:         "immediate success",
			behavior:     updateSuccess,
			wantClaimed:  true,
			wantAttempts: 1,
		},
		{
			name: "NOT_MY_VBUCKET then success",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				if attempt == 1 {
					return 0, nil, notMyVbucketError()
				}
				return updateSuccess(attempt)
			},
			wantClaimed:  true,
			wantAttempts: 2,
		},
		{
			name:         "CAS mismatch signals retry rather than failure",
			behavior:     func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, casMismatchError() },
			wantRetry:    true,
			wantAttempts: 1,
		},
		{
			name:         "SYNC_WRITE_IN_PROGRESS signals retry rather than failure",
			behavior:     func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, syncWriteInProgressError() },
			wantRetry:    true,
			wantAttempts: 1,
		},
		{
			// The gap this test closes: an ambiguous outcome must be treated the same
			// as CAS mismatch (re-fetch and re-check the claimer field), not as a hard
			// failure -- our own update may have actually landed.
			name:         "SYNC_WRITE_AMBIGUOUS signals retry rather than failure",
			behavior:     func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, syncWriteAmbiguousError() },
			wantRetry:    true,
			wantAttempts: 1,
		},
		{
			name: "non-retryable failure fails immediately",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				return 0, nil, errors.Errors{errors.NewOtherDatastoreError(nil, "some fatal error")}
			},
			wantErrCode:  errors.E_NL_CHAT_RESUME_FAILED,
			wantAttempts: 1,
		},
		{
			name:         "exhausted retries without a definitive outcome",
			behavior:     func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, notMyVbucketError() },
			wantAttempts: maxRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeQueryMetadataKeyspace{updateBehavior: tt.behavior}
			udpairs := value.Pairs{{Name: "aichat::chat1", Value: value.NewValue(map[string]interface{}{})}}
			claimed, retryClaim, err := claimChatDocument("chat1", "aichat::chat1", fake, datastore.NULL_QUERY_CONTEXT, udpairs)

			if tt.wantErrCode == 0 {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			} else if err == nil || err.Code() != tt.wantErrCode {
				t.Fatalf("expected error code %v, got %v", tt.wantErrCode, err)
			}
			if claimed != tt.wantClaimed {
				t.Errorf("claimed = %v, want %v", claimed, tt.wantClaimed)
			}
			if retryClaim != tt.wantRetry {
				t.Errorf("retryClaim = %v, want %v", retryClaim, tt.wantRetry)
			}
			if fake.updateAttempt != tt.wantAttempts {
				t.Errorf("Update call count = %d, want %d", fake.updateAttempt, tt.wantAttempts)
			}
		})
	}
}

func TestCompleteChatClaim(t *testing.T) {
	withFastRetry(t)
	deleteSuccess := func(attempt int) (int, value.Pairs, errors.Errors) { return 1, nil, nil }

	tests := []struct {
		name         string
		behavior     func(attempt int) (int, value.Pairs, errors.Errors)
		wantErrCode  errors.ErrorCode // 0 means expect nil error
		wantAttempts int
	}{
		{
			name:         "immediate success",
			behavior:     deleteSuccess,
			wantAttempts: 1,
		},
		{
			name: "SYNC_WRITE_IN_PROGRESS then success",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				if attempt == 1 {
					return 0, nil, syncWriteInProgressError()
				}
				return deleteSuccess(attempt)
			},
			wantAttempts: 2,
		},
		{
			name: "SYNC_WRITE_AMBIGUOUS then success",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				if attempt == 1 {
					return 0, nil, syncWriteAmbiguousError()
				}
				return deleteSuccess(attempt)
			},
			wantAttempts: 2,
		},
		{
			name: "non-retryable failure fails immediately",
			behavior: func(attempt int) (int, value.Pairs, errors.Errors) {
				return 0, nil, errors.Errors{errors.NewOtherDatastoreError(nil, "some fatal error")}
			},
			wantErrCode:  errors.E_NL_CHAT_RESUME_FAILED,
			wantAttempts: 1,
		},
		{
			name:         "exhausted retries with persistent ambiguity fails",
			behavior:     func(attempt int) (int, value.Pairs, errors.Errors) { return 0, nil, syncWriteAmbiguousError() },
			wantErrCode:  errors.E_NL_CHAT_RESUME_FAILED,
			wantAttempts: maxRetry,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeQueryMetadataKeyspace{behavior: tt.behavior}
			err := completeChatClaim("chat1", "aichat::chat1", fake, datastore.NULL_QUERY_CONTEXT)

			if tt.wantErrCode == 0 {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			} else if err == nil || err.Code() != tt.wantErrCode {
				t.Fatalf("expected error code %v, got %v", tt.wantErrCode, err)
			}
			if fake.attempt != tt.wantAttempts {
				t.Errorf("Delete call count = %d, want %d", fake.attempt, tt.wantAttempts)
			}
		})
	}
}

// capellaPickProviderAndModel never picks a model: an empty model is passed
// through for iQ to apply its default, and a given model is matched against the
// listed models only to use the listed spelling (an unlisted model is passed
// through as given). The provider defaults to openai, else the first enabled one.
func TestCapellaPickProviderAndModel(t *testing.T) {
	providers := []capellaModelProvider{
		{ID: "bedrock", Models: []string{"us.anthropic.claude-sonnet-4-5-20250929-v1:0"}, Enabled: true},
		{ID: "openai", Models: []string{"gpt-3.5-turbo", "GPT-4o"}, Enabled: true},
		{ID: "gemini", Enabled: false},
	}
	noOpenAI := []capellaModelProvider{
		{ID: "openai", Enabled: false},
		{ID: "bedrock", Enabled: true},
	}
	cases := []struct {
		name         string
		vendor       string
		model        string
		providers    []capellaModelProvider
		wantCode     errors.ErrorCode
		wantProvider string
		wantModel    string
	}{
		{"model without vendor", "", "gpt-4o", providers, errors.E_NL_MODEL_WITHOUT_VENDOR, "", ""},
		{"no providers", "openai", "", nil, errors.E_NL_NO_VENDORS_AVAILABLE, "", ""},
		{"unknown vendor", "mistral", "", providers, errors.E_NL_VENDOR_NOT_SUPPORTED, "", ""},
		{"disabled vendor", "gemini", "", providers, errors.E_NL_VENDOR_NOT_ENABLED, "", ""},
		{"none enabled", "", "", []capellaModelProvider{{ID: "openai"}}, errors.E_NL_NO_VENDORS_ENABLED, "", ""},
		{"neither given: openai preferred, no model", "", "", providers, 0, "openai", ""},
		{"neither given: first enabled when openai is disabled", "", "", noOpenAI, 0, "bedrock", ""},
		{"vendor only: no model", "Bedrock", "", providers, 0, "bedrock", ""},
		{"listed model", "openai", "gpt-3.5-turbo", providers, 0, "openai", "gpt-3.5-turbo"},
		{"case-insensitive match uses listed spelling", "openai", "gpt-4o", providers, 0, "openai", "GPT-4o"},
		{"unlisted model passes through", "openai", "gpt-5.4-2026-03-05", providers, 0, "openai",
			"gpt-5.4-2026-03-05"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			provider, model, err := capellaPickProviderAndModel(c.vendor, c.model, c.providers)
			if c.wantCode != 0 {
				if err == nil || err.Code() != c.wantCode {
					t.Fatalf("expected error code %v, got %v", c.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if provider != c.wantProvider || model != c.wantModel {
				t.Fatalf("got (%q, %q), want (%q, %q)", provider, model, c.wantProvider, c.wantModel)
			}
		})
	}
}

// A chat turn's vendor and model are always applied as a pair, so one vendor's
// model is never sent to another vendor. Each case runs the turn as
// ProcessCapellaConversationalRequest does: pick the pair to resolve, resolve it
// against the listed providers, then build the iterative prompt.
func TestCapellaChatTurnVendorModel(t *testing.T) {
	providers := []capellaModelProvider{
		{ID: "openai", Models: []string{"gpt-5.4", "gpt-4o"}, Enabled: true},
		{ID: "bedrock", Models: []string{"us.anthropic.claude-sonnet-5"}, Enabled: true},
	}
	cases := []struct {
		name                  string
		prevProvider          string // set when the previous turn was on the direct path
		prevVendor, prevModel string
		vendor, model         string
		wantVendor, wantModel string
		wantTemperature       float64
	}{
		{"no vendor or model reuses the chat's pair",
			"", "bedrock", "us.anthropic.claude-sonnet-5", "", "", "bedrock", "us.anthropic.claude-sonnet-5", 0},
		{"no vendor or model reuses the chat's empty model",
			"", "bedrock", "", "", "", "bedrock", "", 0},
		{"vendor only drops the previous vendor's model and temperature",
			"", "openai", "gpt-5.4", "bedrock", "", "bedrock", "", 0},
		{"new pair replaces the chat's pair",
			"", "bedrock", "us.anthropic.claude-sonnet-5", "openai", "GPT-5.4", "openai", "gpt-5.4", 1},
		{"after a direct turn, no vendor or model starts fresh instead of reusing its model",
			"gemini", "", "gemini-3.8-flash", "", "", "openai", "", 0},
		{"after a direct turn, vendor only does not keep its model",
			"openai", "", "gpt-5.4", "bedrock", "", "bedrock", "", 0},
	}
	for _, c := range cases {
		for _, output := range []string{"sql", "jsudf"} {
			chat := &prompt{
				Provider: c.prevProvider,
				Vendor:   c.prevVendor,
				CompletionSettings: completionSettings{
					Model:       c.prevModel,
					Temperature: capellaGetTemperatureForModel(c.prevVendor, c.prevModel),
				},
			}
			vendor, model := capellaChatTurnVendorModel(&ChatEntry{prompt: chat}, c.vendor, c.model)
			provider, model, err := capellaPickProviderAndModel(vendor, model, providers)
			if err != nil {
				t.Fatalf("%s (%s): unexpected error: %v", c.name, output, err)
			}
			if output == "sql" {
				chat = newCapellaSQLIterativePrompt(chat, "q", "", false, provider, model)
			} else {
				chat = newCapellaJSUDFIterativePrompt(chat, "q", "", provider, model)
			}
			// The whole prompt is sent to iQ, so no direct-path provider may remain.
			if chat.Provider != "" || chat.Vendor != c.wantVendor || chat.CompletionSettings.Model != c.wantModel ||
				chat.CompletionSettings.Temperature != c.wantTemperature {
				t.Errorf("%s (%s): sent provider=%q vendor=%q model=%q temperature=%v, want no provider, vendor=%q model=%q temperature=%v",
					c.name, output, chat.Provider, chat.Vendor, chat.CompletionSettings.Model,
					chat.CompletionSettings.Temperature, c.wantVendor, c.wantModel, c.wantTemperature)
			}
		}
	}
}

// The modelProviders list is reused within the TTL, per organization and
// credential; an empty list is never cached.
func TestCapellaModelProvidersCache(t *testing.T) {
	providers := []capellaModelProvider{{ID: "openai", Enabled: true}}
	now := time.Now()
	key := capellaModelProvidersCacheKey("org-cache-test", "user:pass")
	capellaCacheModelProviders(key, providers, now)

	if got := capellaCachedModelProviders(key, now.Add(capellaModelProvidersTTL-time.Second)); len(got) != 1 {
		t.Errorf("within the TTL: got %v, want the cached list", got)
	}
	if got := capellaCachedModelProviders(key, now.Add(capellaModelProvidersTTL)); got != nil {
		t.Errorf("at the TTL: got %v, want nil (refetch)", got)
	}
	for _, other := range []string{
		capellaModelProvidersCacheKey("org-cache-test", "other:pass"),
		capellaModelProvidersCacheKey("org-cache-test-2", "user:pass"),
	} {
		if got := capellaCachedModelProviders(other, now); got != nil {
			t.Errorf("key %q: got %v, want nil (another org or credential)", other, got)
		}
	}

	emptyKey := capellaModelProvidersCacheKey("org-cache-test-empty", "user:pass")
	capellaCacheModelProviders(emptyKey, []capellaModelProvider{}, now)
	if got := capellaCachedModelProviders(emptyKey, now); got != nil {
		t.Errorf("empty list: got %v, want nil (not cached)", got)
	}
}

// A turn whose model the vendor rejects (19258) puts the chat's previous pair
// back, so the next turn naming neither does not reuse the rejected pair.
func TestCapellaSaveVendorModelRestoresAfterRejectedModel(t *testing.T) {
	cases := []struct {
		name string
		prev prompt
	}{
		{"previous capella pair", prompt{Vendor: "bedrock",
			CompletionSettings: completionSettings{Model: "us.anthropic.claude-sonnet-5"}}},
		{"previous capella default model", prompt{Vendor: "openai"}},
		{"previous direct turn", prompt{Provider: "gemini",
			CompletionSettings: completionSettings{Model: "gemini-3.8-flash"}}},
	}
	for _, c := range cases {
		chat := c.prev
		chat.CompletionSettings.Temperature = capellaGetTemperatureForModel(chat.Vendor, chat.CompletionSettings.Model)
		want := chat
		wantVendor, wantModel := capellaChatTurnVendorModel(&ChatEntry{prompt: &chat}, "", "")

		restore := capellaSaveVendorModel(&chat)
		newCapellaSQLIterativePrompt(&chat, "q", "", false, "openai", "gpt-4o-2024-05-13")
		restore()

		if chat.Provider != want.Provider || chat.Vendor != want.Vendor ||
			chat.CompletionSettings != want.CompletionSettings {
			t.Errorf("%s: restored provider=%q vendor=%q settings=%+v, want provider=%q vendor=%q settings=%+v",
				c.name, chat.Provider, chat.Vendor, chat.CompletionSettings,
				want.Provider, want.Vendor, want.CompletionSettings)
		}
		if vendor, model := capellaChatTurnVendorModel(&ChatEntry{prompt: &chat}, "", ""); vendor != wantVendor ||
			model != wantModel {
			t.Errorf("%s: next turn resolves vendor=%q model=%q, want vendor=%q model=%q",
				c.name, vendor, model, wantVendor, wantModel)
		}
	}
}

// Summarizing replaces the prompt with a summary. The Capella vendor and model of
// the last turn are kept, persisted with a paused chat, and reused by the next
// turn naming neither, until that turn rebuilds the prompt; a pair left by a
// direct turn is not kept.
func TestCapellaChatTurnVendorModelAfterSummary(t *testing.T) {
	cases := []struct {
		name                  string
		prev                  prompt
		wantVendor, wantModel string
	}{
		{"Capella turn", prompt{Vendor: "bedrock", CompletionSettings: completionSettings{Model: "us.anthropic.claude-sonnet-5"}},
			"bedrock", "us.anthropic.claude-sonnet-5"},
		{"Capella turn with iQ's default model", prompt{Vendor: "bedrock"}, "bedrock", ""},
		{"direct turn", prompt{Provider: "gemini", CompletionSettings: completionSettings{Model: "gemini-3.8-flash"}}, "", ""},
	}
	for _, c := range cases {
		prev := c.prev
		ce := &ChatEntry{Keyspaces: testPaths(), prompt: &prev}
		ce.summarized("prior context")
		if ce.prompt != nil || ce.Summary != "prior context" {
			t.Fatalf("%s: prompt not replaced by the summary: prompt=%v summary=%q", c.name, ce.prompt, ce.Summary)
		}

		b, err := json.Marshal(ce)
		if err != nil {
			t.Fatalf("%s: marshal: %v", c.name, err)
		}
		if c.wantVendor == "" && strings.Contains(string(b), `"vendor"`) {
			t.Fatalf("%s: vendor persisted after a direct turn: %s", c.name, b)
		}
		var got ChatEntry
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("%s: unmarshal: %v", c.name, err)
		}

		for _, entry := range []*ChatEntry{ce, &got} {
			if vendor, model := capellaChatTurnVendorModel(entry, "", ""); vendor != c.wantVendor || model != c.wantModel {
				t.Errorf("%s: reused vendor=%q model=%q, want vendor=%q model=%q",
					c.name, vendor, model, c.wantVendor, c.wantModel)
			}
			// A turn naming either is not mixed with the kept pair.
			if vendor, model := capellaChatTurnVendorModel(entry, "openai", ""); vendor != "openai" || model != "" {
				t.Errorf("%s: vendor-only turn resolved vendor=%q model=%q, want openai and no model", c.name, vendor, model)
			}
		}

		// A turn that fails stores no prompt (no content), so the summary and the
		// kept pair stay for the next turn.
		ce.Id = "summary-test-" + c.name
		rebuilt := &prompt{Vendor: "openai"}
		completeConversationPromptLocked("", ce, rebuilt)
		if ce.prompt != nil || ce.Summary != "prior context" || ce.vendor != c.wantVendor || ce.model != c.wantModel {
			t.Errorf("%s: failed turn changed the summary state: prompt=%v summary=%q vendor=%q model=%q",
				c.name, ce.prompt, ce.Summary, ce.vendor, ce.model)
		}

		// Storing the rebuilt prompt drops them: the summary is now part of it.
		completeConversationPromptLocked("SELECT 1", ce, rebuilt)
		ce.stopInactivityTimer()
		naturalchatHistory.Delete(ce.Id, nil)
		if ce.prompt != rebuilt || ce.Summary != "" || ce.vendor != "" || ce.model != "" {
			t.Errorf("%s: summary state left after the rebuilt prompt was stored: summary=%q vendor=%q model=%q",
				c.name, ce.Summary, ce.vendor, ce.model)
		}
	}
}

// The 19258 message names the model, or says the default model when the request
// named none (iQ applied its default).
func TestCapellaModelNotAvailableError(t *testing.T) {
	for model, want := range map[string]string{
		"gpt-4o": `Model "gpt-4o" is not available for vendor "openai"`,
		"":       `The default model is not available for vendor "openai"`,
	} {
		err := capellaModelNotAvailableError(model, "openai", map[string]interface{}{"code": "model_not_found"})
		if err.Code() != errors.E_NL_MODEL_NOT_AVAILABLE || !strings.Contains(err.Error(), want) {
			t.Errorf("model %q: got %v, want 19258 containing %q", model, err, want)
		}
	}
}

// The REST "natural" parameter check and the Capella path name missing
// credentials through the same helper, in the same order.
func TestCapellaMissingCreds(t *testing.T) {
	cases := []struct {
		cred, orgId string
		want        []string
	}{
		{"c", "o", nil},
		{"", "o", []string{`"natural_cred"`}},
		{"c", "", []string{`"natural_orgid"`}},
		{"", "", []string{`"natural_cred"`, `"natural_orgid"`}},
	}
	for _, c := range cases {
		if got := CapellaMissingCreds(c.cred, c.orgId); !slices.Equal(got, c.want) {
			t.Errorf("CapellaMissingCreds(%q, %q) = %q, want %q", c.cred, c.orgId, got, c.want)
		}
		err := CapellaCheckCreds(c.cred, c.orgId)
		if (err == nil) != (len(c.want) == 0) {
			t.Fatalf("CapellaCheckCreds(%q, %q) = %v", c.cred, c.orgId, err)
		}
		if err != nil && (err.Code() != errors.E_NL_MISSING_NL_PARAM || !strings.Contains(err.Error(), strings.Join(c.want, ","))) {
			t.Errorf("CapellaCheckCreds(%q, %q) = %v, want 19213 naming %s", c.cred, c.orgId, err, strings.Join(c.want, ","))
		}
	}
}

// A direct turn applies natural_config's provider and model as a pair and clears
// any Capella vendor, so a later Capella turn does not reuse a direct-path model.
func TestDirectChatTurnProviderModel(t *testing.T) {
	for _, output := range []string{"sql", "jsudf"} {
		// The previous turn was on the Capella path.
		chat := &prompt{
			Vendor: "openai",
			CompletionSettings: completionSettings{
				Model:       "gpt-5.4",
				Temperature: capellaGetTemperatureForModel("openai", "gpt-5.4"),
			},
		}
		if output == "sql" {
			chat = newDirectSQLIterativePrompt(chat, "q", "", false, "gemini", "gemini-3.8-flash")
		} else {
			chat = newDirectJSUDFIterativePrompt(chat, "q", "", "gemini", "gemini-3.8-flash")
		}
		if chat.Provider != "gemini" || chat.Vendor != "" || chat.CompletionSettings.Model != "gemini-3.8-flash" ||
			chat.CompletionSettings.Temperature != 0 {
			t.Errorf("%s: got provider=%q vendor=%q model=%q temperature=%v, want provider=gemini, no vendor, "+
				"model=gemini-3.8-flash temperature=0", output, chat.Provider, chat.Vendor,
				chat.CompletionSettings.Model, chat.CompletionSettings.Temperature)
		}
		if vendor, model := capellaChatTurnVendorModel(&ChatEntry{prompt: chat}, "", ""); vendor != "" || model != "" {
			t.Errorf("%s: a Capella turn after this would reuse vendor=%q model=%q, want nothing", output, vendor, model)
		}
	}
}

// With no model, no temperature is chosen, so none is sent (omitempty) and iQ's
// default model cannot be given a temperature it rejects.
func TestCapellaNoModelSendsNoTemperature(t *testing.T) {
	if got := capellaGetTemperatureForModel("openai", ""); got != 0 {
		t.Fatalf("got temperature %v for an empty model, want 0", got)
	}
	b, err := json.Marshal(completionSettings{Model: "", Temperature: capellaGetTemperatureForModel("openai", "")})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "temperature") {
		t.Fatalf("temperature sent with no model: %s", b)
	}
}

// gpt-5 models only accept temperature 1; the model carries the listed spelling,
// so the prefix check must not depend on case.
func TestCapellaGetTemperatureForModel(t *testing.T) {
	cases := []struct {
		provider, model string
		want            float64
	}{
		{"openai", "gpt-5.4", 1},
		{"openai", "GPT-5.4", 1},
		{"openai", "gpt-4o", 0},
		{"bedrock", "us.anthropic.claude-sonnet-5", 0},
	}
	for _, c := range cases {
		if got := capellaGetTemperatureForModel(c.provider, c.model); got != c.want {
			t.Errorf("%s/%s: got %v, want %v", c.provider, c.model, got, c.want)
		}
	}
}
