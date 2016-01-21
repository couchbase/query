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
	SetWriter(writetmp)

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
