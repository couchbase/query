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

	"github.com/couchbase/query/errors"
)

/* Copyright Command */
type Copyright struct {
	ShellCommand
}

func (this *Copyright) Name() string {
	return "COPYRIGHT"
}

func (this *Copyright) CommandCompletion() bool {
	return false
}

func (this *Copyright) MinArgs() int {
	return ZERO_ARGS
}

func (this *Copyright) MaxArgs() int {
	return ZERO_ARGS
}

func (this *Copyright) ExecCommand(args []string) (int, string) {
	/* Print the Copyright information for the shell. If the
	   command contains an input argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""
	} else {
		io.WriteString(W, COPYRIGHTMSG)
	}
	return 0, ""
}

func (this *Copyright) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, HCOPYRIGHT)
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
