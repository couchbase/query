//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"io"
	"strings"

	"github.com/couchbase/godbc/n1ql"
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

func (this *Connect) ExecCommand(args []string) (errors.ErrorCode, string) {
	/* Command to connect to the input query service or cluster
	   endpoint. Use the Server flag and set it to the value
	   of service_url. If the command contains no input argument
	   or more than 1 argument then throw an error.
	*/
	if len(args) > this.MaxArgs() {
		return errors.E_SHELL_TOO_MANY_ARGS, ""

	} else if len(args) < this.MinArgs() {
		return errors.E_SHELL_TOO_FEW_ARGS, ""
	} else {
		SERVICE_URL = args[0]

		// Support couchbase couchbases when using the connect command as well.
		// call command.ParseURL()
		var errCode errors.ErrorCode
		var errStr string
		var pURL *UrlRes

		pURL, errCode, errStr = ParseURL(SERVICE_URL)

		if errCode != 0 {
			return errCode, errStr
		}

		SERVICE_URL = pURL.ServerFlag

		// Connect to secure ports depending on -no-ssl-verify flag when cbq is started.
		if strings.HasPrefix(strings.ToLower(SERVICE_URL), "https://") {
			if SKIPVERIFY == false {
				PrintStr(W, SSLVERIFY_FALSE)
			} else {
				PrintStr(W, SSLVERIFY_TRUE)
			}
		}
		if pURL.Username != "" && pURL.Password != "" {
			n1ql.SetUsernamePassword(pURL.Username, pURL.Password)
		}

		// Do the check for different values here as well.
		err := Ping(SERVICE_URL)
		if err != nil {
			return errors.E_SHELL_CONNECTION_REFUSED, err.Error()
		}
		io.WriteString(W, NewMessage(STARTUP, SERVICE_URL)+EXITMSG)
	}
	return 0, ""
}

func (this *Connect) PrintHelp(desc bool) (errors.ErrorCode, string) {
	_, werr := io.WriteString(W, HCONNECT)
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
