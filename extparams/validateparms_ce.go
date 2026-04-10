//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package extparams

func ValidateCatalog(params map[string]any) map[string]*ExternalParamsError {
	rv := make(map[string]*ExternalParamsError, len(params))
	for k, v := range params {
		rv[k] = &ExternalParamsError{"CE unsupported", "Parameter not supported."}
	}
	return rv
}

func ValidateCollection(params map[string]any) map[string]*ExternalParamsError {
	rv := make(map[string]*ExternalParamsError, len(params))
	for k, v := range params {
		rv[k] = &ExternalParamsError{"CE unsupported", "Parameter not supported."}
	}
	return rv
}
