//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

// We implement here sync types that in sync don't do exactly what we want.
// Our implementation tends to be leaner too.

import (
	"net/http"
)

// same as Header.Get, but withou canonicalising the key
func HeaderGet(h http.Header, k string) string {
	s := h[k]
	if len(s) == 0 {
		return ""
	}
	return s[0]
}
