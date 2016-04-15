//  Copyright (c) 2015-2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

func (this *Set) ExecCommand(args []string) (int, string) {
	/* Command to set the value of the given parameter to
	   the input value. The top value of the parameter stack
	   is modified. If the command contains no input argument
	   then display all the parameter stacks. If it has 1 input
	   argument then throw error.
	*/

	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""
	} else if len(args) < this.MinArgs() {
		if len(args) == 0 {

			//For \SET with no arguments display the values for
			//all the parameter stacks. This includes the following :
			// Query Parameters
			// Session Paramters  : User Defined
			// Session Parameters : Pre-defined
			// Named Paramters

			//Query Parameters
			var valuestr string = ""
			var werr error
			io.WriteString(W, "Query Parameters :: \n")
			for name, value := range QueryParam {
				//Do not print the password when printing the credentials
				if name == "creds" {
					var users string
					for _, v := range *value {
						users = users + " " + strings.Join(usernames(fmt.Sprintf("%s", v)), "")
					}

					valuestr = fmt.Sprintln("Parameter name :", name, "Value [", users, "]")
					_, werr = io.WriteString(W, valuestr)

				} else {
					valuestr = fmt.Sprintln("Parameter name :", name, "Value ", *value)
					_, werr = io.WriteString(W, valuestr)
				}
			}
			_, werr = io.WriteString(W, "\n")

			//Named Parameters
			valuestr = ""
			io.WriteString(W, "Named Parameters :: \n")
			for name, value := range NamedParam {
				valuestr = fmt.Sprintln("Parameter name :", name, "Value ", *value)
				io.WriteString(W, valuestr)
			}
			_, werr = io.WriteString(W, "\n")

			//User Defined Session Parameters
			valuestr = ""
			io.WriteString(W, "User Defined Session Parameters :: \n")
			for name, value := range UserDefSV {
				valuestr = fmt.Sprintln("Parameter name :", name, "Value ", *value)
				io.WriteString(W, valuestr)
			}
			_, werr = io.WriteString(W, "\n")

			//Predefined Session Parameters
			valuestr = ""
			io.WriteString(W, "Predefined Session Parameters :: \n")
			for name, value := range PreDefSV {
				valuestr = fmt.Sprintln("Parameter name :", name, "Value ", *value)
				io.WriteString(W, valuestr)
			}
			_, werr = io.WriteString(W, "\n")

			if werr != nil {
				return errors.WRITER_OUTPUT, werr.Error()
			}

		} else {
			return errors.TOO_FEW_ARGS, ""
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

func (this *Set) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\SET [ parameter value ]\n")
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
