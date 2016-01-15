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
	"io"
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

func (this *Echo) ExecCommand(args []string) (int, string) {
	var werr error
	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.TOO_FEW_ARGS, ""

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
				tmpstr = strings.Replace(tmp[0], "\"", "", -1)

				//Use the string value directly as output.
				_, werr = io.WriteString(W, tmpstr)
				_, werr = io.WriteString(W, " ")

			} else {
				// If the value type is string then output it directly.
				if v.Type() == value.STRING {
					//Use the string value directly as output.
					_, werr = io.WriteString(W, v.Actual().(string))
					_, werr = io.WriteString(W, " ")

				} else {
					// Convert non string values to string and then output.
					_, werr = io.WriteString(W, ValToStr(v))
					_, werr = io.WriteString(W, " ")

				}

			}

		}
	}

	_, werr = io.WriteString(W, "\n")
	if werr != nil {
		return errors.WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}

func (this *Echo) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\ECHO <arg>...\n")
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
