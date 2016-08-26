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
	"fmt"
	"io"

	"github.com/couchbase/query/errors"
)

/* The following variables are used to display the error
   messages in red text and then reset the terminal prompt
   color.
*/
var reset = "\x1b[0m"
var fgRed = "\x1b[31m"

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

func HandleError(err int, msg string) errors.Error {

	switch err {

	//Connection errors
	case errors.CONNECTION_REFUSED:
		return errors.NewShellErrorCannotConnect(msg)
	case errors.UNSUPPORTED_PROTOCOL:
		return errors.NewShellErrorUnsupportedProtocol(SERVICE_URL)
	case errors.NO_SUCH_HOST:
		return errors.NewShellErrorNoSuchHost(SERVICE_URL)
	case errors.NO_HOST_IN_URL:
		return errors.NewShellErrorNoHostInRequestUrl(SERVICE_URL)
	case errors.UNKNOWN_PORT_TCP:
		return errors.NewShellErrorUnknownPorttcp(SERVICE_URL)
	case errors.NO_ROUTE_TO_HOST:
		return errors.NewShellErrorNoRouteToHost(SERVICE_URL)
	case errors.UNREACHABLE_NETWORK:
		return errors.NewShellErrorUnreachableNetwork("")
	case errors.NO_CONNECTION:
		return errors.NewShellErrorNoConnection("")
	case errors.DRIVER_OPEN:
		return errors.NewShellErrorDriverOpen(msg)
	case errors.INVALID_URL:
		return errors.NewShellErrorInvalidURL(msg)

	//Read/Write/Update file errors
	case errors.READ_FILE:
		return errors.NewShellErrorReadFile(msg)
	case errors.WRITE_FILE:
		return errors.NewShellErrorWriteFile(msg)
	case errors.FILE_OPEN:
		return errors.NewShellErrorOpenFile(msg)
	case errors.FILE_CLOSE:
		return errors.NewShellErrorCloseFile(msg)

	//Authentication Errors.
	case errors.INVALID_PASSWORD:
		return errors.NewShellErrorInvalidPassword(msg)
	case errors.INVALID_USERNAME:
		return errors.NewShellErrorInvalidUsername("")
	case errors.MISSING_CREDENTIAL:
		return errors.NewShellErrorMissingCredential("")
	case errors.INVALID_CREDENTIAL:
		return errors.NewShellErrorInvalidCredential("")

	//Command Errors
	case errors.NO_SUCH_COMMAND:
		return errors.NewShellErrorNoSuchCommand(msg)
	case errors.NO_SUCH_PARAM:
		return errors.NewShellErrorNoSuchParam(msg)
	case errors.TOO_MANY_ARGS:
		return errors.NewShellErrorTooManyArgs("")
	case errors.TOO_FEW_ARGS:
		return errors.NewShellErrorTooFewArgs("")
	case errors.STACK_EMPTY:
		return errors.NewShellErrorStackEmpty("")
	case errors.NO_SUCH_ALIAS:
		return errors.NewShellErrorNoSuchAlias(msg)
	case errors.BATCH_MODE:
		return errors.NewShellErrorBatchMode("")

	//Generic Errors
	case errors.OPERATION_TIMEOUT:
		return errors.NewShellErrorOperationTimeout(SERVICE_URL)
	case errors.ROWS_SCAN:
		return errors.NewShellErrorRowsScan(msg)
	case errors.JSON_MARSHAL:
		return errors.NewShellErrorJsonMarshal(msg)
	case errors.JSON_UNMARSHAL:
		return errors.NewShellErrorJsonUnmarshal(msg)
	case errors.DRIVER_QUERY:
		return errors.NewShellErrorDriverQueryMethod(msg)
	case errors.WRITER_OUTPUT:
		return errors.NewShellErrorWriterOutput(msg)
	case errors.UNBALANCED_PAREN:
		return errors.NewShellErrorUnbalancedParen("")
	case errors.ROWS_CLOSE:
		return errors.NewShellErrorRowsClose(msg)
	case errors.CMD_LINE_ARG:
		return errors.NewShellErrorCmdLineArgs("")

	default:
		return errors.NewShellErrorUnkownError(msg)
	}

}

/*
	Function to print the error in Red.
*/
func PrintError(s_err errors.Error) {
	tmpstr := fmt.Sprintln(fgRed, "ERROR", s_err.Code(), ":", s_err, reset)
	io.WriteString(W, tmpstr+"\n")
}
