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
	CONNECTION_REFUSED   = 100
	UNSUPPORTED_PROTOCOL = 101
	NO_SUCH_HOST         = 102
	NO_HOST_IN_URL       = 103
	UNKNOWN_PORT_TCP     = 104
	NO_ROUTE_TO_HOST     = 105
	UNREACHABLE_NETWORK  = 106
	OPERATION_TIMEOUT    = 120
)

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

func NewShellErrorOperationTimeout(msg string) Error {
        return &err{level: EXCEPTION, ICode: OPERATION_TIMEOUT, IKey: "shell.operation.timeout", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewShellErrorUnreachableNetwork(msg string) Error {
        return &err{level: EXCEPTION, ICode: UNREACHABLE_NETWORK, IKey: "shell.unreachable.network", InternalMsg: msg, InternalCaller: CallerN(1)}
}
