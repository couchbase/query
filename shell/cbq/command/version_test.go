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

	"github.com/couchbase/query/errors"
)

/*
   Test the \VERSION command.
*/

func TestVersion(t *testing.T) {

	version := COMMAND_LIST["\\version"]
	args := make([]string, 2)

	errCode, errStr := version.ExecCommand(args)

	if errCode == errors.E_SHELL_TOO_MANY_ARGS {
		t.Logf("Correctly evaluated error: Too manay args.")
	} else {
		t.Error("The max number of args for \\Version is 0.")
	}

	var b bytes.Buffer
	args = make([]string, 0)
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	errCode, errStr = version.ExecCommand(args)
	writetmp.Flush()

	if errCode != 0 {
		t.Errorf("Error :: %s", HandleError(errCode, errStr))
	} else {
		t.Logf("%s", b.String())
	}

}
