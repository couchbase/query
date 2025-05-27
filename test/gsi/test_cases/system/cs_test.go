//  Copyright 2013-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package system

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestSystem(t *testing.T) {
	if strings.ToLower(os.Getenv("GSI_TEST")) != "true" {
		return
	}

	qc := start_cs()

	runStmt(qc, "delete from system:completed_requests")
	runMatch("case_system_completed.json", false, false, qc, t)

	time.Sleep(2 * time.Second)

	runMatch("case_system_my_user_info.json", false, false, qc, t)
	runMatch("case_system_prepareds.json", false, false, qc, t)
	runMatch("case_system_user_info.json", false, false, qc, t)

}
