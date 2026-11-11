//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/couchbase/query/util"
)

// After OwnStrings the prepared statement must not share memory with the request body
// its strings were sliced from.
// The encoded plan can only come with the request when encoded plans are enabled, and is
// only copied then.
func TestPreparedOwnStrings(t *testing.T) {
	featureControls := util.GetN1qlFeatureControl()
	defer util.SetN1qlFeatureControl(featureControls)

	for _, encodedPlans := range []bool{true, false} {
		if encodedPlans {
			util.SetN1qlFeatureControl(featureControls &^ util.N1QL_ENCODED_PLAN)
		} else {
			util.SetN1qlFeatureControl(featureControls | util.N1QL_ENCODED_PLAN)
		}
		testPreparedOwnStrings(t, encodedPlans)
	}
}

func testPreparedOwnStrings(t *testing.T, encodedPlans bool) {
	body := `{"prepared":"p1","statement":"SELECT RAW 1","encoded_plan":"H4sIAAAAAAAA","namespace":"default",` +
		`"query_context":"default:b1.s1","tenant":"b1","$data":"..."}`
	inBody := func(s string) bool {
		if len(s) == 0 {
			return false
		}
		p := uintptr(unsafe.Pointer(unsafe.StringData(s)))
		b := uintptr(unsafe.Pointer(unsafe.StringData(body)))
		return p >= b && p < b+uintptr(len(body))
	}
	field := func(name string) string {
		i := strings.Index(body, `"`+name+`":"`) + len(name) + 4
		return body[i : i+strings.IndexByte(body[i:], '"')]
	}

	prep := &Prepared{}
	prep.SetName(field("prepared"))
	prep.SetText(field("statement"))
	prep.SetEncodedPlan(field("encoded_plan"))
	prep.SetNamespace(field("namespace"))
	prep.SetQueryContext(field("query_context"))
	prep.tenant = field("tenant")

	get := map[string]func() string{
		"prepared": prep.Name, "statement": prep.Text, "encoded_plan": prep.EncodedPlan,
		"namespace": prep.Namespace, "query_context": prep.QueryContext, "tenant": prep.Tenant,
	}
	for what, f := range get {
		if !inBody(f()) {
			t.Fatalf("test setup: %s is expected to point into the body", what)
		}
	}

	prep.OwnStrings()

	for what, f := range get {
		copied := what != "encoded_plan" || encodedPlans
		if in := inBody(f()); in == copied {
			t.Fatalf("encoded plans enabled %v: %s points into the request body: %v, expected %v",
				encodedPlans, what, in, !copied)
		}
		if f() != field(what) {
			t.Fatalf("%s changed: %q, expected %q", what, f(), field(what))
		}
	}
}
