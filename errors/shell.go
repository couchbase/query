//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

import ()

// Shell errors -- errors in the command line shell

const (
	//Connection errors (100 - 115)
	CONNECTION_REFUSED       = 100
	CONNECTION_REFUSED_MSG   = "Unable to connect to "
	UNSUPPORTED_PROTOCOL     = 101
	UNSUPPORTED_PROTOCOL_MSG = "Unsupported protocol scheme "
	NO_SUCH_HOST             = 102
	NO_SUCH_HOST_MSG         = "No such host "
	NO_HOST_IN_URL           = 103
	NO_HOST_IN_URL_MSG       = "No host in request URL "
	UNKNOWN_PORT_TCP         = 104
	UNKNOWN_PORT_TCP_MSG     = "Unknown port "
	NO_ROUTE_TO_HOST         = 105
	NO_ROUTE_TO_HOST_MSG     = "No route to host "
	UNREACHABLE_NETWORK      = 106
	UNREACHABLE_NETWORK_MSG  = "Network is unreachable."
	NO_CONNECTION            = 107
	NO_CONNECTION_MSG        = "Not connected to any cluster. Use \\CONNECT command."
	DRIVER_OPEN              = 108
	DRIVER_OPEN_MSG          = ""
	INVALID_URL              = 109
	INVALID_URL_MSG          = "Invalid input URL "

	//Read/Write/Update file errors (116 - 120)
	READ_FILE      = 116
	READ_FILE_MSG  = "Error during file read "
	WRITE_FILE     = 117
	WRITE_FILE_MSG = "Error during file write "
	FILE_OPEN      = 118
	FILE_OPEN_MSG  = "Unable to open file "
	FILE_CLOSE     = 119
	FILE_CLOSE_MSG = "Unable to close file "

	//Authentication Errors (121 - 135)
	//Missing or invalid username/password.
	INVALID_PASSWORD       = 121
	INVALID_PASSWORD_MSG   = "Invalid password "
	INVALID_USERNAME       = 122
	INVALID_USERNAME_MSG   = "Invalid username. "
	MISSING_CREDENTIAL     = 123
	MISSING_CREDENTIAL_MSG = "Username missing in -credentials/-c option."

	//Command Errors (136 - 169)
	NO_SUCH_COMMAND     = 136
	NO_SUCH_COMMAND_MSG = "Command does not exist."
	NO_SUCH_PARAM       = 137
	NO_SUCH_PARAM_MSG   = "Parameter does not exist "
	TOO_MANY_ARGS       = 138
	TOO_MANY_ARGS_MSG   = "Too many input arguments to command."
	TOO_FEW_ARGS        = 139
	TOO_FEW_ARGS_MSG    = "Too few input arguments to command."
	STACK_EMPTY         = 140
	STACK_EMPTY_MSG     = "Stack empty."
	NO_SUCH_ALIAS       = 141
	NO_SUCH_ALIAS_MSG   = "Alias does not exist "

	//Generic Errors (170 - 199)
	OPERATION_TIMEOUT     = 170
	OPERATION_TIMEOUT_MSG = "Operation timed out. Check query service url "
	ROWS_SCAN             = 171
	ROWS_SCAN_MSG         = ""
	JSON_MARSHAL          = 172
	JSON_MARSHAL_MSG      = ""
	JSON_UNMARSHAL        = 173
	JSON_UNMARSHAL_MSG    = ""
	DRIVER_QUERY          = 174
	DRIVER_QUERY_MSG      = ""
	WRITER_OUTPUT         = 175
	WRITER_OUTPUT_MSG     = "Error with io Writer. "
	UNBALANCED_PAREN      = 176
	UNBALANCED_PAREN_MSG  = "Unbalanced parenthesis in the input."
	ROWS_CLOSE            = 177
	ROWS_CLOSE_MSG        = ""
	CMD_LINE_ARG          = 178
	CMD_LINE_ARG_MSG      = "Place input argument URL at the end, after input flags. "

	//Untracked error
	UNKNOWN_ERROR     = 199
	UNKNOWN_ERROR_MSG = ""
)

