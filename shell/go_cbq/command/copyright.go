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

/* Copyright Command */
type Copyright struct {
	ShellCommand
}

func (this *Copyright) Name() string {
	return "COPYRIGHT"
}

func (this *Copyright) CommandCompletion() bool {
	return false
}

func (this *Copyright) MinArgs() int {
	return 0
}

func (this *Copyright) MaxArgs() int {
	return 0
}

func (this *Copyright) ExecCommand(args []string) (int, string) {
	/* Print the Copyright information for the shell. If the
	   command contains an input argument then throw an error.
	*/
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""
	} else {
		io.WriteString(W, "Copyright (c) 2015 Couchbase, Inc. Licensed under the Apache License, Version 2.0 (the \"License\"); \nyou may not use this file except in compliance with the License. You may obtain a copy of the \nLicense at http://www.apache.org/licenses/LICENSE-2.0\nUnless required by applicable law or agreed to in writing, software distributed under the\nLicense is distributed on an \"AS IS\" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,\neither express or implied. See the License for the specific language governing permissions\nand limitations under the License.")
	}
	return 0, ""
}

func (this *Copyright) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, "\\COPYRIGHT\n")
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
