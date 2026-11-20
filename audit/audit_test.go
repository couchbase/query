/*
Copyright 2017-Present Couchbase, Inc.

Use of this software is governed by the Business Source License included in
the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
file, in accordance with the Business Source License, use of this software will
be governed by the Apache License, Version 2.0, included in the file
licenses/APL2.txt.
*/

package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	adt "github.com/couchbase/goutils/go-cbaudit"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

// An auditor the just records the audit events that would be sent to the audit daemon,
// nothing more.
type mockAuditor struct {
	info           *datastore.AuditInfo
	recordedEvents []auditQueueEntry
	metricRegStub  accounting_stub.MetricRegistryStub
}

func (ma *mockAuditor) submit(entry auditQueueEntry) {
	ma.recordedEvents = append(ma.recordedEvents, entry)
}

func (ma *mockAuditor) setAuditInfo(info *datastore.AuditInfo) {
	ma.info = info
}

func (ma *mockAuditor) auditInfo() *datastore.AuditInfo {
	return ma.info
}

func (ma *mockAuditor) metricRegistry() accounting.MetricRegistry {
	return ma.metricRegStub
}

// A fixed structure that implements the Auditable interface
type simpleAuditable struct {
	genericFields          adt.GenericFields
	remote                 string
	local                  string
	status                 string
	statement              string
	queryContext           string
	eventId                string
	eventType              string
	eventUsers             []string
	userAgent              string
	eventNodeName          string
	eventNamedArgs         map[string]interface{}
	clientContextId        string
	eventPositionalArgs    []interface{}
	isAdHoc                bool
	preparedId             string
	elapsedTime            time.Duration
	executionTime          time.Duration
	transactionElapsedTime time.Duration
	eventResultCount       int
	eventResultSize        int
	mutationCount          uint64
	sortCount              uint64
	eventErrorCount        int
	eventWarningCount      int
	txId                   string
	errors                 []errors.Error
}

func (this *simpleAuditable) EventErrorMessage() []errors.Error {
	return this.errors
}

func (sa *simpleAuditable) EventGenericFields() adt.GenericFields {
	return sa.genericFields
}

func (sa *simpleAuditable) EventRemoteAddress() string {
	return sa.remote
}

func (sa *simpleAuditable) EventLocalAddress() string {
	return sa.local
}

func (sa *simpleAuditable) EventStatus() string {
	return sa.status
}

func (sa *simpleAuditable) EventStatement() string {
	return sa.statement
}

func (sa *simpleAuditable) EventQueryContext() string {
	return sa.queryContext
}

func (sa *simpleAuditable) EventId() string {
	return sa.eventId
}

func (sa *simpleAuditable) EventTxId() string {
	return sa.txId
}

func (sa *simpleAuditable) EventType() string {
	return sa.eventType
}

func (sa *simpleAuditable) EventUsers() []string {
	return sa.eventUsers
}

func (sa *simpleAuditable) UserAgent() string {
	return sa.userAgent
}

func (sa *simpleAuditable) EventNodeName() string {
	return sa.eventNodeName
}

func (sa *simpleAuditable) EventNamedArgs() map[string]interface{} {
	return sa.eventNamedArgs
}

func (sa *simpleAuditable) EventPositionalArgs() []interface{} {
	return sa.eventPositionalArgs
}

func (sa *simpleAuditable) ClientContextId() string {
	return sa.clientContextId
}

func (sa *simpleAuditable) IsAdHoc() bool {
	return sa.isAdHoc
}

func (sa *simpleAuditable) PreparedId() string {
	return sa.preparedId
}

func (sa *simpleAuditable) ElapsedTime() float64 {
	return sa.elapsedTime.Seconds()
}

func (sa *simpleAuditable) TransactionElapsedTime() float64 {
	return sa.transactionElapsedTime.Seconds()
}
func (sa *simpleAuditable) TransactionRemainingTime() string {
	return ""
}

func (sa *simpleAuditable) ExecutionTime() float64 {
	return sa.executionTime.Seconds()
}

func (sa *simpleAuditable) EventResultCount() int {
	return sa.eventResultCount
}

func (sa *simpleAuditable) EventResultSize() int {
	return sa.eventResultSize
}

func (sa *simpleAuditable) MutationCount() uint64 {
	return sa.mutationCount
}

func (sa *simpleAuditable) SortCount() uint64 {
	return sa.sortCount
}

func (sa *simpleAuditable) EventErrorCount() int {
	return sa.eventErrorCount
}

func (sa *simpleAuditable) EventWarningCount() int {
	return sa.eventWarningCount
}

