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

/* Exit and Quit Commands */
type Exit struct {
	ShellCommand
}

func (this *Exit) Name() string {
	return "EXIT"
}

func (this *Exit) CommandCompletion() bool {
	return false
}

func (this *Exit) MinArgs() int {
	return ZERO_ARGS
}

func (this *Exit) MaxArgs() int {
	return ZERO_ARGS
}

func (this *Exit) ExecCommand(args []string) (int, string) {
	/* Command to Exit the shell. We set the EXIT flag to true.
	Once this command is processed, and executequery returns to
	HandleInteractiveMode, handle errors (if any) and then exit
	with the correct exit status. If the command contains an
	input argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""
	} else {
		_, werr := io.WriteString(W, "Exiting the shell.\n")
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		}
		EXIT = true
	}
	return 0, ""
}

func (this *Exit) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\EXIT \n\\QUIT\n")
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
