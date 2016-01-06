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
	CONNECTION_REFUSED   = 100
	UNSUPPORTED_PROTOCOL = 101
	NO_SUCH_HOST         = 102
	NO_HOST_IN_URL       = 103
	UNKNOWN_PORT_TCP     = 104
	NO_ROUTE_TO_HOST     = 105
	UNREACHABLE_NETWORK  = 106
	NO_CONNECTION        = 107
	GO_N1QL_OPEN         = 108

	//Read/Write/Update file errors (116 - 120)
	READ_FILE  = 116
	WRITE_FILE = 117
	FILE_OPEN  = 118
	FILE_CLOSE = 119

	//Authentication Errors (121 - 135)
	//Missing or invalid username/password.
	INVALID_PASSWORD   = 121
	INVALID_USERNAME   = 122
	MISSING_CREDENTIAL = 123

	//Command Errors (136 - 169)
	NO_SUCH_COMMAND = 136
	NO_SUCH_PARAM   = 137
	TOO_MANY_ARGS   = 138
	TOO_FEW_ARGS    = 139
	STACK_EMPTY     = 140
	NO_SUCH_ALIAS   = 141

	//Generic Errors (170 - 199)
	OPERATION_TIMEOUT = 170
	ROWS_SCAN         = 171
	JSON_MARSHAL      = 172
	JSON_UNMARSHAL    = 173
	GON1QL_QUERY      = 174
	WRITER_OUTPUT     = 175
	UNBALANCED_PAREN  = 176
	ROWS_CLOSE        = 177

	//Untracked error
	UNKNOWN_ERROR = 199
)

//Connection errors
func NewShellErrorCannotConnect(msg string) Error {
	return &err{level: EXCEPTION, ICode: CONNECTION_REFUSED, IKey: "shell.connection.refused", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnsupportedProtocol(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNSUPPORTED_PROTOCOL, IKey: "shell.unsupported.protocol", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_HOST, IKey: "shell.no.such.host", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoHostInRequestUrl(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_HOST_IN_URL, IKey: "shell.no.host.in.request.url", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnknownPorttcp(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_PORT_TCP, IKey: "shell.unknown.port.tcp", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoRouteToHost(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_ROUTE_TO_HOST, IKey: "shell.no.route.to.host", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnreachableNetwork(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNREACHABLE_NETWORK, IKey: "shell.unreachable.network", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoConnection(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_CONNECTION, IKey: "shell.not.connected.to.any.instance", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorGon1qlOpen(msg string) Error {
	return &err{level: EXCEPTION, ICode: GO_N1QL_OPEN, IKey: "shell.gon1ql.Open.method.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

//Read/Write/Update file errors
func NewShellErrorReadFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: READ_FILE, IKey: "shell.read.history", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriteFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: WRITE_FILE, IKey: "shell.write.history", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorOpenFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: FILE_OPEN, IKey: "shell.unable.to.open.file", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorCloseFile(msg string) Error {
	return &err{level: EXCEPTION, ICode: FILE_CLOSE, IKey: "shell.unable.to.open.file", InternalMsg: msg, InternalCaller: CallerN(1)}
}

//Authentication Errors. Missing or invalid username/password.
func NewShellErrorInvalidPassword(msg string) Error {
	return &err{level: EXCEPTION, ICode: INVALID_PASSWORD, IKey: "shell.invalid.password", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorInvalidUsername(msg string) Error {
	return &err{level: EXCEPTION, ICode: INVALID_USERNAME, IKey: "shell.invalid.username", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorMissingCredential(msg string) Error {
	return &err{level: EXCEPTION, ICode: MISSING_CREDENTIAL, IKey: "shell.missing.credentials", InternalMsg: msg, InternalCaller: CallerN(1)}

}

//Command Errors
func NewShellErrorNoSuchCommand(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_COMMAND, IKey: "shell.no.such.command", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorNoSuchParam(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_PARAM, IKey: "shell.no.such.param", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooManyArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: TOO_MANY_ARGS, IKey: "shell.too.many.args", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorTooFewArgs(msg string) Error {
	return &err{level: EXCEPTION, ICode: TOO_FEW_ARGS, IKey: "shell.too.few.args", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorStackEmpty(msg string) Error {
	return &err{level: EXCEPTION, ICode: STACK_EMPTY, IKey: "shell.parameter.stack.empty", InternalMsg: msg, InternalCaller: CallerN(1)}

}

func NewShellErrorNoSuchAlias(msg string) Error {
	return &err{level: EXCEPTION, ICode: NO_SUCH_ALIAS, IKey: "shell.alias.does.not.exist", InternalMsg: msg, InternalCaller: CallerN(1)}

}

//Generic Errors

func NewShellErrorOperationTimeout(msg string) Error {
	return &err{level: EXCEPTION, ICode: OPERATION_TIMEOUT, IKey: "shell.operation.timeout", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorRowsScan(msg string) Error {
	return &err{level: EXCEPTION, ICode: ROWS_SCAN, IKey: "shell.rows.scan.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonMarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: JSON_MARSHAL, IKey: "shell.json.marshal.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorJsonUnmarshal(msg string) Error {
	return &err{level: EXCEPTION, ICode: JSON_UNMARSHAL, IKey: "shell.json.unmarshal.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}
func NewShellErrorGon1qlQueryMethod(msg string) Error {
	return &err{level: EXCEPTION, ICode: GON1QL_QUERY, IKey: "shell.gon1ql.query.method.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorWriterOutput(msg string) Error {
	return &err{level: EXCEPTION, ICode: WRITER_OUTPUT, IKey: "shell.write.to.writer.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnbalancedParen(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNBALANCED_PAREN, IKey: "shell.unbalanced.parenthesis", InternalMsg: msg, InternalCaller: CallerN(1)}

}

func NewShellErrorRowsClose(msg string) Error {
	return &err{level: EXCEPTION, ICode: ROWS_CLOSE, IKey: "shell.rows.close.error", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnkownError(msg string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_ERROR, IKey: "shell.internal.error.uncaptured", InternalMsg: msg, InternalCaller: CallerN(1)}
}
