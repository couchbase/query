//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

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

func (this *Redirect) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to load a file into the shell.
	 */
	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""
	} else {
		if strings.ToLower(args[0]) == "off" {
			FILE_RW_MODE = false
		} else {
			FILE_RW_MODE = true
			FILE_OUTPUT = args[0]
			if strings.HasPrefix(FILE_OUTPUT, "+") {
				FILE_APPEND_MODE = true
				FILE_OUTPUT = strings.TrimPrefix(FILE_OUTPUT, "+")
			} else {
				FILE_APPEND_MODE = false
			}
		}
	}
	return 0, ""
}

func (this *Redirect) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HREDIRECT)
	if desc {
		err_code, err_str := printDesc(this.Name())
		if err_code != 0 {
			return err_code, err_str
		}
	}
	_, werr = io.WriteString(W, "\n")
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}
