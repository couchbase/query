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

/* Version Command */
type Version struct {
	ShellCommand
}

func (this *Version) Name() string {
	return "VERSION"
}

func (this *Version) CommandCompletion() bool {
	return false
}

func (this *Version) MinArgs() int {
	return 0
}

func (this *Version) MaxArgs() int {
	return 0
}

func (this *Version) ExecCommand(args []string) (int, string) {
	/* Print the shell version. If the command contains an input
	   argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""
	} else {
		_, werr := io.WriteString(W, "SHELL VERSION : "+SHELL_VERSION+"\n")
		_, werr = io.WriteString(W, "Use N1QL commands select version() or select min_version() to display server version.\n")
		if werr != nil {
			return errors.WRITER_OUTPUT, werr.Error()
		}
	}
	return 0, ""
}

func (this *Version) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\VERSION\n")
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