func TestEventIdGeneration(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:    true,
			EventDisabled:   make(map[uint32]bool),
			UserAllowlisted: make(map[datastore.UserInfo]bool),
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT"}
	Submit(auditable)

	auditable.eventType = "INSERT"
	Submit(auditable)

	auditable.eventType = "UPDATE"
	Submit(auditable)

	auditable.eventType = "DELETE"
	Submit(auditable)

	auditable.eventType = "GARBAGE"
	Submit(auditable)

	expectedTypes := []uint32{28672, 28676, 28679, 28678, 28687}

	numEvents := len(mockAuditor.recordedEvents)
	if numEvents != len(expectedTypes) {
		t.Fatalf("Expected %d events, found %d", len(expectedTypes), numEvents)
	}

	for i, v := range expectedTypes {
		if v != mockAuditor.recordedEvents[i].eventId {
			t.Fatalf("Expected event id %d, found %d", v, mockAuditor.recordedEvents[i].eventId)
		}
	}
}

// One submitted auditable request with three separate credentials should result in
// three separate audit records, one for each user.
func TestMultiUserRequest(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:    true,
			EventDisabled:   make(map[uint32]bool),
			UserAllowlisted: make(map[datastore.UserInfo]bool),
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT", eventUsers: []string{"bill", "bob", "external:james"}}
	Submit(auditable)

	expectedEventRealUserIds := []adt.RealUserId{
		adt.RealUserId{Domain: "local", Username: "bill"},
		adt.RealUserId{Domain: "local", Username: "bob"},
		adt.RealUserId{Domain: "external", Username: "james"},
	}

	numExpected := len(expectedEventRealUserIds)
	numFound := len(mockAuditor.recordedEvents)
	if numExpected != numFound {
		t.Fatalf("Expected %d events, found %d", numExpected, numFound)
	}

	for i, expected := range expectedEventRealUserIds {
		found := mockAuditor.recordedEvents[i].queryAuditRecord.RealUserid
		if expected != found {
			t.Fatalf("Expected user %v but found user %v", expected, found)
		}
	}
}

func TestAuditDisabled(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:    false,
			EventDisabled:   make(map[uint32]bool),
			UserAllowlisted: make(map[datastore.UserInfo]bool),
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT"}
	Submit(auditable)

	numEvents := len(mockAuditor.recordedEvents)
	if numEvents != 0 {
		t.Fatalf("Expected 0 events, found %d", numEvents)
	}
}

func TestDisabledEvents(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:    true,
			EventDisabled:   map[uint32]bool{28678: true, 28679: true},
			UserAllowlisted: make(map[datastore.UserInfo]bool),
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT"}
	Submit(auditable)

	auditable.eventType = "INSERT"
	Submit(auditable)

	auditable.eventType = "UPDATE"
	Submit(auditable)

	auditable.eventType = "DELETE"
	Submit(auditable)

	auditable.eventType = "GARBAGE"
	Submit(auditable)

	expectedTypes := []uint32{28672, 28676, 28687}

	numEvents := len(mockAuditor.recordedEvents)
	if numEvents != len(expectedTypes) {
		t.Fatalf("Expected %d events, found %d", len(expectedTypes), numEvents)
	}

	for i, v := range expectedTypes {
		if v != mockAuditor.recordedEvents[i].eventId {
			t.Fatalf("Expected event id %d, found %d", v, mockAuditor.recordedEvents[i].eventId)
		}
	}
}

func TestAllowlistedUsers(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:  true,
			EventDisabled: make(map[uint32]bool),
			UserAllowlisted: map[datastore.UserInfo]bool{
				datastore.UserInfo{Name: "nina", Domain: "local"}: true,
				datastore.UserInfo{Name: "nick", Domain: "local"}: true,
				datastore.UserInfo{Name: "neil", Domain: "local"}: true},
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT", eventUsers: []string{"bill"}}
	Submit(auditable)

	auditable = &simpleAuditable{eventType: "SELECT", eventUsers: []string{"nina"}}
	Submit(auditable)

	auditable = &simpleAuditable{eventType: "SELECT", eventUsers: []string{}}
	Submit(auditable)

	auditable = &simpleAuditable{eventType: "SELECT", eventUsers: []string{"nick", "bob"}}
	Submit(auditable)

	auditable = &simpleAuditable{eventType: "SELECT", eventUsers: []string{"nina", "neil"}}
	Submit(auditable)

	expectedEventRealUserIds := []adt.RealUserId{
		adt.RealUserId{Domain: "local", Username: "bill"},
		adt.RealUserId{Domain: "", Username: ""},
		adt.RealUserId{Domain: "local", Username: "bob"},
	}

	numExpected := len(expectedEventRealUserIds)
	numFound := len(mockAuditor.recordedEvents)
	if numExpected != numFound {
		t.Fatalf("Expected %d events, found %d", numExpected, numFound)
	}

	for i, expected := range expectedEventRealUserIds {
		found := mockAuditor.recordedEvents[i].queryAuditRecord.RealUserid
		if expected != found {
			t.Fatalf("Expected user %v but found user %v", expected, found)
		}
	}
}

