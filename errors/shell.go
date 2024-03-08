//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

// Shell errors -- errors in the command line shell

const (
	CONNECTION_REFUSED_MSG   = ""
	UNSUPPORTED_PROTOCOL_MSG = "Unsupported protocol scheme "
	INVALID_PROTOCOL_MSG     = "Invalid protocol. Mixed protocols are not permitted in engine list."
	NO_SUCH_HOST_MSG         = "No such host "
	NO_HOST_IN_URL_MSG       = "No host in request URL "
	UNKNOWN_PORT_TCP_MSG     = "Unknown port "
	NO_ROUTE_TO_HOST_MSG     = "No route to host "
	UNREACHABLE_NETWORK_MSG  = "Network is unreachable."
	NO_CONNECTION_MSG        = "Not connected to any cluster. Use \\CONNECT command."
	DRIVER_OPEN_MSG          = ""
	INVALID_URL_MSG          = "Invalid input URL "

	READ_FILE_MSG  = "Error during file read "
	WRITE_FILE_MSG = "Error during file write "
	FILE_OPEN_MSG  = "Unable to open file "
	FILE_CLOSE_MSG = "Unable to close file "

	INVALID_PASSWORD_MSG   = "Invalid password "
	INVALID_USERNAME_MSG   = "Invalid username. "
	MISSING_CREDENTIAL_MSG = "Username missing in -credentials/-c option."
	INVALID_CREDENTIAL_MSG = "Invalid format for credentials. Separate username and password by a :. "

	NO_SUCH_COMMAND_MSG = "Command does not exist."
	NO_SUCH_PARAM_MSG   = "Parameter does not exist "
	TOO_MANY_ARGS_MSG   = "Too many input arguments to command."
	TOO_FEW_ARGS_MSG    = "Too few input arguments to command."
	STACK_EMPTY_MSG     = "Stack empty."
	NO_SUCH_ALIAS_MSG   = "Alias does not exist "
	BATCH_MODE_MSG      = "Error when running in batch mode for Analytics. Incorrect input value"
	STRING_WRITE_MSG    = "Cannot write to string buffer. "

	OPERATION_TIMEOUT_MSG       = "Operation timed out. Check query service url "
	ROWS_SCAN_MSG               = ""
	JSON_MARSHAL_MSG            = ""
	JSON_UNMARSHAL_MSG          = ""
	DRIVER_QUERY_MSG            = ""
	WRITER_OUTPUT_MSG           = "Error with io Writer. "
	UNBALANCED_PAREN_MSG        = "Unbalanced parenthesis in the input."
	ROWS_CLOSE_MSG              = ""
	CMD_LINE_ARG_MSG            = "Place input argument URL at the end, after input flags. "
	INVALID_INPUT_ARGUMENTS_MSG = "Input Argument format is invalid."
	INVALID_ARGUMENT_MSG        = "Invalid argument."
	ERROR_ON_REFRESH_MSG        = "Query APIs cannot be initialized from Cluster Map."

	UNKNOWN_ERROR_MSG = ""
)

