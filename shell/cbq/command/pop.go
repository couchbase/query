//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/errors"
)

/* Pop Command */
type Pop struct {
	ShellCommand
}

func (this *Pop) Name() string {
	return "POP"
}

func (this *Pop) CommandCompletion() bool {
	return false
}

func (this *Pop) MinArgs() int {
	return ZERO_ARGS
}

func (this *Pop) MaxArgs() int {
	return ONE_ARG
}

func (this *Pop) ExecCommand(args []string) (errors.ErrorCode, string) {

	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""

	} else if len(args) == 0 {
		/* For \Pop with no input arguments, Pop the top value
		on the stack for every variable. Dont return errors in
		this case as any one of these stacks can be empty.
		*/

		//Named Parameters
		Popparam_Helper(NamedParam, true, true)

		//Query Parameters
		Popparam_Helper(QueryParam, true, false)

		//User Defined Session Variables
		Popparam_Helper(UserDefSV, false, false)

		//Should not pop predefined variables unless
		//they are expicitely specified as an arg to POP.
		//Popparam_Helper(PreDefSV, false)

	} else {
		//Check what kind of parameter needs to be popped

		if strings.HasPrefix(args[0], "-$") || strings.HasPrefix(args[0], "-@") {
			// For Named Parameters
			vble := args[0]
			vble = vble[2:]

			err_code, err_string := PopValue_Helper(false, NamedParam, vble)
			if err_code != 0 {
				return err_code, err_string
			}

			st_val, ok := NamedParam[vble]

			if ok {
				if NamedParam[vble].Len() == 0 {
					name := "$" + vble
					n1ql.UnsetQueryParams(name)
				} else {
					name := "$" + vble
					err_code, err_str := setNewParamPop(name, st_val)
					if err_code != 0 {
						return err_code, err_str
					}
				}

			} else {
				name := "$" + vble
				n1ql.UnsetQueryParams(name)
			}

		} else if strings.HasPrefix(args[0], "-") {
			// For query parameters
			vble := args[0]
			vble = vble[1:]

			err_code, err_string := PopValue_Helper(false, QueryParam, vble)
			if err_code != 0 {
				return err_code, err_string
			}

			st_val, ok := QueryParam[vble]

			if ok {
				if QueryParam[vble].Len() == 0 {
					n1ql.UnsetQueryParams(vble)
				} else {
					err_code, err_str := setNewParamPop(vble, st_val)
					if err_code != 0 {
						return err_code, err_str
					}
				}

			} else {
				n1ql.UnsetQueryParams(vble)
			}

		} else if strings.HasPrefix(args[0], "$") {
			// For User defined session variables
			vble := args[0]
			vble = vble[1:]

			err_code, err_string := PopValue_Helper(false, UserDefSV, vble)
			if err_code != 0 {
				return err_code, err_string
			}

		} else {
			// For Predefined session variables
			vble := args[0]

			err_code, err_string := PopValue_Helper(false, PreDefSV, vble)
			if err_code != 0 {
				return err_code, err_string
			}
			var nval string

			if vble == "batch" {
				st_val, ok := PreDefSV["batch"]
				if ok {
					newval, err_code, err_str := st_val.Top()
					if err_code != 0 {
						return err_code, err_str
					}
					nval = ValToStr(newval)
					nval = handleStrings(nval)
				} else {
					err_code, err_str := PushValue_Helper(false, PreDefSV, "batch", "off")
					if err_code != 0 {
						return err_code, err_str

					}
					nval = "off"
				}
				BATCH = nval

			} else if vble == "histfile" {
				//Predefined variables are only allowed to be specifically
				//popped
				st_val, ok := PreDefSV["histfile"]
				if ok {
					newval, err_code, err_str := st_val.Top()
					if err_code != 0 {
						return err_code, err_str
					}
					nval = ValToStr(newval)
					nval = handleStrings(nval)
				} else {
					err_code, err_str := PushValue_Helper(false, PreDefSV, "histfile", "\".cbq_history\"")
					if err_code != 0 {
						return err_code, err_str

					}
					nval = ".cbq_history"
				}

				HISTFILE = nval

				//Print the path to histfile
				err_code, err_str := printPath(HISTFILE)
				if err_code != 0 {
					return err_code, err_str
				}
			} else if vble == "quiet" {
				st_val, ok := PreDefSV["quiet"]
				if ok {
					newval, err_code, err_str := st_val.Top()
					if err_code != 0 {
						return err_code, err_str
					}
					nval = ValToStr(newval)
					nval = handleStrings(nval)
				} else {
					err_code, err_str := PushValue_Helper(false, PreDefSV, "quiet", strconv.FormatBool(false))
					if err_code != 0 {
						return err_code, err_str

					}
					nval = strconv.FormatBool(false)
				}

				// Set QUIET boolean value from nval string.
				// Dont need to worry about error handling here as we make sure we push correct values
				// into the stack.
				QUIET, _ = strconv.ParseBool(nval)
			} else if vble == "terse" {
				st_val, ok := PreDefSV["terse"]
				if ok {
					newval, err_code, err_str := st_val.Top()
					if err_code != 0 {
						return err_code, err_str
					}
					nval = ValToStr(newval)
					nval = handleStrings(nval)
				} else {
					err_code, err_str := PushValue_Helper(false, PreDefSV, "terse", strconv.FormatBool(false))
					if err_code != 0 {
						return err_code, err_str

					}
					nval = strconv.FormatBool(false)
				}
				TERSE, _ = strconv.ParseBool(nval)
			}

		}
	}
	return 0, ""
}

