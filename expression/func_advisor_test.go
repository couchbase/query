// Copyright 2026-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in
// that file, in accordance with the Business Source License, use of this
// software will be governed by the Apache License, Version 2.0, included in
// the file licenses/APL2.txt.

// Tests for func_advisor.go, focused on the N1QL-injection hardening for
// MB-73772:
//   - getSession() must reject anything that is not a well-formed UUID.
//   - queryContext() and analyzeWorkload() must pass tenant-controlled input
//     (query_context, profile) as named parameters, never concatenate it into
//     the statement text.

package expression

import (
	"strings"
	"testing"

	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// advisorTestContext embeds the shared testContextStub (defined in
// cred_handler_test.go) and overrides only QueryContextParts.
type advisorTestContext struct {
	*testContextStub
	parts []string
}

func (c *advisorTestContext) QueryContextParts() []string { return c.parts }

func newAdvisor() *Advisor {
	return NewAdvisor(NewConstant(value.NewValue("x"))).(*Advisor)
}

func sessionMap(v interface{}) map[string]interface{} {
	return map[string]interface{}{"session": value.NewValue(v)}
}

func TestAdvisorIsSessionNameValid(t *testing.T) {
	valid := []string{
		"1d712a95-6eb9-4771-bead-73d6a61789c5",
		"00000000-0000-0000-0000-000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	for _, s := range valid {
		if !isSessionNameValid(s) {
			t.Errorf("isSessionNameValid(%q) = false, want true", s)
		}
	}

	invalid := []string{
		"",
		"not-a-uuid",
		`x" OR 1=1 OR name="x`,
		"1d712a95-6eb9-4771-bead-73d6a61789c5 OR 1=1", // trailing junk
		"1d712a956eb94771bead73d6a61789c5",            // no dashes
		"1d712a95-6eb9-4771-bead-73d6a61789c",         // too short (35)
		"1d712a95-6eb9-4771-bead-73d6a61789c55",       // too long (37)
		"1d712a95_6eb9_4771_bead_73d6a61789c5",        // wrong separators
		"g1712a95-6eb9-4771-bead-73d6a61789c5",        // non-hex 'g'
		`1d712a95-6eb9-4771-bead-73d6a61789"5`,        // embedded quote
		"1D712A95-6EB9-4771-BEAD-73D6A61789C5",        // uppercase (caller lower-cases first)
	}
	for _, s := range invalid {
		if isSessionNameValid(s) {
			t.Errorf("isSessionNameValid(%q) = true, want false", s)
		}
	}
}

func TestAdvisorGetSession(t *testing.T) {
	adv := newAdvisor()

	// A real UUIDV4() value must be accepted.
	uuid, err := util.UUIDV4()
	if err != nil {
		t.Fatalf("UUIDV4: %v", err)
	}
	if got, err := adv.getSession(sessionMap(uuid)); err != nil || got != uuid {
		t.Errorf("getSession(real uuid) = (%q, %v), want (%q, nil)", got, err, uuid)
	}

	// Uppercase UUID is accepted and lower-cased.
	upper := "1D712A95-6EB9-4771-BEAD-73D6A61789C5"
	lower := "1d712a95-6eb9-4771-bead-73d6a61789c5"
	if got, err := adv.getSession(sessionMap(upper)); err != nil || got != lower {
		t.Errorf("getSession(uppercase) = (%q, %v), want (%q, nil)", got, err, lower)
	}

	// Injection payloads and malformed values are rejected.
	for _, bad := range []string{
		`x" OR 1=1 OR name="x`,
		`x" OR (SELECT RAW ssn FROM default:tenanta USE KEYS "k")[0] LIKE "1%" OR name="x`,
		"not-a-uuid",
		"",
	} {
		if got, err := adv.getSession(sessionMap(bad)); err == nil {
			t.Errorf("getSession(%q) = (%q, nil), want error", bad, got)
		}
	}

	// Missing 'session'.
	if _, err := adv.getSession(map[string]interface{}{}); err == nil {
		t.Error("getSession(missing) = nil error, want error")
	}
	// Non-string 'session'.
	if _, err := adv.getSession(map[string]interface{}{"session": value.NewValue(42)}); err == nil {
		t.Error("getSession(number) = nil error, want error")
	}
}

func TestAdvisorQueryContextParameterized(t *testing.T) {
	// A benign query context.
	ctx := &advisorTestContext{parts: []string{"default", "tenanta"}}
	args := map[string]value.Value{}
	clause := queryContext(ctx, args)

	if !strings.Contains(clause, "$av_qc") || !strings.Contains(clause, "$av_qclike") {
		t.Errorf("clause missing named parameters: %q", clause)
	}
	if got := args["av_qc"].ToString(); got != "default:tenanta" {
		t.Errorf("av_qc = %q, want %q", got, "default:tenanta")
	}
	if got := args["av_qclike"].ToString(); got != "default:tenanta.%" {
		t.Errorf("av_qclike = %q, want %q", got, "default:tenanta.%")
	}

	// A malicious query context value must land ONLY in the parameter, never in
	// the statement text.
	evil := &advisorTestContext{parts: []string{"default", `evil" OR 1=1 OR queryContext="`}}
	eargs := map[string]value.Value{}
	eclause := queryContext(evil, eargs)
	if strings.Contains(eclause, "1=1") || strings.Contains(eclause, `"`) {
		t.Errorf("tenant value leaked into statement text: %q", eclause)
	}
	if !strings.Contains(eargs["av_qc"].ToString(), "1=1") {
		t.Errorf("av_qc did not carry the raw value: %q", eargs["av_qc"].ToString())
	}

	// Fewer than 2 parts yields no filter and no args.
	short := &advisorTestContext{parts: []string{"default"}}
	sargs := map[string]value.Value{}
	if clause := queryContext(short, sargs); clause != "" || len(sargs) != 0 {
		t.Errorf("short context: clause=%q args=%v, want empty", clause, sargs)
	}
}

func TestAdvisorAnalyzeWorkloadParameterizesProfile(t *testing.T) {
	ctx := &advisorTestContext{parts: []string{"default", "tenanta"}}

	// profile is tenant-controlled: must be a parameter, not concatenated.
	profile := `john" OR 1=1`
	q, args := analyzeWorkload(profile, "", 10, 100, ctx)

	if !strings.Contains(q, "users LIKE $av_profile") {
		t.Errorf("statement does not parameterize profile: %q", q)
	}
	if strings.Contains(q, "1=1") || strings.Contains(q, profile) {
		t.Errorf("profile leaked into statement text: %q", q)
	}
	if got, want := args["av_profile"].ToString(), "%"+profile+"%"; got != want {
		t.Errorf("av_profile = %q, want %q", got, want)
	}

	// Empty profile: no LIKE clause and no parameter.
	q2, args2 := analyzeWorkload("", "", 10, 100, ctx)
	if strings.Contains(q2, "users LIKE") {
		t.Errorf("empty profile still produced a LIKE clause: %q", q2)
	}
	if _, ok := args2["av_profile"]; ok {
		t.Error("empty profile still set av_profile")
	}
}
