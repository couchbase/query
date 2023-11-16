//  Copyright 2023-Present Couchbase, Inc.
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

var _seq = map[ErrorCode][2]string{
	E_SEQUENCE_NOT_ENABLED:    {"not_enabled", "Sequence support is not enabled for '%v'"},
	E_SEQUENCE_CREATE:         {"create", "Create failed for sequence '%v'"},
	E_SEQUENCE_ALTER:          {"alter", "Alter failed for sequence '%v'"},
	E_SEQUENCE_DROP:           {"drop", "Drop failed for sequence '%v'"},
	E_SEQUENCE_DROP_ALL:       {"drop_all", "Drop failed for sequences '%v'"},
	E_SEQUENCE_INVALID_CACHE:  {"cache", "Invalid cache value %v"},
	E_SEQUENCE_INVALID_RANGE:  {"range", "Invalid range %v"},
	E_SEQUENCE_NOT_FOUND:      {"not_found", "Sequence '%v' not found"},
	E_SEQUENCE:                {"error", "Error accessing sequence"},
	E_SEQUENCE_ALREADY_EXISTS: {"duplicate", "Sequence '%v' already exists"},
	E_SEQUENCE_METAKV:         {"metakv", "Error accessing sequences cache monitor data"},
	E_SEQUENCE_INVALID_DATA:   {"invalid_data", "Invalid sequence data"},
	E_SEQUENCE_EXHAUSTED:      {"exhausted", "Sequence '%v' has reached its limit"},
	E_SEQUENCE_CYCLE:          {"cycle", "Cycle failed for sequence '%v'"},
	E_SEQUENCE_INVALID_NAME:   {"invalid_name", "Invalid sequence name '%v'"},
	E_SEQUENCE_READ_ONLY_REQ:  {"read_only", "Sequences cannot be used in read-only requests"},
	W_SEQUENCE_CACHE_SIZE:     {"warn_cache_size", "Cache size (%v) below recommended minimum"},
	E_SEQUENCE_NAME_PARTS:     {"name_parts", "Sequence name resolves to '%v' - check query_context?%v"},
	W_SEQUENCE_NO_PREV_VALUE:  {"no_prev_value", "Sequence previous value cannot be accessed before next value generation."},
}

// non-exception levels
var _seqLevel = map[ErrorCode]int{
	W_SEQUENCE_CACHE_SIZE:    WARNING,
	W_SEQUENCE_NO_PREV_VALUE: WARNING,
}

func NewSequenceError(code ErrorCode, args ...interface{}) Error {
	l, ok := _seqLevel[code]
	if !ok {
		l = EXCEPTION
	}
	e := &err{level: l, ICode: code, InternalCaller: CallerN(1),
		IKey: "datastore.sequence." + _seq[code][0], InternalMsg: _seq[code][1]}
	var fmtArgs []interface{}
	for _, a := range args {
		switch a := a.(type) {
		case string:
			fmtArgs = append(fmtArgs, a)
		case Error:
			e.cause = a
		case error:
			e.cause = a
		case nil:
			// ignore
		default:
			panic(fmt.Sprintf("invalid argument (%T) to NewSequenceError", a))
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}
