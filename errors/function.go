//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package errors

import (
	"fmt"
)

const (
	//Function errors
	FTS_MISSING_PORT = 10003
	NODE_ACCESS_ERR  = 10004
)

func NewFTSMissingPortErr(e string) Error {
	return &err{level: EXCEPTION, ICode: FTS_MISSING_PORT, IKey: "fts.url.format.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Missing or Incorrect port in input url."),
		InternalCaller: CallerN(1)}
}

func NewNodeInfoAccessErr(e string) Error {
	return &err{level: EXCEPTION, ICode: NODE_ACCESS_ERR, IKey: "node.access.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("Issue with accessing node information for rest endpoint %v", e),
		InternalCaller: CallerN(1)}
}

func NewNodeServiceErr(e string) Error {
	return &err{level: EXCEPTION, ICode: NODE_ACCESS_ERR, IKey: "node.service.error", ICause: fmt.Errorf("%v", e),
		InternalMsg:    fmt.Sprintf("No FTS node in server %v", e),
		InternalCaller: CallerN(1)}
}
