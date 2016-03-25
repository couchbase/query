//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	SetWriter(writetmp)

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
