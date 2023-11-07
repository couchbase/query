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
	return ZERO_ARGS
}

func (this *Disconnect) MaxArgs() int {
	return ZERO_ARGS
}

func (this *Disconnect) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to disconnect service. Use the noQueryService
	   flag value and the disconnect flag value to determine
	   disconnection. If the command contains an input argument
	   then throw an error.
	*/
	if len(args) != 0 {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else {
		DISCONNECT = true
		io.WriteString(W, NOCONNMSG)
	}
	return 0, ""
}

func (this *Disconnect) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HDISCONNECT)
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
