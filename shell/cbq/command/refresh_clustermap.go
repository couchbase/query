//  Copyright 2020-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

func (this *Refresh_cluster_map) ExecCommand(args []string) (errors.ErrorCode, string) {
	if len(args) != 0 {
		return errors.E_SHELL_TOO_MANY_ARGS, ""
	} else {
		// REFRESH LOGIC HERE
		// Set new Service URL and then Ping
		err := Ping(REFRESH_URL)
		if err != nil {
			// try the SERVICE_URL if the REFRESH_URL fails
			err = Ping(SERVICE_URL)
			if err != nil {
				return errors.E_SHELL_ON_REFRESH, err.Error()
			}
		}
	}
	return 0, ""
}

func (this *Refresh_cluster_map) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HREFRESH_CLUSTERMAP)
	if desc {
		err_code, err_str := printDesc(this.Name())
		if err_code != 0 {
			return err_code, err_str
		}
	}
	_, werr = io.WriteString(W, NEWLINE)
	if werr != nil {
		return errors.E_SHELL_WRITER_OUTPUT, werr.Error()
	}
	return 0, ""
}
