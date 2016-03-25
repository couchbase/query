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
		return errors.NewShellErrorCannotConnect("Unable to connect to " + SERVICE_URL + ". " + msg)
	case errors.UNSUPPORTED_PROTOCOL:
		return errors.NewShellErrorUnsupportedProtocol("Unsupported Protocol Scheme " + SERVICE_URL)
	case errors.NO_SUCH_HOST:
		return errors.NewShellErrorNoSuchHost("No such Host " + SERVICE_URL)
	case errors.NO_HOST_IN_URL:
		return errors.NewShellErrorNoHostInRequestUrl("No Host in request URL " + SERVICE_URL)
	case errors.UNKNOWN_PORT_TCP:
		return errors.NewShellErrorUnknownPorttcp("Unknown port " + SERVICE_URL)
	case errors.NO_ROUTE_TO_HOST:
		return errors.NewShellErrorNoRouteToHost("No Route to host " + SERVICE_URL)
	case errors.UNREACHABLE_NETWORK:
		return errors.NewShellErrorUnreachableNetwork("Network is unreachable ")
	case errors.NO_CONNECTION:
		return errors.NewShellErrorNoConnection("Not Connected to any instance. Use \\CONNECT command. ")
	case errors.DRIVER_OPEN:
		return errors.NewShellErrorDriverOpen(msg)
	case errors.INVALID_URL:
		return errors.NewShellErrorInvalidURL("Invalid input url : " + msg)

	//Read/Write/Update file errors
	case errors.READ_FILE:
		return errors.NewShellErrorReadFile("Error during file read. " + msg)
	case errors.WRITE_FILE:
		return errors.NewShellErrorWriteFile("Error during file write. " + msg)
	case errors.FILE_OPEN:
		return errors.NewShellErrorOpenFile("Unable to open file. " + msg)
	case errors.FILE_CLOSE:
		return errors.NewShellErrorCloseFile("Unable to close file. ")

	//Authentication Errors.
	case errors.INVALID_PASSWORD:
		return errors.NewShellErrorInvalidPassword("Invalid Password. " + msg)
	case errors.INVALID_USERNAME:
		return errors.NewShellErrorInvalidUsername("Invalid Username. ")
	case errors.MISSING_CREDENTIAL:
		return errors.NewShellErrorMissingCredential("Username missing in -credentials/-c option.")

	//Command Errors
	case errors.NO_SUCH_COMMAND:
		return errors.NewShellErrorNoSuchCommand("Command does not exist.")
	case errors.NO_SUCH_PARAM:
		return errors.NewShellErrorNoSuchParam("Parameter does not exist : " + msg)
	case errors.TOO_MANY_ARGS:
		return errors.NewShellErrorTooManyArgs("Too many input arguments to command.")
	case errors.TOO_FEW_ARGS:
		return errors.NewShellErrorTooFewArgs("Too few input arguments to command.")
	case errors.STACK_EMPTY:
		return errors.NewShellErrorStackEmpty("Stack Empty.")
	case errors.NO_SUCH_ALIAS:
		return errors.NewShellErrorNoSuchAlias("Alias does not exist : " + msg)

	//Generic Errors
	case errors.OPERATION_TIMEOUT:
		return errors.NewShellErrorOperationTimeout("Operation timed out. Check query service url " + SERVICE_URL)
	case errors.ROWS_SCAN:
		return errors.NewShellErrorRowsScan(msg)
	case errors.JSON_MARSHAL:
		return errors.NewShellErrorJsonMarshal(msg)
	case errors.JSON_UNMARSHAL:
		return errors.NewShellErrorJsonUnmarshal(msg)
	case errors.DRIVER_QUERY:
		return errors.NewShellErrorDriverQueryMethod(msg)
	case errors.WRITER_OUTPUT:
		return errors.NewShellErrorWriterOutput("Error with io Writer. " + msg)
	case errors.UNBALANCED_PAREN:
		return errors.NewShellErrorUnbalancedParen("Unbalanced Parenthesis in the input.")
	case errors.ROWS_CLOSE:
		return errors.NewShellErrorRowsClose(msg)
	case errors.CMD_LINE_ARG:
		return errors.NewShellErrorCmdLineArgs("Place input argument url at the end, after input flags. ")

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
