//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func (this *Alias) ExecCommand(args []string) (errors.ErrorCode, string) {

	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {

		if len(args) == 0 {
			// \ALIAS without input args lists the aliases present.
			if len(AliasCommand) == 0 {
				return errors.E_SHELL_NO_SUCH_ALIAS, ""
			}

			for k, v := range AliasCommand {

				tmp := fmt.Sprintf("%-14s %-14s"+NEWLINE, k, v)
				_, werr := io.WriteString(W, tmp)
				if werr != nil {
					return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
				}
			}

		} else {
			// Error out if it has 1 argument.
			return errors.E_SHELL_TOO_FEW_ARGS, ""
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

func (this *Alias) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HALIAS)
	if desc {
		err_code, err_str := printDesc(this.Name())
		if err_code != 0 {
			return err_code, err_str
		}
	}
	_, werr = io.WriteString(W, NEWLINE)
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}
