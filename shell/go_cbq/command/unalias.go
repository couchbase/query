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

/* Unalias Command */
type Unalias struct {
	ShellCommand
}

func (this *Unalias) Name() string {
	return "UNALIAS"
}

func (this *Unalias) CommandCompletion() bool {
	return false
}

func (this *Unalias) MinArgs() int {
	return 1
}

func (this *Unalias) MaxArgs() int {
	return MAX_ARGS
}

func (this *Unalias) ExecCommand(args []string) (int, string) {

	//Cascade errors for non-existing alias into final error message

	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.TOO_FEW_ARGS, ""

	} else {

		// Range over input aliases amd delete if they exist.
		for _, k := range args {
			_, ok := AliasCommand[k]
			if ok {
				delete(AliasCommand, k)
			} else {
				// Handle and print error as they appear.
				s_err := HandleError(errors.NO_SUCH_ALIAS, " "+k+". ")
				PrintError(s_err)
			}
		}

	}
	return 0, ""
}

func (this *Unalias) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\UNALIAS <alias name>...\n")
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