//Connection errors
func NewShellErrorCannotConnect(msg string) Error {
	return &err{level: EXCEPTION, ICode: CONNECTION_REFUSED, IKey: "shell.connection.refused", InternalMsg: CONNECTION_REFUSED_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnsupportedProtocol(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNSUPPORTED_PROTOCOL, IKey: "shell.unsupported.protocol", InternalMsg: UNSUPPORTED_PROTOCOL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_HOST, IKey: "shell.no.such.host", InternalMsg: NO_SUCH_HOST_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoHostInRequestUrl(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_HOST_IN_URL, IKey: "shell.no.host.in.request.url", InternalMsg: NO_HOST_IN_URL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnknownPorttcp(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_PORT_TCP, IKey: "shell.unknown.port.tcp", InternalMsg: UNKNOWN_PORT_TCP_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoRouteToHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_ROUTE_TO_HOST, IKey: "shell.no.route.to.host", InternalMsg: NO_ROUTE_TO_HOST_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnreachableNetwork(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNREACHABLE_NETWORK, IKey: "shell.unreachable.network", InternalMsg: UNREACHABLE_NETWORK_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoConnection(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_CONNECTION, IKey: "shell.not.connected.to.any.instance", InternalMsg: NO_CONNECTION_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorDriverOpen(msg string) Error {
	return &err{level: EXCEPTION, ICode: DRIVER_OPEN, IKey: "shell.driver.Open.method.error", InternalMsg: DRIVER_OPEN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidURL(msg string) Error {
	return &err{level: EXCEPTION, ICode: INVALID_URL, IKey: "shell.Invalid.URL", InternalMsg: INVALID_URL_MSG + msg, InternalCaller: CallerN(1)}
}

//Read/Write/Update file errors
func NewShellErrorReadFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: READ_FILE, IKey: "shell.read.history", InternalMsg: READ_FILE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriteFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: WRITE_FILE, IKey: "shell.write.history", InternalMsg: WRITE_FILE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorOpenFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: FILE_OPEN, IKey: "shell.unable.to.open.file", InternalMsg: FILE_OPEN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorCloseFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: FILE_CLOSE, IKey: "shell.unable.to.open.file", InternalMsg: FILE_CLOSE_MSG + msg, InternalCaller: CallerN(1)}
}

//Authentication Errors. Missing or invalid username/password.
func NewShellErrorInvalidPassword(msg string) Error {
	return &err{level: EXCEPTION, ICode: INVALID_PASSWORD, IKey: "shell.invalid.password", InternalMsg: INVALID_PASSWORD_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidUsername(msg string) Error {
	return &err{level: EXCEPTION, ICode: INVALID_USERNAME, IKey: "shell.invalid.username", InternalMsg: INVALID_USERNAME_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorMissingCredential(msg string) Error {
	return &err{level: EXCEPTION, ICode: MISSING_CREDENTIAL, IKey: "shell.missing.credentials", InternalMsg: MISSING_CREDENTIAL_MSG + msg, InternalCaller: CallerN(1)}

}

//Command Errors
func NewShellErrorNoSuchCommand(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_COMMAND, IKey: "shell.no.such.command", InternalMsg: NO_SUCH_COMMAND_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchParam(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_PARAM, IKey: "shell.no.such.param", InternalMsg: NO_SUCH_PARAM_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooManyArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: TOO_MANY_ARGS, IKey: "shell.too.many.args", InternalMsg: TOO_MANY_ARGS_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooFewArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: TOO_FEW_ARGS, IKey: "shell.too.few.args", InternalMsg: TOO_FEW_ARGS_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorStackEmpty(msg string) Error {
	return &err{level: EXCEPTION, ICode: STACK_EMPTY, IKey: "shell.parameter.stack.empty", InternalMsg: STACK_EMPTY_MSG + msg, InternalCaller: CallerN(1)}

}

func NewShellErrorNoSuchAlias(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_ALIAS, IKey: "shell.alias.does.not.exist", InternalMsg: NO_SUCH_ALIAS_MSG + msg, InternalCaller: CallerN(1)}

}

//Generic Errors

func NewShellErrorOperationTimeout(msg string) Error {
	return &err{level: EXCEPTION, ICode: OPERATION_TIMEOUT, IKey: "shell.operation.timeout", InternalMsg: OPERATION_TIMEOUT_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorRowsScan(msg string) Error {
	return &err{level: EXCEPTION, ICode: ROWS_SCAN, IKey: "shell.rows.scan.error", InternalMsg: ROWS_SCAN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonMarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: JSON_MARSHAL, IKey: "shell.json.marshal.error", InternalMsg: JSON_MARSHAL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonUnmarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: JSON_UNMARSHAL, IKey: "shell.json.unmarshal.error", InternalMsg: JSON_UNMARSHAL_MSG + msg, InternalCaller: CallerN(1)}
}
func NewShellErrorDriverQueryMethod(msg string) Error {
	return &err{level: EXCEPTION, ICode: DRIVER_QUERY, IKey: "shell.driver.query.method.error", InternalMsg: DRIVER_QUERY_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriterOutput(msg string) Error {
	return &err{level: EXCEPTION, ICode: WRITER_OUTPUT, IKey: "shell.write.to.writer.error", InternalMsg: WRITER_OUTPUT_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnbalancedParen(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNBALANCED_PAREN, IKey: "shell.unbalanced.parenthesis", InternalMsg: UNBALANCED_PAREN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorRowsClose(msg string) Error {
	return &err{level: EXCEPTION, ICode: ROWS_CLOSE, IKey: "shell.rows.close.error", InternalMsg: ROWS_CLOSE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorCmdLineArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: CMD_LINE_ARG, IKey: "shell.command.line.args", InternalMsg: CMD_LINE_ARG_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnkownError(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_ERROR, IKey: "shell.internal.error.uncaptured", InternalMsg: UNKNOWN_ERROR_MSG + msg, InternalCaller: CallerN(1)}
}