// Connection errors
func NewShellErrorCannotConnect(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_CONNECTION_REFUSED, IKey: "shell.connection.refused",
		InternalMsg: CONNECTION_REFUSED_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnsupportedProtocol(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_UNSUPPORTED_PROTOCOL, IKey: "shell.unsupported.protocol",
		InternalMsg: UNSUPPORTED_PROTOCOL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidProtocol() Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_PROTOCOL, IKey: "shell.invalid.protocol",
		InternalMsg: INVALID_PROTOCOL_MSG, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_SUCH_HOST, IKey: "shell.no.such.host",
		InternalMsg: NO_SUCH_HOST_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoHostInRequestUrl(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_HOST_IN_REQUEST_URL, IKey: "shell.no.host.in.request.url",
		InternalMsg: NO_HOST_IN_URL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnknownPorttcp(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_UNKNOWN_PORT_TCP, IKey: "shell.unknown.port.tcp",
		InternalMsg: UNKNOWN_PORT_TCP_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoRouteToHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_ROUTE_TO_HOST, IKey: "shell.no.route.to.host",
		InternalMsg: NO_ROUTE_TO_HOST_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnreachableNetwork(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_UNREACHABLE_NETWORK, IKey: "shell.unreachable.network",
		InternalMsg: UNREACHABLE_NETWORK_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoConnection(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_CONNECTION, IKey: "shell.not.connected.to.any.instance",
		InternalMsg: NO_CONNECTION_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorDriverOpen(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_DRIVER_OPEN, IKey: "shell.driver.Open.method.error",
		InternalMsg: DRIVER_OPEN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidURL(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_URL, IKey: "shell.Invalid.URL",
		InternalMsg: INVALID_URL_MSG + msg, InternalCaller: CallerN(1)}
}

// Read/Write/Update file errors
func NewShellErrorReadFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_READ_FILE, IKey: "shell.read.history",
		InternalMsg: READ_FILE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriteFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_WRITE_FILE, IKey: "shell.write.history",
		InternalMsg: WRITE_FILE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorOpenFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_OPEN_FILE, IKey: "shell.unable.to.open.file",
		InternalMsg: FILE_OPEN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorCloseFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_CLOSE_FILE, IKey: "shell.unable.to.open.file",
		InternalMsg: FILE_CLOSE_MSG + msg, InternalCaller: CallerN(1)}
}

// Authentication Errors. Missing or invalid username/password.
func NewShellErrorInvalidPassword(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_PASSWORD, IKey: "shell.invalid.password",
		InternalMsg: INVALID_PASSWORD_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidUsername(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_USERNAME, IKey: "shell.invalid.username",
		InternalMsg: INVALID_USERNAME_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorMissingCredential(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_MISSING_CREDENTIAL, IKey: "shell.missing.credentials",
		InternalMsg: MISSING_CREDENTIAL_MSG + msg, InternalCaller: CallerN(1)}

}

func NewShellErrorInvalidCredential(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_CREDENTIAL, IKey: "shell.invalid.credentials",
		InternalMsg: INVALID_CREDENTIAL_MSG + msg, InternalCaller: CallerN(1)}

}

// Command Errors
func NewShellErrorNoSuchCommand(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_SUCH_COMMAND, IKey: "shell.no.such.command",
		InternalMsg: NO_SUCH_COMMAND_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchParam(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_SUCH_PARAM, IKey: "shell.no.such.param",
		InternalMsg: NO_SUCH_PARAM_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooManyArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_TOO_MANY_ARGS, IKey: "shell.too.many.args",
		InternalMsg: TOO_MANY_ARGS_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooFewArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_TOO_FEW_ARGS, IKey: "shell.too.few.args",
		InternalMsg: TOO_FEW_ARGS_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorStackEmpty(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_STACK_EMPTY, IKey: "shell.parameter.stack.empty",
		InternalMsg: STACK_EMPTY_MSG + msg, InternalCaller: CallerN(1)}

}

func NewShellErrorNoSuchAlias(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_NO_SUCH_ALIAS, IKey: "shell.alias.does.not.exist",
		InternalMsg: NO_SUCH_ALIAS_MSG + msg, InternalCaller: CallerN(1)}

}

func NewShellErrorBatchMode(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_BATCH_MODE, IKey: "batch.mode.incorrect.input",
		InternalMsg: BATCH_MODE_MSG + msg, InternalCaller: CallerN(1)}

}

func NewShellErrorStringWrite(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_STRING_WRITE, IKey: "string.buffer.write.error",
		InternalMsg: STRING_WRITE_MSG + msg, InternalCaller: CallerN(1)}

}

//Generic Errors

func NewShellErrorOperationTimeout(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_OPERATION_TIMEOUT, IKey: "shell.operation.timeout",
		InternalMsg: OPERATION_TIMEOUT_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorRowsScan(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_ROWS_SCAN, IKey: "shell.rows.scan.error",
		InternalMsg: ROWS_SCAN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonMarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_JSON_MARSHAL, IKey: "shell.json.marshal.error",
		InternalMsg: JSON_MARSHAL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonUnmarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_JSON_UNMARSHAL, IKey: "shell.json.unmarshal.error",
		InternalMsg: JSON_UNMARSHAL_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorDriverQueryMethod(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_DRIVER_QUERY_METHOD, IKey: "shell.driver.query.method.error",
		InternalMsg: DRIVER_QUERY_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriterOutput(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_WRITER_OUTPUT, IKey: "shell.write.to.writer.error",
		InternalMsg: WRITER_OUTPUT_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnbalancedParen(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_UNBALANCED_PAREN, IKey: "shell.unbalanced.parenthesis",
		InternalMsg: UNBALANCED_PAREN_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorRowsClose(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_ROWS_CLOSE, IKey: "shell.rows.close.error",
		InternalMsg: ROWS_CLOSE_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorCmdLineArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_CMD_LINE_ARGS, IKey: "shell.command.line.args",
		InternalMsg: CMD_LINE_ARG_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidInputArguments(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_INPUT_ARGUMENTS, IKey: "shell.invalid.input.arguments",
		InternalMsg: INVALID_INPUT_ARGUMENTS_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnknownError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_UNKNOWN, IKey: "shell.internal.error.uncaptured",
		InternalMsg: UNKNOWN_ERROR_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorOnRefresh(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_ON_REFRESH, IKey: "shell.cluster.map.refresh.error",
		InternalMsg: ERROR_ON_REFRESH_MSG + msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidArgument() Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INVALID_ARGUMENT, IKey: "shell.invalid.argument",
		InternalMsg: INVALID_ARGUMENT_MSG, InternalCaller: CallerN(1)}
}

func NewShellErrorInitTerminal(c error) Error {
	return &err{level: EXCEPTION, ICode: E_SHELL_INIT_FAILURE, IKey: "shell.init.terminal.failure", cause: c,
		InternalMsg: "Terminal set-up failed (check not legacy console)", InternalCaller: CallerN(1)}
}
