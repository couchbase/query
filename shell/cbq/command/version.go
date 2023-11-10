//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"runtime"

	"github.com/couchbase/query/errors"
)

/* Version Command */
type Version struct {
	ShellCommand
}

func (this *Version) Name() string {
	return "VERSION"
}

func (this *Version) CommandCompletion() bool {
	return false
}

func (this *Version) MinArgs() int {
	return ZERO_ARGS
}

func (this *Version) MaxArgs() int {
	return ZERO_ARGS
}

func (this *Version) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Print the shell version. If the command contains an input
	   argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.E_SHELL_TOO_MANY_ARGS, ""
	} else {
		_, werr := OUTPUT.WriteString(NewMessage(GOVERSIONMSG, runtime.Version()) + NEWLINE)
		if werr == nil {
			_, werr = OUTPUT.WriteString(NewMessage(VERSIONMSG, SHELL_VERSION) + NEWLINE)
		}
		if werr == nil {
			_, werr = OUTPUT.WriteString(SERVERVERSIONMSG)
		}
		if werr != nil {
			return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
		}
	}
	return 0, ""
}

func (this *Version) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := OUTPUT.WriteString(HVERSION)
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
