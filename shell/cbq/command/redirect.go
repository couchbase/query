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
	"os"
	"strings"
	"time"

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
	return TWO_ARGS
}

func (this *Redirect) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to load a file into the shell.
	 */
	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""
	} else {
		if len(args) == 1 && strings.ToLower(args[0]) == "off" {
			SetOutput(os.Stdout, false)
		} else {
			i := 0
			if len(args) == 2 {
				if strings.ToLower(args[i]) == "tee" {
					i++
				} else {
					return errors.E_SHELL_INVALID_ARGUMENT, ""
				}
			}

			var w io.Writer
			var err error
			if strings.HasPrefix(args[i], "+") {
				if len(args[i]) < 2 {
					return errors.E_SHELL_INVALID_ARGUMENT, ""
				}
				w, err = os.OpenFile(args[i][1:], os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
				if err == nil {
					_, err = w.Write([]byte("-- " + time.Now().Format("2006-01-02T15:04:05.999Z07:00") + ": opened for appending\n"))
					if err != nil {
						w.(*os.File).Close()
						w = nil
					}
				}
			} else {
				if len(args[i]) < 1 {
					return errors.E_SHELL_INVALID_ARGUMENT, ""
				}
				w, err = os.OpenFile(args[i], os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			}
			if err != nil {
				return errors.E_SHELL_WRITER_OUTPUT, err.Error()
			}

			SetOutput(w, true)
			if i != 0 {
				AddOutput(os.Stdout, false)
			}
		}
	}
	return 0, ""
}

func (this *Redirect) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := OUTPUT.WriteString(HREDIRECT)
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