func (this *Pop) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HPOP)
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

/* Pop the top value of the parameter stack.
   This is used by the \POP command with no arguments.
*/
func Popparam_Helper(param map[string]*Stack, isrestp bool, isnamep bool) (errors.ErrorCode, string) {
	for name, val := range param {
		_, err_code, err_str := val.Pop()

		if isrestp == true && val.Len() == 0 {
			delete(param, name)
			if isnamep == true {
				name = "$" + name
			}
			n1ql.UnsetQueryParams(name)
		}

		if err_code != 0 {
			return err_code, err_str
		}

		if isrestp == true && val.Len() != 0 {
			if isnamep == true {
				name = "$" + name
			}
			err_code, err_str = setNewParamPop(name, val)
			if err_code != 0 {
				return err_code, err_str
			}
		}
	}
	return 0, ""
}

func setNewParamPop(name string, paramst *Stack) (errors.ErrorCode, string) {
	newval, err_code, err_str := paramst.Top()
	if err_code != 0 {
		return err_code, err_str
	}

	//Convert top of stack to string, to represent the new value.
	nval := ValToStr(newval)

	if name == "creds" {
		// Define credentials as user/pass and convert into
		// JSON object credentials

		var creds Credentials
		creds_ret, err_code, err_str := ToCreds(nval)
		if err_code != 0 {
			return err_code, err_str
		}

		for _, v := range creds_ret {
			creds = append(creds, v)
		}

		ac, err := json.Marshal(creds)
		if err != nil {
			return errors.E_SHELL_JSON_MARSHAL, ""
		}
		nval = string(ac)
	}
	n1ql.SetQueryParams(name, nval)
	return 0, ""
}

func handleStrings(nval string) string {
	//In order to store it in a format that can be read,
	//we escaped the strings. Now we remove the escape chars.
	nval = strings.Replace(nval, "\\\\", "\\", -1)
	if nval != "" &&
		(strings.HasPrefix(nval, "\"") &&
			strings.HasSuffix(nval, "\"")) ||
		(strings.HasPrefix(nval, "'") &&
			strings.HasSuffix(nval, "'")) {
		//Discount the quotes " .. "
		nval = nval[1 : len(nval)-1]
	}
	return nval
}
