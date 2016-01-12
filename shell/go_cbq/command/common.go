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
	"strconv"
	"strings"

	go_n1ql "github.com/couchbase/go_n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
	"github.com/sbinet/liner"
)

//type PtrStrings *[]string

var (
	QueryParam map[string]*Stack = map[string]*Stack{}
	NamedParam map[string]*Stack = map[string]*Stack{}
	UserDefSV  map[string]*Stack = map[string]*Stack{}
	PreDefSV   map[string]*Stack = map[string]*Stack{
		"limit":      Stack_Helper(),
		"histfile":   Stack_Helper(),
		"histsize":   Stack_Helper(),
		"autoconfig": Stack_Helper(),
	}
)

type Credential map[string]string
type Credentials []Credential

var creds Credentials

func init() {

	/* Populate the Predefined user variable map with default
	   values.
	*/

	var err_code int
	var err_str string
	//var werr error

	err_code, err_str = PushValue_Helper(false, PreDefSV, "histfile", "\".cbq_history\"")
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)

	}

	err_code, err_str = PushValue_Helper(false, PreDefSV, "autoconfig", "false")
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)
	}

	histlim := int(liner.HistoryLimit)
	err_code, err_str = PushValue_Helper(false, PreDefSV, "histsize", strconv.Itoa(histlim))
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)
	}

	err_code, err_str = PushValue_Helper(false, PreDefSV, "limit", "0")
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)
	}
}

/* The Resolve method is used to evaluate the input parameter
   to the \SET / \PUSH / \POP / \UNSET and \ECHO commands. It
   takes in a string, and resolves it to the appropriate value.
   The input string can be broadly classified into 2 categories,
   1. Parameters (here we will need to read the top value from
   the parameter stack)
   2. Actual values that can be converted to value.Value using
   the StrToVal method.
*/
func Resolve(param string) (val value.Value, err_code int, err_str string) {

	/* Parse the input string to check whether it is a parameter
	   or a value. If it is a parameter, then we parse it
	   appropriately to check which stacks top value needs to be
	   returned.
	*/
	err_code = 0
	err_str = ""

	param = strings.TrimSpace(param)

	if strings.HasPrefix(param, "\\\\") {
		/* It is a Command alias */
		key := param[2:]
		st_val, ok := AliasCommand[key]
		if !ok {
			err_code = errors.NO_SUCH_ALIAS
			err_str = " " + key + " "
		} else {

			//Quote input properly so that resolve returns string and not binary.
			if !strings.HasPrefix(st_val, "\"") {
				st_val = "\"" + st_val + "\""
			}
			val = StrToVal(st_val)
		}

	} else if strings.HasPrefix(param, "-$") {
		key := param[2:]
		v, ok := NamedParam[key]
		if !ok {
			err_code = errors.NO_SUCH_PARAM
			err_str = " " + param + " "
		} else {
			val, err_code, err_str = v.Top()
		}

	} else if strings.HasPrefix(param, "-") {
		/* Then it is a query parameter. Retrieve its value and
		return.
		*/

		key := param[1:]
		v, ok := QueryParam[key]

		if !ok {
			err_code = errors.NO_SUCH_PARAM
			err_str = " " + param + " "
		} else {
			val, err_code, err_str = v.Top()

		}

	} else if strings.HasPrefix(param, "$") {
		key := param[1:]

		v, ok := UserDefSV[key]
		if !ok {
			err_code = errors.NO_SUCH_PARAM
			err_str = " " + param + " "
		} else {
			val, err_code, err_str = v.Top()
		}

	} else {

		/* There can be two possibilities. 1. Its a Predefined
		   Session Parameter. In this case we lookup its value
		   and return that. 2. It is a value, in which case we
		   directly convert it to a value.Value type and return
		   it.
		*/

		v, ok := PreDefSV[param]
		if ok {
			val, err_code, err_str = v.Top()
		} else {
			if !strings.HasPrefix(param, "\"") {
				param = "\"" + param + "\""
			}
			val = StrToVal(param)
		}
	}
	return
}

/* The StrToVal method converts the input string into a
   value.Value type.
*/
func StrToVal(param string) (val value.Value) {

	param = strings.TrimSpace(param)

	if strings.HasPrefix(param, "\"") {
		if strings.HasSuffix(param, "\"") {
			param = param[1 : len(param)-1]
		}
	}

	bytes := []byte(param)

	val = value.NewValue(bytes)

	if val.Type() == value.BINARY {
		param = "\"" + param + "\""
		bytes := []byte(param)
		val = value.NewValue(bytes)
	}
	return

}

