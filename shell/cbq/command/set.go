//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"fmt"
	"io"
	"strings"

	"github.com/couchbase/query/errors"
)

/* Set Command */
type Set struct {
	ShellCommand
}

func (this *Set) Name() string {
	return "SET"
}

func (this *Set) CommandCompletion() bool {
	return false
}

func (this *Set) MinArgs() int {
	return TWO_ARGS
}

func (this *Set) MaxArgs() int {
	return MAX_ARGS
}

func (this *Set) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to set the value of the given parameter to
	   the input value. The top value of the parameter stack
	   is modified. If the command contains no input argument
	   then display all the parameter stacks. If it has 1 input
	   argument then throw error.
	*/

	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""
	} else if len(args) < this.MinArgs() {
		if len(args) == 0 {

			//For \SET with no arguments display the values for
			//all the parameter stacks. This includes the following :
			// Query Parameters
			// Session Paramters  : User Defined
			// Session Parameters : Pre-defined
			// Named Paramters

			//Query Parameters
			var werr error
			_, werr = io.WriteString(W, QUERYP)
			for name, value := range QueryParam {
				//Do not print the password when printing the credentials
				if name == "creds" {
					var users string
					for _, v := range *value {
						users = users + " " + strings.Join(usernames(fmt.Sprintf("%s", v)), "")
					}
					werr = printSET(name, "["+users+"]")

				} else {
					werr = printSET(name, fmt.Sprintf("%v", *value))
				}
			}
			_, werr = io.WriteString(W, "\n")

			//Named Parameters
			_, werr = io.WriteString(W, NAMEDP)
			for name, value := range NamedParam {
				if len(name) > 2 && name[0] == '_' && name[len(name)-1] == '_' {
					vals := make([]string, len(*value))
					for i := range vals {
						vals[i] = "***"
					}
					werr = printSET(name, fmt.Sprintf("%v", vals))
				} else {
					werr = printSET(name, fmt.Sprintf("%v", *value))
				}
			}
			_, werr = io.WriteString(W, "\n")

			//User Defined Session Parameters
			_, werr = io.WriteString(W, USERDEFP)
			for name, value := range UserDefSV {
				werr = printSET(name, fmt.Sprintf("%v", *value))
			}
			_, werr = io.WriteString(W, "\n")

			//Predefined Session Parameters
			_, werr = io.WriteString(W, PREDEFP)
			for name, value := range PreDefSV {
				werr = printSET(name, fmt.Sprintf("%v", *value))
			}
			_, werr = io.WriteString(W, "\n")

			if werr != nil {
				return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
			}

		} else {
			return errors.E_SHELL_TOO_FEW_ARGS, ""
		}

	} else {
		//Check what kind of parameter needs to be set.
		err_code, err_str := PushOrSet(args, true)
		if err_code != 0 {
			return err_code, err_str
		}
	}
	return 0, ""
}

func (this *Set) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HSET)
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

func printSET(name, value string) (werr error) {
	valuestr := NewMessage(PNAME, name) + "\n" + NewMessage(PVAL, value)
	_, werr = io.WriteString(W, valuestr)
	_, werr = io.WriteString(W, "\n\n")
	return
}

func usernames(arrcreds string) []string {

	//arrcreds = strings.Replace(arrcreds, "[", "", -1)
	//arrcreds = strings.Replace(arrcreds, "]", "", -1)

	arrcreds = strings.Replace(arrcreds, "\"", "", -1)
	users := strings.Split(arrcreds, ",")

	var unames []string
	unames = append(unames, "\"")
	next := false
	for _, v := range users {
		if next == true {
			unames = append(unames, ",")
		}
		tmp := strings.Split(v, ":")
		unames = append(unames, tmp[0]+":*")
		next = true
	}
	unames = append(unames, "\" ")

	return unames
}
