//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
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

func (this *Help) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Input Command : \HELP;
	   Print Help information for all commands. */
	if len(args) == 0 {
		_, werr := OUTPUT.WriteString(HELPMSG)
		if werr != nil {
			return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
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
		if args[0] == "syntax" && len(args) > 1 {
			return COMMAND_LIST["\\syntax"].ExecCommand(args[1:])
		}
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
				return errors.E_SHELL_NO_SUCH_COMMAND, ""
			}
		}

	}
	return 0, ""
}

func (this *Help) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := OUTPUT.WriteString(HHELP)
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
