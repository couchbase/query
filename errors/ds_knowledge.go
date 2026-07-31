//  Copyright 2026-Present Couchbase, Inc.
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

var _knowledge = map[ErrorCode][2]string{
	E_KNOWLEDGE_CREATE:         {"create", "Create failed for knowledge '%v'"},
	E_KNOWLEDGE_DROP:           {"drop", "Drop failed for knowledge '%v'"},
	E_KNOWLEDGE_NOT_FOUND:      {"not_found", "Knowledge '%v' not found"},
	E_KNOWLEDGE:                {"error", "Error accessing knowledge"},
	E_KNOWLEDGE_ALREADY_EXISTS: {"duplicate", "Knowledge '%v' already exists"},
	E_KNOWLEDGE_INVALID_DATA:   {"invalid_data", "Invalid knowledge data: %v"},
	E_KNOWLEDGE_INVALID_PATH:   {"invalid_path", "Invalid knowledge keyspace path '%v'"},
	E_KNOWLEDGE_LIMIT_EXCEEDED: {"limit_exceeded", "%v"},
}

func NewKnowledgeError(code ErrorCode, args ...interface{}) Error {
	e := &err{level: EXCEPTION, ICode: code, InternalCaller: CallerN(1),
		IKey: "datastore.knowledge." + _knowledge[code][0], InternalMsg: _knowledge[code][1]}
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
			panic(fmt.Sprintf("invalid argument (%T) to NewKnowledgeError", a))
		}
	}
	if len(fmtArgs) > 0 {
		e.InternalMsg = fmt.Sprintf(e.InternalMsg, fmtArgs...)
	}
	return e
}
