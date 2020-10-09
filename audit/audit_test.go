package audit

import (
	"testing"
	"time"

	adt "github.com/couchbase/goutils/go-cbaudit"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/accounting/stub"
	"github.com/couchbase/query/datastore"
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
	genericFields       adt.GenericFields
	remote              string
	local               string
	status              string
	statement           string
	queryContext        string
	eventId             string
	eventType           string
	eventUsers          []string
	userAgent           string
	eventNodeName       string
	eventNamedArgs      map[string]interface{}
	clientContextId     string
	eventPositionalArgs []interface{}
	isAdHoc             bool
	preparedId          string
	elapsedTime         time.Duration
	executionTime       time.Duration
	eventResultCount    int
	eventResultSize     int
	mutationCount       uint64
	sortCount           uint64
	eventErrorCount     int
	eventWarningCount   int
	txId                string
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

func (sa *simpleAuditable) ElapsedTime() time.Duration {
	return sa.elapsedTime
}

func (sa *simpleAuditable) ExecutionTime() time.Duration {
	return sa.executionTime
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
			UserWhitelisted: make(map[datastore.UserInfo]bool),
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
			UserWhitelisted: make(map[datastore.UserInfo]bool),
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
			UserWhitelisted: make(map[datastore.UserInfo]bool),
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
			UserWhitelisted: make(map[datastore.UserInfo]bool),
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

func TestWhitelistedUsers(t *testing.T) {
	mockAuditor := &mockAuditor{
		info: &datastore.AuditInfo{
			AuditEnabled:  true,
			EventDisabled: make(map[uint32]bool),
			UserWhitelisted: map[datastore.UserInfo]bool{
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
