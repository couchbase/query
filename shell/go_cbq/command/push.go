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
	"encoding/json"
	"io"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
	go_n1ql "github.com/couchbase/go_n1ql"
)

/* Push Command */
type Push struct {
	ShellCommand
}

func (this *Push) Name() string {
	return "PUSH"
}

func (this *Push) CommandCompletion() bool {
	return false
}

func (this *Push) MinArgs() int {
	return 0
}

func (this *Push) MaxArgs() int {
	return MAX_ARGS
}

func (this *Push) ExecCommand(args []string) (int, string) {
	/* Command to set the value of the given parameter to
	   the input value. The top value of the parameter stack
	   is modified. If the command contains no input argument
	   or more than 1 argument then throw an error.
	*/

	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) == 1 {
		return errors.TOO_FEW_ARGS, ""

	} else if len(args) == 0 {
		/* For \PUSH with no input arguments, push the top value
		on the stack for every variable. Dont return errors in
		this case as any one of these stacks can be empty.
		*/

		//Named Parameters
		Pushparam_Helper(NamedParam, true, true)

		//Query Parameters
		Pushparam_Helper(QueryParam, true, false)

		//User Defined Session Variables
		Pushparam_Helper(UserDefSV, false, false)

		//Should not push predefined variables unless
		//they are expicitely specified as an arg to PUSH.
		//Pushparam_Helper(PreDefSV)

	} else {
		//Check what kind of parameter needs to be pushed.
		err_code, err_Str := PushOrSet(args, false)
		if err_code != 0 {
			return err_code, err_Str
		}
	}
	return 0, ""
}

func (this *Push) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\PUSH \n\\PUSH <parameter> <value>\n")
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

/* Push value from the Top of the stack onto the parameter stack.
   This is used by the \PUSH command with no arguments.
*/
func Pushparam_Helper(param map[string]*Stack, isrestp bool, isnamep bool) (int, string) {
	for name, v := range param {
		t, err_code, err_string := v.Top()
		if err_code != 0 {
			return err_code, err_string
		}
		v.Push(t)

		// When passing the query rest api parameter to go_n1ql
		// we need to convert to string only if the value isnt
		// already a string

		if isrestp == true {
			var val string = ""
			if t.Type() == value.STRING {
				val = t.Actual().(string)
			} else {
				val = ValToStr(t)
			}

			if isnamep == true {
				name = "$" + name
			} else {
				//We know it is a query credential
				if name == "creds" {
					// Define credentials as user/pass and convert into
					// JSON object credentials

					var creds Credentials
					creds_ret, err_code, err_str := ToCreds(val)
					if err_code != 0 {
						return err_code, err_str
					}

					for _, v := range creds_ret {
						creds = append(creds, v)
					}

					ac, err := json.Marshal(creds)
					if err != nil {
						return errors.JSON_MARSHAL, ""
					}
					val = string(ac)
				}
			}
			go_n1ql.SetQueryParams(name, val)
		}
	}
	return 0, ""
}
