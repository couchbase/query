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

	"github.com/couchbase/query/errors"
)

/* Connect Command */
type Connect struct {
	ShellCommand
}

func (this *Connect) Name() string {
	return "CONNECT"
}

func (this *Connect) CommandCompletion() bool {
	return false
}

func (this *Connect) MinArgs() int {
	return ONE_ARG
}

func (this *Connect) MaxArgs() int {
	return ONE_ARG
}

func (this *Connect) ExecCommand(args []string) (int, string) {
	/* Command to connect to the input query service or cluster
	   endpoint. Use the Server flag and set it to the value
	   of service_url. If the command contains no input argument
	   or more than 1 argument then throw an error.
	*/
	if len(args) > this.MaxArgs() {
		return errors.TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.TOO_FEW_ARGS, ""
	} else {
		SERVICE_URL = args[0]
		io.WriteString(W, "\nEndpoint to Connect to : "+SERVICE_URL+" . Type Ctrl-D / \\exit / \\quit to exit.\n")
	}
	return 0, ""
}

func (this *Connect) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\CONNECT <url>\n")
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