/* The ValToStr method converts the input value into a
   string type.
*/
func ValToStr(item value.Value) string {
	//Call String() method in value.Value to convert
	//value to string.

	return item.String()
}

/* Helper function to push or set a value in a stack. */
func PushValue_Helper(set bool, param map[string]*Stack, vble, value string) (err_code int, err_str string) {
	err_code = 0
	err_str = ""

	st_Val, ok := param[vble]

	v, err_code, err_str := Resolve(value)
	if err_code != 0 {
		return err_code, err_str
	} else {
		//Stack already exists
		if ok {
			if set == true {
				err_code, err_str = st_Val.SetTop(v)
				if err_code != 0 {
					return err_code, err_str
				}
			} else if set == false {
				st_Val.Push(v)
			}

		} else {
			/* If the stack for the input variable is empty then
			   push the current value onto the variable stack.
			*/
			param[vble] = Stack_Helper()
			param[vble].Push(v)
		}
	}
	return

}

/* Helper function to pop or unset a value in a stack. */
func PopValue_Helper(unset bool, param map[string]*Stack, vble string) (err_code int, err_str string) {
	err_code = 0
	err_str = ""

	st_Val, ok := param[vble]

	if unset == true {
		// Unset the enire stack for given parameter
		if ok {
			for st_Val.Len() > 0 {
				_, err_code, err_str := st_Val.Pop()
				if err_code != 0 {
					return err_code, err_str
				}
			}
			//While unsetting also delete the stack for the
			//given variable.
			delete(param, vble)
		} else {
			err_code = errors.NO_SUCH_PARAM
			err_str = ""
		}
	} else {
		//To pop a value from the input stack
		if ok {
			_, err_code, err_str = st_Val.Pop()

			// If after popping the stack is now empty, then delete the stack.
			// We dont need to check for stack empty here because ok will be false
			// if the stack is empty. So it will return Parameter doesnt exist.
			if st_Val.Len() == 0 {
				delete(param, vble)
			}
		} else {
			err_code = errors.NO_SUCH_PARAM
			err_str = ""
		}
	}
	return

}

func ToCreds(credsFlag string) (Credentials, int, string) {

	//Handle the input string of credentials.
	//The string needs to be parsed into a byte array so as to pass to go_n1ql.
	cred := strings.Split(credsFlag, ",")
	var creds Credentials
	creds = append(creds, Credential{"user": "", "pass": ""})

	/* Append input credentials in [{"user": <username>, "pass" : <password>}]
	format as expected by go_n1ql creds.
	*/
	for _, i := range cred {
		up := strings.Split(i, ":")

		//Make sure there are no leading and trailing spaces
		//when processing the username and password.
		up[0] = strings.TrimSpace(up[0])
		up[1] = strings.TrimSpace(up[1])
		if len(up) < 2 {
			// One of the input credentials is incorrect
			return nil, errors.MISSING_CREDENTIAL, ""
		} else {
			creds = append(creds, Credential{"user": up[0], "pass": up[1]})
		}
	}
	return creds, 0, ""

}

func PushOrSet(args []string, pushvalue bool) (int, string) {

	// Check what kind of parameter needs to be set or pushed
	// depending on the pushvalue boolean value.

	if strings.HasPrefix(args[0], "-$") {

		// For Named Parameters
		vble := args[0]
		vble = vble[2:]

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, NamedParam, vble, args_str)
		if err_code != 0 {
			return err_code, err_str
		}
		//Pass the named parameters to the rest api using the SetQueryParams method
		v, err_code, err_str := NamedParam[vble].Top()
		if err_code != 0 {
			return err_code, err_str
		}

		val := ValToStr(v)

		vble = "$" + vble
		go_n1ql.SetQueryParams(vble, val)

	} else if strings.HasPrefix(args[0], "-") {
		// For query parameters

		vble := args[0]
		vble = vble[1:]

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, QueryParam, vble, args_str)

		if err_code != 0 {
			return err_code, err_str
		}

		if vble == "creds" {
			// Define credentials as user/pass and convert into
			//   JSON object credentials

			var creds Credentials
			args_str := strings.Join(args[1:], " ")
			creds_ret, err_code, err_str := ToCreds(args_str)

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
			go_n1ql.SetQueryParams("creds", string(ac))

		} else {

			v, err_code, err_str := QueryParam[vble].Top()
			if err_code != 0 {
				return err_code, err_str
			}

			// When passing the query rest api parameter to go_n1ql
			// we need to convert to string only if the value isnt
			// already a string
			var val string = ""
			if v.Type() == value.STRING {
				val = v.Actual().(string)
			} else {
				val = ValToStr(v)
			}

			go_n1ql.SetQueryParams(vble, val)

		}

	} else if strings.HasPrefix(args[0], "$") {

		// For User defined session variables
		vble := args[0]
		vble = vble[1:]

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, UserDefSV, vble, args_str)
		if err_code != 0 {
			return err_code, err_str
		}

	} else {
		// For Predefined session variables

		vble := args[0]

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, PreDefSV, vble, args_str)

		if vble == "histfile" {
			HISTFILE = args[1]
		}

		if err_code != 0 {
			return err_code, err_str
		}
	}
	return 0, ""
}

