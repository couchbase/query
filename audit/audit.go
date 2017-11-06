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

// Use Event... as prefix of every method, to avoid interference
// with existing interfaces.
type Auditable interface {
	// success/fatal/stopped/etc.
	EventResult() string

	// The N1QL statement executed.
	EventStatement() string

	// Statement id.
	EventId() string

	// Event type. eg. "SELECT", "DELETE", "PREPARE"
	EventType() string
}

var doAudit = false

func Submit(event Auditable) {
	if !doAudit {
		return
	}
	// For now, just log the audit events.
	logging.Infof("result=\"%s\", statement=\"%s\", id=\"%s\", type=\"%s\"", event.EventResult(), event.EventStatement(), event.EventId(), event.EventType())
}
