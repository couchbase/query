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
	"fmt"
	"io"
	"strings"

	"github.com/couchbase/query/errors"
)

/* Alias Command */
type Alias struct {
	ShellCommand
}

func (this *Alias) Name() string {
	return "ALIAS"
}

func (this *Alias) CommandCompletion() bool {
	return false
}

func (this *Alias) MinArgs() int {
	return TWO_ARGS
}

func (this *Alias) MaxArgs() int {
	return MAX_ARGS
}

func (this *Alias) ExecCommand(args []string) (int, string) {

	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {

		if len(args) == 0 {
			// \ALIAS without input args lists the aliases present.
			if len(AliasCommand) == 0 {
				return errors.NO_SUCH_ALIAS, ""
			}

			for k, v := range AliasCommand {

				tmp := fmt.Sprintf("%-14s %-14s\n", k, v)
				_, werr := io.WriteString(W, tmp)
				if werr != nil {
					return errors.WRITER_OUTPUT, werr.Error()
				}
			}

		} else {
			// Error out if it has 1 argument.
			return errors.TOO_FEW_ARGS, ""
		}

	} else {
		// Concatenate the elements of args with separator " "
		// to give the input value
		value := strings.Join(args[1:], " ")

		//Add this to the map for Aliases
		key := args[0]

		//Aliases can be replaced.
		if key != "" {
			AliasCommand[key] = value
		}

	}
	return 0, ""

}

func (this *Alias) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, HALIAS)
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
