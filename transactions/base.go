//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package transactions

func IsValidStatement(txId, stmtType string, tximplicit, allow bool) (bool, string) {
	switch stmtType {
	case "SELECT", "UPDATE", "INSERT", "UPSERT", "DELETE", "MERGE":
		return true, ""
	case "EXECUTE", "PREPARE":
		return true, ""
	case "EXECUTE_FUNCTION", "EXPLAIN_FUNCTION":
		return true, ""
	case "COMMIT", "ROLLBACK", "ROLLBACK_SAVEPOINT", "SET_TRANSACTION_ISOLATION", "SAVEPOINT":
		return allow || txId != "", "outside the"
	case "START_TRANSACTION":
		return allow || txId == "", "within the"
	default:
		if txId != "" {
			return false, "within the"
		}
		return !tximplicit, "in implicit"
	}
}