func printDesc(cmdname string) (int, string) {
	var werr error
	switch cmdname {

	case ALIAS_CMD:
		_, werr = io.WriteString(W, "Create an alias for input. <command> = <shell command> or <query statement>\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\ALIAS serverversion \"select version(), min_version()\" ;\n\t        \\ALIAS \"\\SET -max-parallelism 8\";\n")

	case CONNECT_CMD:
		_, werr = io.WriteString(W, "Connect to the query service or cluster endpoint url.\n")
		_, werr = io.WriteString(W, "Default : http://localhost:8091\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\CONNECT http://172.6.23.2:8091 ; \n\t         \\CONNECT https://my.secure.node.com:18093 ;\n")

	case COPYRIGHT_CMD:
		_, werr = io.WriteString(W, "Print Couchbase Copyright information\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\COPYRIGHT;\n")

	case DISCONNECT_CMD:
		_, werr = io.WriteString(W, "Disconnect from the query service or cluster endpoint url.\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\DISCONNECT;")

	case ECHO_CMD:
		_, werr = io.WriteString(W, "Echo the value of the input. <arg> = <prefix><name> (a parameter) or \n <arg> = <alias> (command alias) or \n <arg> = <input> (any input statement) \n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\ECHO -$r ;\n\t        \\ECHO \\Com; \n")

	case EXIT_CMD:
		_, werr = io.WriteString(W, "Exit the shell\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\EXIT; \n\t        \\QUIT;\n")

	case HELP_CMD:
		_, werr = io.WriteString(W, "The input arguments are shell commands. If a * is input then the command displays HELP information for all input shell commands.\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\HELP VERSION; \n\t        \\HELP EXIT DISCONNECT VERSION; \n\t        \\HELP;\n")

	case POP_CMD:
		_, werr = io.WriteString(W, "Pop the value of the given parameter from the input parameter stack. <parameter> = <prefix><name>\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\Pop -$r ;\n\t        \\Pop $Val ; \n\t        \\Pop ;\n")

	case PUSH_CMD:
		_, werr = io.WriteString(W, "Push the value of the given parameter to the input parameter stack. <parameter> = <prefix><name>\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\PUSH -$r 9.5 ;\n\t        \\PUSH $Val -$r; \n\t        \\PUSH ;\n")

	case SET_CMD:
		_, werr = io.WriteString(W, "Set the value of the given parameter to the input value. <parameter> = <prefix><name>\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\SET -$r 9.5 ;\n\t        \\SET $Val -$r ;\n")

	case SOURCE_CMD:
		_, werr = io.WriteString(W, "Load input file into shell\n")
		_, werr = io.WriteString(W, " For Example : \n\t \\SOURCE temp1.txt ;\n")

	case UNALIAS_CMD:
		_, werr = io.WriteString(W, "Delete the alias given by <alias name>.\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\UNALIAS serverversion;\n\t        \\UNALIAS subcommand1 subcommand2 serverversion;\n")

	case UNSET_CMD:
		_, werr = io.WriteString(W, "Unset the value of the given parameter. <parameter> = <prefix><name> \n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\Unset -$r ;\n\t        \\Unset $Val ;\n")

	case VERSION_CMD:
		_, werr = io.WriteString(W, "Print the Shell Version\n")
		_, werr = io.WriteString(W, "\tExample : \n\t        \\VERSION;\n")

	default:
		_, werr = io.WriteString(W, "IshaFix : Does not exist\n")

	}
	if werr != nil {
		return errors.WRITER_OUTPUT, werr.Error()
	}
	return 0, ""

}
