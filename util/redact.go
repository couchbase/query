//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

func Redacted(in string, redact bool) string {
	if redact {
		return "<ud>" + in + "</ud>"
	}
	return in
}

func InterfaceRedacted(in interface{}, redact bool) interface{} {
	if redact {
		out := make([]interface{}, 0, 3)
		out = append(out, "<ud>")
		out = append(out, in)
		out = append(out, "</ud>")
		return out
	}
	return in
}
