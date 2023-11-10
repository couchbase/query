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
	"github.com/couchbase/query/value"
)

/* Echo Command */
type Echo struct {
	ShellCommand
}

func (this *Echo) Name() string {
	return "ECHO"
}

func (this *Echo) CommandCompletion() bool {
	return false
}

func (this *Echo) MinArgs() int {
	return ONE_ARG
}

func (this *Echo) MaxArgs() int {
	return MAX_ARGS
}

func (this *Echo) ExecCommand(args []string) (errors.ErrorCode, string) {
	var werr error
	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""

	} else {
		//This is to cascade errors at the end

		// Range over the input arguments to echo.
		for _, val := range args {

			// Resolve each value to return a value.Value.
			v, err_code, err_string := Resolve(val)
			if err_code != 0 {
				//Print each error as you see it.
				s_err := HandleError(err_code, err_string)
				PrintError(s_err)
				continue
			}

			//Do not print the password when printing the credentials
			if val == "-creds" {

				tmpstr := ValToStr(v)
				tmp := usernames(tmpstr)

				fval := strings.Join(tmp, "")
				tmpstr = strings.Replace(fval, "\"", "", -1)

				//Use the string value directly as output.
				_, werr = OUTPUT.WriteString(tmpstr)
				_, werr = OUTPUT.WriteString(" ")

			} else {
				// If the value type is string then output it directly.
				if v.Type() == value.STRING {
					//Use the string value directly as output.
					_, werr = OUTPUT.WriteString(v.Actual().(string))
					_, werr = OUTPUT.WriteString(" ")

				} else {
					// Convert non string values to string and then output.
					_, werr = OUTPUT.WriteString(ValToStr(v))
					_, werr = OUTPUT.WriteString(" ")

				}

			}

		}
	}

	_, werr = OUTPUT.WriteString(NEWLINE)
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}

func (this *Echo) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := OUTPUT.WriteString(HECHO)
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
