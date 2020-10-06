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
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/couchbase/godbc/n1ql"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

var (
	QueryParam map[string]*Stack = map[string]*Stack{}
	NamedParam map[string]*Stack = map[string]*Stack{}
	UserDefSV  map[string]*Stack = map[string]*Stack{}
	PreDefSV   map[string]*Stack = map[string]*Stack{
		"histfile": Stack_Helper(),
		"batch":    Stack_Helper(),
		"quiet":    Stack_Helper(),
	}
)

type Credential map[string]string
type Credentials []Credential

var DbN1ql n1ql.N1qlDB

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

	err_code, err_str = PushValue_Helper(false, PreDefSV, "batch", BATCH)
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)

	}

	err_code, err_str = PushValue_Helper(false, PreDefSV, "quiet", strconv.FormatBool(QUIET))
	if err_code != 0 {
		s_err := HandleError(err_code, err_str)
		PrintError(s_err)
	}

}

func SetWriter(Wt io.Writer) {
	W = Wt
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

		key = strings.ToLower(key)

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

	// Escape any \ so that we dont conver to value.BINARY
	if strings.Contains(param, "\\") {
		param = strings.Replace(param, "\\", "\\\\", -1)
		bytes := []byte(param)
		val = value.NewValue(bytes)
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
func PushValue_Helper(set bool, param map[string]*Stack, vble, value_ip string) (err_code int, err_str string) {
	err_code = 0
	err_str = ""

	st_Val, ok := param[vble]

	v, err_code, err_str := Resolve(value_ip)
	if err_code != 0 {
		return err_code, err_str
	} else {

		//if the input value is a BINARY value, then throw an error.
		if v.Type() == value.BINARY {
			return errors.INVALID_INPUT_ARGUMENTS, ""
		}

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
			err_str = " " + vble + " "
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
			err_str = " " + vble + " "
		}
	}
	return

}

func ToCreds(credsFlag string) (Credentials, int, string) {

	//Handle the input string of credentials.
	//The string needs to be parsed into a byte array so as to pass to godbc/n1ql.
	cred := strings.Split(credsFlag, ",")
	var creds Credentials

	/* Append input credentials in [{"user": <username>, "pass" : <password>}]
	format as expected by godbc/n1ql creds.
	*/

	for _, i := range cred {

		// Make sure the format of the credentials is correct.
		// If not return an error.
		if !strings.Contains(i, ":") {
			return nil, errors.INVALID_CREDENTIAL, ""
		}

		up := strings.Split(i, ":")

		switch len(up) {
		case 0, 1:
			return nil, errors.MISSING_CREDENTIAL, ""
			//Make sure there are no leading and trailing spaces
		case 2:
			up[0] = strings.TrimSpace(up[0])
			up[1] = strings.TrimSpace(up[1])
		default:
			// Support passwords like "local:xxx" or "admin:xxx"
			up[0] = strings.TrimSpace(up[0])
			up[1] = strings.Join(up[1:], ":")
		}

		//when processing the username and password.
		if up[0] == "" && up[1] != "" {
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

		if strings.TrimSpace(vble) == "" {
			return errors.TOO_FEW_ARGS, ""
		}

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
		n1ql.SetQueryParams(vble, val)

	} else if strings.HasPrefix(args[0], "-") {
		// For query parameters

		vble := args[0]
		vble = vble[1:]

		if strings.TrimSpace(vble) == "" {
			return errors.TOO_FEW_ARGS, ""
		}

		vble = strings.ToLower(vble)

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, QueryParam, vble, args_str)

		if err_code != 0 {
			return err_code, err_str
		}

		if vble == "creds" {
			// Define credentials as user/pass and convert into
			//  JSON object credentials

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

			n1ql.SetQueryParams("creds", string(ac))
			n1ql.SetUsernamePassword(creds[0]["user"], creds[0]["pass"])

		} else {

			v, err_code, err_str := QueryParam[vble].Top()
			if err_code != 0 {
				return err_code, err_str
			}

			// When passing the query rest api parameter to godbc/n1ql
			// we need to convert to string only if the value isnt
			// already a string
			var val string = ""
			if v.Type() == value.STRING {
				val = v.Actual().(string)
			} else {
				val = ValToStr(v)
			}

			n1ql.SetQueryParams(vble, val)

		}

	} else if strings.HasPrefix(args[0], "$") {

		// For User defined session variables
		vble := args[0]
		vble = vble[1:]

		if strings.TrimSpace(vble) == "" {
			return errors.TOO_FEW_ARGS, ""
		}

		args_str := strings.Join(args[1:], " ")

		err_code, err_str := PushValue_Helper(pushvalue, UserDefSV, vble, args_str)
		if err_code != 0 {
			return err_code, err_str
		}

	} else {
		// For Predefined session variables

		vble := args[0]

		vble = strings.ToLower(vble)

		args_str := strings.Join(args[1:], " ")

		if vble == "histfile" {
			//Verify if the value for histfile is valid.
			//the path is given is relative to the HOME dir.
			//dir+"/"+HISTFILE ==>

			homeDir, err_code, err_str := GetHome()
			if err_code != 0 {
				return err_code, err_str
			}

			path := GetPath(homeDir, args_str)

			_, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
			//If err then the value for histfile is invalid. Hence return an error.
			//For this case, the HISTFILE will retain its original value.
			if err != nil {
				return errors.FILE_OPEN, err.Error()
			} else {
				HISTFILE = path
				if !QUIET {
					io.WriteString(W, NewMessage(HISTORYMSG, path)+" \n")
				}
			}
		} else if vble == "batch" {
			if args_str != "on" && args_str != "off" {
				return errors.BATCH_MODE, ""
			}
			BATCH = args_str
		} else if vble == "quiet" {
			var errQ error
			QUIET, errQ = strconv.ParseBool(args_str)
			if errQ != nil {
				return errors.INVALID_INPUT_ARGUMENTS, ""
			}
		}

		err_code, err_str := PushValue_Helper(pushvalue, PreDefSV, vble, args_str)
		if err_code != 0 {
			return err_code, err_str
		}

	}
	return 0, ""
}

func printDesc(cmdname string) (int, string) {

	switch cmdname {

	case ALIAS_CMD:
		return PrintStr(W, DALIAS)

	case CONNECT_CMD:
		return PrintStr(W, DCONNECT)

	case COPYRIGHT_CMD:
		return PrintStr(W, DCOPYRIGHT)

	case DISCONNECT_CMD:
		return PrintStr(W, DDISCONNECT)

	case ECHO_CMD:
		return PrintStr(W, DECHO)

	case EXIT_CMD:
		return PrintStr(W, DEXIT)

	case HELP_CMD:
		return PrintStr(W, DHELP)

	case POP_CMD:
		return PrintStr(W, DPOP)

	case PUSH_CMD:
		return PrintStr(W, DPUSH)

	case SET_CMD:
		return PrintStr(W, DSET)

	case SOURCE_CMD:
		return PrintStr(W, DSOURCE)

	case UNALIAS_CMD:
		return PrintStr(W, DUNALIAS)

	case UNSET_CMD:
		return PrintStr(W, DUNSET)

	case VERSION_CMD:
		return PrintStr(W, DVERSION)

	case REDIRECT_CMD:
		return PrintStr(W, DREDIRECT)

	case REFRESH_CLUSTER_MAP_CMD:
		return PrintStr(W, DREFRESH_CLUSTERMAP)

	default:
		return PrintStr(W, DDEFAULT)

	}
	return 0, ""

}

func Ping(server string) error {
	var err error
	oldDbN1ql := DbN1ql
	DbN1ql, err = n1ql.OpenExtended(server)
	if err != nil {
		DbN1ql = oldDbN1ql
		return err
	}

	err = DbN1ql.Ping()
	return err
}

/* Find the HOME environment variable. If it isnt set then
   try USERPROFILE for windows. If neither is found then
   the cli cant find the history file to read from.
*/
func GetHome() (homeDir string, err_code int, err_Str string) {
	//Detect OS using the runtime.GOOS
	if runtime.GOOS == "windows" {
		homeDir = os.Getenv("USERPROFILE")
		WINDOWS = true
	} else {
		homeDir = os.Getenv("HOME")
	}

	if homeDir == "" {
		_, werr := io.WriteString(W, ERRHOME)
		if werr != nil {
			return "", errors.WRITER_OUTPUT, werr.Error()
		}
	}
	return homeDir, 0, ""
}

func GetPath(homeDir, inputPath string) (path string) {
	//When verifying the path, check to see if input is an absolute path
	//or not. At this time for history we do not support Relative paths.

	//In order to handle cases such as
	//         \alias p /tmp/history.txt;
	//		   \set histfile \\p;
	//		   \echo histfile;
	//			Output : /tmp/history;
	//  OR     \set $a /tmp/history.txt;
	//			\set histfile $a;

	if strings.HasPrefix(inputPath, "\\\\") || strings.HasPrefix(inputPath, "$") {
		val, err_code, _ := Resolve(inputPath)
		if err_code != 0 {
			//This means it wasnt an alias and we need
			//to treat it as an input path directly.
		} else {
			//If there is no error, then we replace the
			//input path with the value of the alias.
			inputPath = ValToStr(val)
		}
	}

	//support paths with quotations.
	if (strings.HasPrefix(inputPath, "\"") &&
		strings.HasSuffix(inputPath, "\"")) ||
		(strings.HasPrefix(inputPath, "'") &&
			strings.HasSuffix(inputPath, "'")) {

		inputPath = inputPath[1 : len(inputPath)-1]
	}

	if WINDOWS {
		//Check for absolute path first. If not assume that the path is
		//relative to USERPROFILE.

		if strings.HasPrefix(inputPath, "\\") ||
			strings.Index(inputPath, ":\\") == 1 ||
			strings.HasPrefix(inputPath, "\\\\") {

			path = inputPath
		} else {
			path = homeDir + "\\" + inputPath
		}

	} else {
		if strings.HasPrefix(inputPath, "/") {
			//This is an absolute path. Hence we need not prefix it with
			//$HOME
			path = inputPath
		} else {
			//Prefix this with $HOME
			path = homeDir + "/" + inputPath
		}

	}
	return

}

func printPath(nval string) (int, string) {
	if !QUIET {
		homeDir, err_code, err_str := GetHome()
		if err_code != 0 {
			return err_code, err_str
		}

		path := GetPath(homeDir, nval)

		io.WriteString(W, NewMessage(HISTORYMSG, path)+" \n")
	}
	return 0, ""

}

//Use this method to writestring to output.
func PrintStr(W io.Writer, val string) (int, string) {
	_, werr := io.WriteString(W, val)
	if werr != nil {
		return errors.WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}

type UrlRes struct {
	ServerFlag string
	Username   string
	Password   string
}

// Method to return the new value of the server flag based on input url string
func ParseURL(serverFlag string) (*UrlRes, int, string) {

	pURL := &UrlRes{serverFlag, "", ""}

	// the URL golang Parse method has the limitation that when we pass in a host that is a string
	// and not an ip, without the protocol scheme, it mis-interprets the url string. For such cases we
	// need to explicitely make sure that we are missing a protocol scheme.

	// If no protocol exists, then append http:// as default protocol.

	if !strings.HasPrefix(strings.ToLower(serverFlag), "https://") &&
		!strings.HasPrefix(strings.ToLower(serverFlag), "http://") &&
		!strings.HasPrefix(strings.ToLower(serverFlag), "couchbase://") &&
		!strings.HasPrefix(strings.ToLower(serverFlag), "couchbases://") {
		//There is something else wrong and we need to throw an error.
		serverFlag = "http://" + serverFlag
	}

	//Parse the url
	parsedURL, err := url.Parse(serverFlag)
	if err != nil {
		return pURL, errors.INVALID_URL, err.Error()
	}

	if parsedURL.Host == "" {
		return pURL, errors.INVALID_URL, INVALIDHOST
	}

	// Check if the input url is a DNS SRV
	_, addr, err := net.LookupSRV(parsedURL.Scheme, "tcp", parsedURL.Hostname())
	if err == nil {
		// It is a DNS SRV .. Has couchbase or couchbases as a scheme
		parsedURL.Host = addr[0].Target
	}

	// We now have a valid URL. Check if we have a port
	_, portNo, err := net.SplitHostPort(parsedURL.Host)

	// couchbase:// and couchbases:// will represent http:// ... :8091 and
	// https:// ... 18091 respectively. If the port is specified along with
	// the scheme for this case, we throw an error.

	if parsedURL.Hostname() != "" {
		parsedURL.Host = parsedURL.Hostname()
	}

	if portNo == "" {
		if strings.ToLower(parsedURL.Scheme) == "couchbase" || strings.ToLower(parsedURL.Scheme) == "couchbases" {

			if strings.ToLower(parsedURL.Scheme) == "couchbase" {
				parsedURL.Host = net.JoinHostPort(parsedURL.Host, "8091")
				parsedURL.Scheme = "http"

			} else {
				parsedURL.Scheme = "https"
				parsedURL.Host = net.JoinHostPort(parsedURL.Host, "18091")
			}

		} else {
			if strings.ToLower(parsedURL.Scheme) == "http" {
				parsedURL.Host = net.JoinHostPort(parsedURL.Host, "8091")

			} else if strings.ToLower(parsedURL.Scheme) == "https" {
				parsedURL.Host = net.JoinHostPort(parsedURL.Host, "18091")
			}
		}
	} else {
		parsedURL.Host = net.JoinHostPort(parsedURL.Host, portNo)
		// Cannot give port with couchbase:// couchbases://
		if strings.ToLower(parsedURL.Scheme) == "couchbase" || strings.ToLower(parsedURL.Scheme) == "couchbases" {
			return pURL, errors.INVALID_URL, INVALIDPORT
		} else {
			if err != nil {
				return pURL, errors.INVALID_URL, err.Error()
			}
		}
	}

	pURL.Password, _ = parsedURL.User.Password()
	pURL.Username = parsedURL.User.Username()
	pURL.ServerFlag = parsedURL.String()

	return pURL, 0, ""
}
