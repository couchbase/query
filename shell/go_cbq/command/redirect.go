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
	"strings"

	"github.com/couchbase/query/errors"
)

/* Redirect Command */
type Redirect struct {
	ShellCommand
}

func (this *Redirect) Name() string {
	return "REDIRECT"
}

func (this *Redirect) CommandCompletion() bool {
	return false
}

func (this *Redirect) MinArgs() int {
	return ONE_ARG
}

func (this *Redirect) MaxArgs() int {
	return ONE_ARG
}

func (this *Redirect) ExecCommand(args []string) (int, string) {
	/* Command to load a file into the shell.
	 */
	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.TOO_FEW_ARGS, ""
	} else {
		if strings.ToLower(args[0]) == "off" {
			FILE_RW_MODE = false
		} else {
			FILE_RW_MODE = true
			FILE_OUTPUT = args[0]
		}
	}
	return 0, ""
}

func (this *Redirect) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\REDIRECT <filename>\n\\REDIRECT OFF;")
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
