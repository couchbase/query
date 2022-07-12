//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"fmt"
)

var _rs = map[ErrorCode][2]string{
	E_SS_IDX_NOT_FOUND:      {"idx_not_found", "Index not found"},
	E_SS_NOT_SUPPORTED:      {"not_supported", "%v not supported for scan"},
	E_SS_FAILED:             {"failed", "Scan failed"},
	E_SS_INACTIVE:           {"inactive", "Inactive scan in %v"},
	E_SS_INVALID:            {"invalid", "Invalid scan in %v"},
	E_SS_CONTINUE:           {"continue", "Scan continuation failed"},
	E_SS_CREATE:             {"create", "Scan creation failed"},
	E_SS_CANCEL:             {"cancel", "Scan cancellation failed"},
	E_SS_TIMEOUT:            {"timeout", "Scan exceeded permitted duration"},
	E_SS_CID_GET:            {"cid", "Failed to get collection ID for scan"},
	E_SS_CONN:               {"get_connection", "Failed to get connection for scan"},
	E_SS_FETCH_WAIT_TIMEOUT: {"fetch_wait", "Timed out polling scan for data"},
	E_SS_WORKER_ABORT:       {"worker_abort", "A fatal error occurred in scan processing"},
	E_SS_SPILL:              {"spill", "Operation failed on scan spill file"},
	E_SS_VALIDATE:           {"validate_key", "Failed to validate document key"},
}

func NewSSError(code ErrorCode, args ...interface{}) Error {
	e := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1),
		IKey: "datastore.seq_scan." + _rs[code][0], InternalMsg: _rs[code][1]}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a
		default:
			panic("invalid arguments to NewRSError")
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}
