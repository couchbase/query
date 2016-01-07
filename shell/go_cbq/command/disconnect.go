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
	"io"

	"github.com/couchbase/query/errors"
)

/* Disconnect Command */
type Disconnect struct {
	ShellCommand
}

func (this *Disconnect) Name() string {
	return "DISCONNECT"
}

func (this *Disconnect) CommandCompletion() bool {
	return false
}

func (this *Disconnect) MinArgs() int {
	return 0
}

func (this *Disconnect) MaxArgs() int {
	return 0
}

func (this *Disconnect) ExecCommand(args []string) (int, string) {
	/* Command to disconnect service. Use the NoQueryService
	   flag value and the disconnect flag value to determine
	   disconnection. If the command contains an input argument
	   then throw an error.
	*/
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""

	} else {
		DISCONNECT = true
		io.WriteString(W, "\nCouchbase query shell not connected to any endpoint. Use \\CONNECT command to connect.\n")
	}
	return 0, ""
}

func (this *Disconnect) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\DISCONNECT\n")
	if desc {
		err_code, err_str := printDesc(this.Name())
		if err_code != 0 {
			return err_code, err_str
		}
	}
	_, werr = io.WriteString(W, "\n")
	if werr != nil {
		return errors.WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}
