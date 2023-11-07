//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	return ONE_ARG
}

func (this *Unalias) MaxArgs() int {
	return MAX_ARGS
}

func (this *Unalias) ExecCommand(args []string) (errors.ErrorCode, string) {

	//Cascade errors for non-existing alias into final error message

	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""

	} else {

		// Range over input aliases amd delete if they exist.
		for _, k := range args {
			_, ok := AliasCommand[k]
			if ok {
				delete(AliasCommand, k)
			} else {
				// Handle and print error as they appear.
				s_err := HandleError(errors.E_SHELL_NO_SUCH_ALIAS, " "+k+". ")
				PrintError(s_err)
			}
		}

	}
	return 0, ""
}

func (this *Unalias) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HUNALIAS)
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
