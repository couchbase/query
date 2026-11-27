//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"net/url"
	"strings"
	"testing"

	"github.com/couchbase/query/value"
)

// Remote keys go into the URL path of the request to the remote node, followed by the impersonate parameter
// that restricts the remote node to the user's own entries: a key that ends the path ('?', '#') must be escaped,
// or it can drop or override impersonate and reach other users' entries
func TestRemoteKeysEscaped(t *testing.T) {
	recorder := newTwoNodes(t)

	localKeys := []string{"id#", "id?impersonate=&", "id/../other"}
	keys := make([]string, len(localKeys))
	deletes := make(value.Pairs, len(localKeys))
	for i, k := range localKeys {
		keys[i] = "[remote]" + k
		deletes[i] = value.Pair{Name: keys[i]}
	}

	for _, name := range []string{KEYSPACE_NAME_ACTIVE} {
		ks := newTestKeyspace(t, name, nil)
		recorder.keys = nil
		ks.Fetch(keys, map[string]value.AnnotatedValue{}, newUserContext("alice"), nil, nil, false)
		ks.Delete(deletes, newUserContext("alice"), false)

		if len(recorder.keys) != 2*len(localKeys) {
			t.Fatalf("%v: remote keys %v, expected a fetch and a delete of %v", name, recorder.keys, localKeys)
		}
		for i, sent := range recorder.keys {
			expected := localKeys[i%len(localKeys)]
			if strings.ContainsAny(sent, "?#/") {
				t.Errorf("%v: remote key %q for %q is not escaped", name, sent, expected)
			}
			if unescaped, err := url.PathUnescape(sent); err != nil || unescaped != expected {
				t.Errorf("%v: remote key %q unescapes to %q (%v), expected %q", name, sent, unescaped, err, expected)
			}
		}
	}
}
