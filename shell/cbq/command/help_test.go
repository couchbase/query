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
	"strings"
	"testing"
)

/*
   Test test \HELP, and printDesc Method in common.go
   Test it for 1 sample case.
*/

func TestHelp(t *testing.T) {
	//Test the \HELP command and the printDesc method from
	//common.go

	help := COMMAND_LIST["\\help"]
	args := make([]string, 1)
	//Case 1 : Error case when command does not exist.

	var b bytes.Buffer
	writetmp := bufio.NewWriter(&b)
	SetOutput(writetmp, true)

	args[0] = "\\dummy"
	errCode, errStr := help.ExecCommand(args)
	writetmp.Flush()

	if errCode != 0 {
		t.Log(HandleError(errCode, errStr))
	} else {
		t.Error("Unknown command error expected.")
	}

	//Case 2 : Display help for ECHO.
	args[0] = "\\ECHO"
	errCode, errStr = help.ExecCommand(args)
	writetmp.Flush()

	if errCode == 0 {
		t.Log("Echo command help.")
	} else {
		t.Error(HandleError(errCode, errStr))
	}

	//Case 3 : Test display for all errors. \HELP command with no input args.
	b.Reset()
	errCode, errStr = help.ExecCommand([]string{})
	writetmp.Flush()

	if errCode == 0 {
		t.Log("List of commands displayed is : ", strings.Replace(b.String(), "\n", " ", -1))
	} else {
		t.Error(HandleError(errCode, errStr))
	}
}
