//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
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

func (this *Exit) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to Exit the shell. We set the EXIT flag to true.
	Once this command is processed, and executequery returns to
	HandleInteractiveMode, handle errors (if any) and then exit
	with the correct exit status. If the command contains an
	input argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.E_SHELL_TOO_MANY_ARGS, ""
	} else {
		EXIT = true
	}
	return 0, ""
}

func (this *Exit) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := OUTPUT.WriteString(HEXIT)
	if desc {
		err_code, err_str := printDesc(this.Name())
		if err_code != 0 {
			return err_code, err_str
		}
	}
	_, werr = OUTPUT.WriteString(NEWLINE)
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}
