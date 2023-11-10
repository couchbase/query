//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"bufio"
	"bytes"
	"testing"
)

/*
   Test the common methods
*/

// The Resolve tests test the Resolve, PushValue_Helper
// and PopValue_Helper methods

func TestEcho(t *testing.T) {
	echo := COMMAND_LIST["\\echo"]
	args := make([]string, 0)

	errCode, errStr := echo.ExecCommand(args)
	if errCode != 0 {
		t.Log("Error Case :", HandleError(errCode, errStr))
	} else {
		t.Error("Min args for \\Echo command has changed.")
	}

	errCode, errStr = PushValue_Helper(false, QueryParam, "creds", "Administrator:password")
	if errCode != 0 {
		t.Errorf("%s", HandleError(errCode, errStr))
	} else {
		t.Log("Credentials have been set")
	}

	args = make([]string, 5)
	args[0] = "Test1"
	args[1] = "select * from `beer-sample`"
	args[2] = "histfile"
	args[3] = "-creds"
	args[4] = "-$rate"

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr = echo.ExecCommand(args)
	writetmp.Flush()
	if errCode == 0 {
		t.Log("Expected error for -$rate.")
		t.Log(b.String())
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	errCode, errStr = PopValue_Helper(true, QueryParam, "creds")
	if errCode != 0 {
		t.Errorf("Error unsetting parameter : %s", HandleError(errCode, errStr))
	} else {
		t.Log("Credentials have been deleted")
	}
}
