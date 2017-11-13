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
	adt "github.com/couchbase/goutils/go-cbaudit"
	"github.com/couchbase/query/logging"
)

var _AUDIT_SERVICE *adt.AuditSvc

func StartAuditService(server string) {
	var err error
	_AUDIT_SERVICE, err = adt.NewAuditSvc(server)
	if err == nil {
		logging.Infof("Audit service started.")
	} else {
		logging.Errorf("Audit service not started: %v", err)
	}
}

type Auditable interface {
	// Standard fields used for all audit records.
	EventGenericFields() adt.GenericFields

	// success/fatal/stopped/etc.
	EventStatus() string

	// The N1QL statement executed.
	Statement() string

	// Statement id.
	EventId() string

	// Event type. eg. "SELECT", "DELETE", "PREPARE"
	EventType() string

	// User ids submitted with request. eg. ["kirk", "spock"]
	EventUsers() []string

	// The User-Agent string from the request. This is used to identify the type of client
	// that sent the request (SDK, QWB, CBQ, ...)
	UserAgent() string

	// Event server name.
	EventNodeName() string

	// Query parameters.
	EventNamedArgs() map[string]string
	EventPositionalArgs() []string

	IsAdHoc() bool
}

var doAudit = false
var doLogAuditEvent = false

func Submit(event Auditable) {
	if !doAudit {
		return
	}

	if doLogAuditEvent {
		logAuditEvent(event)
	}

	if _AUDIT_SERVICE == nil {
		return
	}

	eventType := event.EventType()
	if eventType == "SELECT" {
		// We build the audit record from the request in the main thread
		// because the request will be destroyed soon after the call to Submit(),
		// and we don't want to cause a race condition.
		auditRecord := buildAuditRecord(event)
		go submitForAudit(28672, auditRecord)
	} else {
		logging.Infof("Event submitted for audit of unknown type %s. Not dispatched.", eventType)
	}
}

func buildAuditRecord(event Auditable) *n1qlAuditEvent {
	return &n1qlAuditEvent{
		GenericFields:  event.EventGenericFields(),
		RequestId:      event.EventId(),
		Statement:      event.Statement(),
		NamedArgs:      event.EventNamedArgs(),
		PositionalArgs: event.EventPositionalArgs(),
		Users:          event.EventUsers(),
		IsAdHoc:        event.IsAdHoc(),
		UserAgent:      event.UserAgent(),
		Node:           event.EventNodeName(),
		Status:         event.EventStatus(),
	}
}

func submitForAudit(eventId uint32, auditRecord *n1qlAuditEvent) {
	err := _AUDIT_SERVICE.Write(eventId, *auditRecord)
	if err != nil {
		logging.Errorf("Unable to submit event %+v for audit: %v", *auditRecord, err)
	}
}

func logAuditEvent(event Auditable) {
	logging.Infof("status=\"%s\", statement=\"%s\", id=\"%s\", type=\"%s\", users=%v, user_agent=\"%s\", user_agent=\"%s\", node_name=\"%s\"",
		event.EventStatus(), event.Statement(), event.EventId(), event.EventType(), event.EventUsers(),
		event.UserAgent(), event.EventNodeName())
	logging.Infof("named_args=%v, positional_args=%v, ad_hoc=%v", event.EventNamedArgs(), event.EventPositionalArgs(), event.IsAdHoc())
}

// If possible, use whatever field names are used elsewhere in the N1QL system.
// Follow whatever naming scheme (under_scores/camelCase/What.Ever) is standard for each field.
// If no standard exists for the field, use camelCase.
type n1qlAuditEvent struct {
	adt.GenericFields

	RequestId      string            `json:"requestId"`
	Statement      string            `json:"statement"`
	NamedArgs      map[string]string `json:"namedArgs,omitempty"`
	PositionalArgs []string          `json:"positionalArgs,omitempty"`

	Users     []string `json:"users"`
	IsAdHoc   bool     `json:"isAdHoc"`
	UserAgent string   `json:"userAgent"`
	Node      string   `json:"node"`

	Status string `json:"status"`
}
