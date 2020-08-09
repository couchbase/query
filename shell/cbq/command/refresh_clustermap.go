//  Copyright (c) 2020 Couchbase, Inc.
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

var REFRESH_URL string

/* Redirect Command */
type Refresh_cluster_map struct {
	ShellCommand
}

func (this *Refresh_cluster_map) Name() string {
	return "REFRESH_CLUSTER_MAP"
}

func (this *Refresh_cluster_map) CommandCompletion() bool {
	return false
}

func (this *Refresh_cluster_map) MinArgs() int {
	return ZERO_ARGS
}

func (this *Refresh_cluster_map) MaxArgs() int {
	return ZERO_ARGS
}

func (this *Refresh_cluster_map) ExecCommand(args []string) (int, string) {
	if len(args) != 0 {
		return errors.TOO_MANY_ARGS, ""
	} else {
		// REFRESH LOGIC HERE
		// Set new Service URL and then Ping
		err := Ping(REFRESH_URL)
		if err != nil {
			return errors.ERROR_ON_REFRESH, err.Error()
		}
	}
	return 0, ""
}

func (this *Refresh_cluster_map) PrintHelp(desc bool) (int, string) {
	_, werr := io.WriteString(W, HREFRESH_CLUSTERMAP)
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