// MB-74936: each request cancelled by a DELETE is audited once per user who cancelled it
func TestRequestCancel(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:  true,
			EventDisabled: make(map[uint32]bool),
			UserAllowlisted: map[datastore.UserInfo]bool{
				datastore.UserInfo{Name: "nina", Domain: "local"}: true},
		},
		metricRegStub: accounting_stub.MetricRegistryStub{},
	}
	_AUDITOR = mockAuditor

	SubmitRequestCancel(&RequestCancelFields{
		Users:         []string{"local:alice", "external:bob", "local:nina"},
		RemoteAddress: "10.0.0.1:5000",
		UserAgent:     "QueryWorkbench",
		RequestId:     "delete-id",
		Cancelled:     []CancelledRequest{{"cancelled-1", "node1:8091"}, {"cancelled-2", "node2:8091"}},
	})

	expected := []struct {
		user    adt.RealUserId
		request string
		node    string
	}{
		{adt.RealUserId{Domain: "local", Username: "alice"}, "cancelled-1", "node1:8091"},
		{adt.RealUserId{Domain: "local", Username: "alice"}, "cancelled-2", "node2:8091"},
		{adt.RealUserId{Domain: "external", Username: "bob"}, "cancelled-1", "node1:8091"},
		{adt.RealUserId{Domain: "external", Username: "bob"}, "cancelled-2", "node2:8091"},
	}
	if len(mockAuditor.recordedEvents) != len(expected) {
		t.Fatalf("Expected %d events, found %d", len(expected), len(mockAuditor.recordedEvents))
	}
	for i, entry := range mockAuditor.recordedEvents {
		record, ok := entry.record.(*n1qlCancelRequestEvent)
		if entry.eventId != API_CANCEL_ACTIVE_REQUEST || !ok {
			t.Fatalf("Expected a cancel event %d, found event %d with record %v", API_CANCEL_ACTIVE_REQUEST,
				entry.eventId, entry.record)
		}
		if record.RealUserid != expected[i].user || record.CancelledRequestId != expected[i].request ||
			record.Node != expected[i].node {
			t.Errorf("Expected %+v, found %+v", expected[i], *record)
		}
		if record.RequestId != "delete-id" || record.UserAgent != "QueryWorkbench" || record.Timestamp == "" ||
			record.Remote == nil || record.Remote.Ip != "10.0.0.1" || record.Remote.Port != 5000 {
			t.Errorf("Unexpected record %+v", *record)
		}
	}

	// a REST DELETE has no statement: requestId is left out
	mockAuditor.recordedEvents = nil
	SubmitRequestCancel(&RequestCancelFields{Users: []string{"local:alice"},
		Cancelled: []CancelledRequest{{"cancelled-1", "node1:8091"}}})
	if len(mockAuditor.recordedEvents) != 1 {
		t.Fatalf("Expected 1 event for a REST cancel, found %d", len(mockAuditor.recordedEvents))
	}
	if b, err := json.Marshal(mockAuditor.recordedEvents[0].record); err != nil || strings.Contains(string(b), "requestId\"") {
		t.Errorf("REST cancel record %s (%v) has a requestId", b, err)
	}

	// not part of a cluster: the mandatory node is "local_node", as for statement events
	mockAuditor.recordedEvents = nil
	SubmitRequestCancel(&RequestCancelFields{Users: []string{"local:alice"},
		Cancelled: []CancelledRequest{{"cancelled-1", ""}}})
	if len(mockAuditor.recordedEvents) != 1 {
		t.Fatalf("Expected 1 event for a standalone cancel, found %d", len(mockAuditor.recordedEvents))
	}
	if record := mockAuditor.recordedEvents[0].record.(*n1qlCancelRequestEvent); record.Node != "local_node" {
		t.Errorf("Standalone cancel record has node %q, expected local_node", record.Node)
	}

	// nothing cancelled: nothing audited
	mockAuditor.recordedEvents = nil
	SubmitRequestCancel(&RequestCancelFields{Users: []string{"local:alice"}, RequestId: "delete-id"})
	if len(mockAuditor.recordedEvents) != 0 {
		t.Errorf("Expected no events when nothing was cancelled, found %d", len(mockAuditor.recordedEvents))
	}

	// disabled event: nothing audited
	mockAuditor.info.EventDisabled[API_CANCEL_ACTIVE_REQUEST] = true
	SubmitRequestCancel(&RequestCancelFields{Users: []string{"local:alice"},
		Cancelled: []CancelledRequest{{"cancelled-1", "node1:8091"}}})
	if len(mockAuditor.recordedEvents) != 0 {
		t.Errorf("Expected no events for a disabled event, found %d", len(mockAuditor.recordedEvents))
	}
}
