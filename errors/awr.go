//  Copyright 2024-Present Couchbase, Inc.
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

var _awr = map[ErrorCode][2]string{
	E_AWR_START:   {"start", "Failed to start workload reporting"},
	E_AWR_SETTING: {"setting", "Invalid value '%s' for setting '%s'"},
	E_AWR_CONFIG:  {"config", "Error processing configuration"},
}

func NewAWRError(code ErrorCode, args ...interface{}) Error {
	e := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1),
		IKey: "service.workload_report." + _awr[code][0], InternalMsg: _awr[code][1]}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a.Error()
		default:
			panic("invalid arguments to NewAWRError")
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}