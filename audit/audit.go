//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package audit

import (
	"github.com/couchbase/query/logging"
)

type Auditable interface {
	// success/fatal/stopped/etc.
	EventResult() string

	// The N1QL statement executed.
	Statement() string

	// Statement id.
	EventId() string

	// Event type. eg. "SELECT", "DELETE", "PREPARE"
	EventType() string

	// Event start time in RFC3339Nano format in UTC, eg. "2017-11-07T14:31:27.800880428Z"
	EventTimestamp() string

	// User ids submitted with request. eg. ["kirk", "spock"]
	EventUsers() []string

	// The User-Agent string from the request. This is used to identify the type of client
	// that sent the request (SDK, QWB, CBQ, ...)
	UserAgent() string

	// The address the request came from.
	RemoteAddr() string

	// Event server name.
	EventServerName() string

	// Event execution metrics.
	EventElapsedTime() string
	EventExecutionTime() string
	EventResultCount() int
	EventResultSize() int
	MutationCount() uint64
	SortCount() uint64
	EventErrorCount() int
	EventWarningCount() int
}

var doAudit = false

func Submit(event Auditable) {
	if !doAudit {
		return
	}
	// For now, just log the audit events.
	logging.Infof("result=\"%s\", statement=\"%s\", id=\"%s\", type=\"%s\", timestamp=\"%s\", users=%v, user_agent=\"%s\", client_address=\"%s\", server_name=\"%s\"",
		event.EventResult(), event.Statement(), event.EventId(), event.EventType(), event.EventTimestamp(), event.EventUsers(),
		event.UserAgent(), event.RemoteAddr(), event.EventServerName())
	logging.Infof("elapsed_time=%s, execution_time=%s, result_count=%d, result_size=%d, mutation_count=%d, sort_count=%d, error_count=%d, warning_count=%d",
		event.EventElapsedTime(), event.EventExecutionTime(), event.EventResultCount(), event.EventResultSize(),
		event.MutationCount(), event.SortCount(), event.EventErrorCount(), event.EventWarningCount())
}
