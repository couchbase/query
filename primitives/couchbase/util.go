//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"fmt"
	"net/url"
)

// ParseURL is a wrapper around url.Parse with some sanity-checking
func ParseURL(urlStr string) (result *url.URL, err error) {
	result, err = url.Parse(urlStr)
	if result != nil && result.Scheme == "" {
		result = nil
		err = fmt.Errorf("invalid URL <%s>", urlStr)
	}
	return
}
