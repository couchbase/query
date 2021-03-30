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

/* Help Command */
type Help struct {
	ShellCommand
}

func (this *Help) Name() string {
	return "HELP"
}

func (this *Help) CommandCompletion() bool {
	return false
}

func (this *Help) MinArgs() int {
	return ZERO_ARGS
}

func (this *Help) MaxArgs() int {
	return MAX_ARGS
}

func (this *Help) ExecCommand(args []string) (int, string) {
	/* Input Command : \HELP;
	   Print Help information for all commands. */
	if len(args) == 0 {
		_, werr := io.WriteString(W, HELPMSG)
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		}
		num := 0

		for _, k := range _SORTED_CMD_LIST {
			//Since EXIT and QUIT map to the same message, print it just once.
			if COMMAND_LIST[k].Name() == "EXIT" && num == 0 {
				num = 1
				continue
			}
			err_code, err_str := COMMAND_LIST[k].PrintHelp(false)
			if err_code != 0 {
				return err_code, err_str
			}

		}
	} else {
		/* Input Command : \HELP SET \VERSION;
		   Print help information for input shell commands. The commands
		   need not contain the \ prefix. Return an error if the Command
		   doesnt exist. */
		for _, val := range args {
			val = strings.ToLower(val)
			if strings.HasPrefix(val, "\\") == false {
				val = "\\" + val
			}
			cmd, ok := COMMAND_LIST[val]
			if ok == true {
				err_code, err_str := cmd.PrintHelp(true)
				if err_code != 0 {
					return err_code, err_str
				}
			} else {
				return errors.NO_SUCH_COMMAND, ""
			}
		}

	}
	return 0, ""
}

func (this *Help) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, HHELP)
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
