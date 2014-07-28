//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package server

import (
	"time"

	"github.com/couchbaselabs/query/execution"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/value"
)

type RequestChannel chan Request

type Request interface {
	RequestTime() time.Time
	Timeout() time.Duration
	Namespace() string
	Command() string
	Plan() plan.Operator
	Arguments() map[string]value.Value
	Output() execution.Output
	Fail(err error)
	Start()
	Finish()
	Expire()
}
