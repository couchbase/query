//  Copyright 2015-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package command

import (
	"fmt"
	"runtime"

	"github.com/couchbase/query/errors"
)

/*
The following variables are used to display the error

	messages in red text and then reset the terminal prompt
	color.
*/
var reset = "\x1b[0m"
var fgRed = "\x1b[31m"

func init() {
	if runtime.GOOS == "windows" {
		reset = ""
		fgRed = ""
	}
}

// Methods that get and set display variables

func SetDispVal(newreset, newfgRed string) {
	reset = newreset
	fgRed = newfgRed
}

func Getreset() string {
	return reset
}

func GetfgRed() string {
	return fgRed
}

/* The handleError method creates the error using the methods
   defined in the n1ql errors package. This is where all the
   shell errors are handled.
*/

func HandleError(err errors.ErrorCode, msg string) errors.Error {

	switch err {

	//Connection errors
	case errors.E_SHELL_CONNECTION_REFUSED:
		return errors.NewShellErrorCannotConnect(msg)
	case errors.E_SHELL_UNSUPPORTED_PROTOCOL:
		return errors.NewShellErrorUnsupportedProtocol(SERVICE_URL)
	case errors.E_SHELL_NO_SUCH_HOST:
		return errors.NewShellErrorNoSuchHost(SERVICE_URL)
	case errors.E_SHELL_NO_HOST_IN_REQUEST_URL:
		return errors.NewShellErrorNoHostInRequestUrl(SERVICE_URL)
	case errors.E_SHELL_UNKNOWN_PORT_TCP:
		return errors.NewShellErrorUnknownPorttcp(SERVICE_URL)
	case errors.E_SHELL_NO_ROUTE_TO_HOST:
		return errors.NewShellErrorNoRouteToHost(SERVICE_URL)
	case errors.E_SHELL_UNREACHABLE_NETWORK:
		return errors.NewShellErrorUnreachableNetwork("")
	case errors.E_SHELL_NO_CONNECTION:
		return errors.NewShellErrorNoConnection("")
	case errors.E_SHELL_DRIVER_OPEN:
		return errors.NewShellErrorDriverOpen(msg)
	case errors.E_SHELL_INVALID_URL:
		return errors.NewShellErrorInvalidURL(msg)
	case errors.E_SHELL_INVALID_PROTOCOL:
		return errors.NewShellErrorInvalidProtocol()

	//Read/Write/Update file errors
	case errors.E_SHELL_READ_FILE:
		return errors.NewShellErrorReadFile(msg)
	case errors.E_SHELL_WRITE_FILE:
		return errors.NewShellErrorWriteFile(msg)
	case errors.E_SHELL_OPEN_FILE:
		return errors.NewShellErrorOpenFile(msg)
	case errors.E_SHELL_CLOSE_FILE:
		return errors.NewShellErrorCloseFile(msg)

	//Authentication Errors.
	case errors.E_SHELL_INVALID_PASSWORD:
		return errors.NewShellErrorInvalidPassword(msg)
	case errors.E_SHELL_INVALID_USERNAME:
		return errors.NewShellErrorInvalidUsername("")
	case errors.E_SHELL_MISSING_CREDENTIAL:
		return errors.NewShellErrorMissingCredential("")
	case errors.E_SHELL_INVALID_CREDENTIAL:
		return errors.NewShellErrorInvalidCredential("")

	//Command Errors
	case errors.E_SHELL_NO_SUCH_COMMAND:
		return errors.NewShellErrorNoSuchCommand(msg)
	case errors.E_SHELL_NO_SUCH_PARAM:
		return errors.NewShellErrorNoSuchParam(msg)
	case errors.E_SHELL_TOO_MANY_ARGS:
		return errors.NewShellErrorTooManyArgs("")
	case errors.E_SHELL_TOO_FEW_ARGS:
		return errors.NewShellErrorTooFewArgs("")
	case errors.E_SHELL_STACK_EMPTY:
		return errors.NewShellErrorStackEmpty("")
	case errors.E_SHELL_NO_SUCH_ALIAS:
		return errors.NewShellErrorNoSuchAlias(msg)
	case errors.E_SHELL_BATCH_MODE:
		return errors.NewShellErrorBatchMode("")

	//Generic Errors
	case errors.E_SHELL_OPERATION_TIMEOUT:
		return errors.NewShellErrorOperationTimeout(SERVICE_URL)
	case errors.E_SHELL_ROWS_SCAN:
		return errors.NewShellErrorRowsScan(msg)
	case errors.E_SHELL_JSON_MARSHAL:
		return errors.NewShellErrorJsonMarshal(msg)
	case errors.E_SHELL_JSON_UNMARSHAL:
		return errors.NewShellErrorJsonUnmarshal(msg)
	case errors.E_SHELL_DRIVER_QUERY_METHOD:
		return errors.NewShellErrorDriverQueryMethod(msg)
	case errors.E_SHELL_WRITER_OUTPUT:
		return errors.NewShellErrorWriterOutput(msg)
	case errors.E_SHELL_UNBALANCED_QUOTES:
		return errors.NewShellErrorUnbalancedQuotes("")
	case errors.E_SHELL_ROWS_CLOSE:
		return errors.NewShellErrorRowsClose(msg)
	case errors.E_SHELL_CMD_LINE_ARGS:
		return errors.NewShellErrorCmdLineArgs("")
	case errors.E_SHELL_INVALID_INPUT_ARGUMENTS:
		return errors.NewShellErrorInvalidInputArguments("")
	case errors.E_SHELL_INVALID_ARGUMENT:
		return errors.NewShellErrorInvalidArgument()
	default:
		return errors.NewShellErrorUnknownError(msg)
	}

}

/*
Function to print the error in Red.
*/
func PrintError(s_err errors.Error) {
	tmpstr := fmt.Sprintln(fgRed, "ERROR", s_err.Code(), ":", s_err, reset)
	OUTPUT.WriteString(tmpstr + NEWLINE)
}
