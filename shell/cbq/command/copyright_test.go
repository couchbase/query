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

	"github.com/couchbase/query/errors"
)

/*
   Test the \COPYRIGHT command.
*/

func TestCopyright(t *testing.T) {
	copyright := COMMAND_LIST["\\copyright"]
	args := make([]string, 2)

	errCode, errStr := copyright.ExecCommand(args)

	if errCode == errors.TOO_MANY_ARGS {
		t.Logf("Correctly evaluated error: Too manay args.")
	} else {
		t.Error("The max number of args for \\Copyright is 0.")
	}

	var b bytes.Buffer
	args = make([]string, 0)
	writetmp := bufio.NewWriter(&b)
	SetWriter(writetmp)

	errCode, errStr = copyright.ExecCommand(args)
	writetmp.Flush()

	if errCode != 0 {
		t.Errorf("Error :: %s", HandleError(errCode, errStr))
	} else {
		t.Logf("%s", b.String())
	}
}
