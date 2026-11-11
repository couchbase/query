//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package server

import (
	"strings"
	"testing"

	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

func formatArgs(t *testing.T, re *RequestLogEntry) (interface{}, interface{}) {
	m, ok := re.Format(true, false, util.DEFAULT).(map[string]interface{})
	if !ok {
		t.Fatalf("Format did not return a map")
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal formatted entry: %v", err)
	}
	var out map[string]interface{}
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal formatted entry: %v", err)
	}
	return out["namedArgs"], out["positionalArgs"]
}

// The entry must keep its own copy of the parameters: changing the request's
// buffer afterwards must not change what the entry reports.
func TestCompletedArgsAreDetached(t *testing.T) {
	body := []byte(`{"a":"original"}`)
	named := map[string]value.Value{"$p": value.NewParsedValue(body, false)}
	positional := value.Values{value.NewParsedValue(body, false)}

	re := &RequestLogEntry{}
	re.namedArgs = marshalArgs(named, 0)
	re.positionalArgs = marshalArgs(positional, 0)

	copy(body, []byte(`{"a":"CHANGED!!"}`))

	n, p := formatArgs(t, re)
	nm, ok := n.(map[string]interface{})
	if !ok {
		t.Fatalf("namedArgs is not an object: %#v", n)
	}
	if got := nm["$p"].(map[string]interface{})["a"]; got != "original" {
		t.Fatalf("namedArgs follows the request buffer: got %v", got)
	}
	pa, ok := p.([]interface{})
	if !ok || len(pa) != 1 {
		t.Fatalf("positionalArgs is not a one element array: %#v", p)
	}
	if got := pa[0].(map[string]interface{})["a"]; got != "original" {
		t.Fatalf("positionalArgs follows the request buffer: got %v", got)
	}
}

func TestCompletedArgsCapped(t *testing.T) {
	big := value.NewValue(strings.Repeat("x", _MAX_ARGS_SIZE+1))
	named := map[string]value.Value{"$big": big}

	// over the limit by the size estimate: never serialized
	re := &RequestLogEntry{}
	re.namedArgs = marshalArgs(named, big.Size())
	n, _ := formatArgs(t, re)
	if w, ok := n.(map[string]interface{})["WARNING"].(string); !ok || !strings.Contains(w, "exceed maximum permitted") {
		t.Fatalf("expected a size warning, got %#v", n)
	}

	// under the limit by the estimate, but over it once serialized
	re.namedArgs = marshalArgs(named, 0)
	n, _ = formatArgs(t, re)
	if _, ok := n.(map[string]interface{})["WARNING"]; !ok {
		t.Fatalf("expected a size warning after marshalling, got %#v", n)
	}
	if len(re.namedArgs) > 1024 {
		t.Fatalf("capped entry still holds %d bytes", len(re.namedArgs))
	}
}

func TestCompletedArgsAbsent(t *testing.T) {
	re := &RequestLogEntry{}
	n, p := formatArgs(t, re)
	if n != nil || p != nil {
		t.Fatalf("expected no parameters, got %#v / %#v", n, p)
	}
}

// requests without named parameters kept reporting "namedArgs": {}
func TestCompletedNamedArgsEmpty(t *testing.T) {
	re := &RequestLogEntry{}
	re.namedArgs = namedArgsCopy(map[string]value.Value{})
	n, _ := formatArgs(t, re)
	if m, ok := n.(map[string]interface{}); !ok || len(m) != 0 {
		t.Fatalf("expected an empty namedArgs object, got %#v", n)
	}

	re.namedArgs = namedArgsCopy(nil)
	n, _ = formatArgs(t, re)
	if n != nil {
		t.Fatalf("expected no namedArgs, got %#v", n)
	}
}

func TestCompletedArgsUnserializable(t *testing.T) {
	re := &RequestLogEntry{}
	re.positionalArgs = marshalArgs(make(chan int), 0)
	_, p := formatArgs(t, re)
	if w, ok := p.(map[string]interface{})["WARNING"].(string); !ok || !strings.Contains(w, "could not be serialized") {
		t.Fatalf("expected a serialization warning, got %#v", p)
	}
}
