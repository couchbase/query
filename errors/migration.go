//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

// migration errors

func NewMigrationError(what, msg string, elems []string, e error) Error {
	var c map[string]interface{}
	errString := "Error occurred during " + what + " migration"
	if len(elems) == 4 {
		// elems, if present, should only have 4 parts (fully qualified name)
		c = make(map[string]interface{}, 4)
		c["namespace"] = elems[0]
		c["bucket"] = elems[1]
		c["scope"] = elems[2]
		if what == "UDF" {
			c["function"] = elems[4]
		} else if what == "CBO_STATS" {
			c["collection"] = elems[4]
		}
		errString += " (" + elems[0] + ":" + elems[1] + "." + elems[2] + "." + elems[3] + ")"
	}
	errString += ": " + msg
	return &err{level: EXCEPTION, ICode: E_MIGRATION, IKey: "migration_error", cause: c,
		ICause: e, InternalMsg: errString, InternalCaller: CallerN(1)}
}

func NewMigrationInternalError(what, msg string, elems []string, e error) Error {
	var c map[string]interface{}
	errString := "Unexpected error occurred during " + what + " migration"
	if len(elems) == 4 {
		// elems, if present, should only have 4 parts (fully qualified name)
		c = make(map[string]interface{}, 4)
		c["namespace"] = elems[0]
		c["bucket"] = elems[1]
		c["scope"] = elems[2]
		if what == "UDF" {
			c["function"] = elems[4]
		} else if what == "CBO_STATS" {
			c["collection"] = elems[4]
		}
		errString += " (" + elems[0] + ":" + elems[1] + "." + elems[2] + "." + elems[3] + ")"
	}
	errString += ": " + msg
	return &err{level: EXCEPTION, ICode: E_MIGRATION_INTERNAL, IKey: "migration_internal_error", cause: c,
		ICause: e, InternalMsg: errString, InternalCaller: CallerN(1)}
}
