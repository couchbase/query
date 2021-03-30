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

/* Source Command */
type Source struct {
	ShellCommand
}

func (this *Source) Name() string {
	return "SOURCE"
}

func (this *Source) CommandCompletion() bool {
	return false
}

func (this *Source) MinArgs() int {
	return ONE_ARG
}

func (this *Source) MaxArgs() int {
	return ONE_ARG
}

func (this *Source) ExecCommand(args []string) (int, string) {
	/* Command to load a file into the shell.
	 */
	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.TOO_FEW_ARGS, ""
	} else {
		/* This case needs to be handled in the ShellCommand
		   in the main package, since we need to run each
		   query as it is being read. Otherwise, if we load it
		   into a buffer, we restrict the number of queries that
		   can be loaded from the file.
		*/
		FILE_RD_MODE = true
		FILE_INPUT = args[0]
	}
	return 0, ""
}

func (this *Source) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, HSOURCE)
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
