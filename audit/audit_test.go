package audit

import (
	"testing"
	"time"

	adt "github.com/couchbase/goutils/go-cbaudit"
)

type auditSubmission struct {
	eventId uint32
	event   *n1qlAuditEvent
}

// An auditor the just records the audit events that would be sent to the audit daemon,
// nothing more.
type mockAuditor struct {
	recordedEvents   []auditSubmission
	disabledAudit    bool
	whitelistedUsers []string
	disabledEvents   []uint32
}

func (ma *mockAuditor) doAudit() bool {
	return !ma.disabledAudit
}

func (ma *mockAuditor) submitInline() bool {
	return true // To simplify testing.
}

func (ma *mockAuditor) submit(eventId uint32, event *n1qlAuditEvent) error {
	submission := auditSubmission{eventId: eventId, event: event}
	ma.recordedEvents = append(ma.recordedEvents, submission)
	return nil
}

func (ma *mockAuditor) userIsWhitelisted(user string) bool {
	for _, u := range ma.whitelistedUsers {
		if user == u {
			return true
		}
	}
	return false
}

func (ma *mockAuditor) eventIsDisabled(eventId uint32) bool {
	for _, e := range ma.disabledEvents {
		if eventId == e {
			return true
		}
	}
	return false
}

// A fixed structure that implements the Auditable interface
type simpleAuditable struct {
	genericFields       adt.GenericFields
	status              string
	statement           string
	eventId             string
	eventType           string
	eventUsers          []string
	userAgent           string
	eventNodeName       string
	eventNamedArgs      map[string]string
	clientContextId     string
	eventPositionalArgs []string
	isAdHoc             bool
	elapsedTime         time.Duration
	executionTime       time.Duration
	eventResultCount    int
	eventResultSize     int
	mutationCount       uint64
	sortCount           uint64
	eventErrorCount     int
	eventWarningCount   int
}

func (sa *simpleAuditable) EventGenericFields() adt.GenericFields {
	return sa.genericFields
}

func (sa *simpleAuditable) EventStatus() string {
	return sa.status
}

func (sa *simpleAuditable) Statement() string {
	return sa.statement
}

func (sa *simpleAuditable) EventId() string {
	return sa.eventId
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

func (sa *simpleAuditable) EventNamedArgs() map[string]string {
	return sa.eventNamedArgs
}

func (sa *simpleAuditable) EventPositionalArgs() []string {
	return sa.eventPositionalArgs
}

func (sa *simpleAuditable) ClientContextId() string {
	return sa.clientContextId
}

func (sa *simpleAuditable) IsAdHoc() bool {
	return sa.isAdHoc
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
	mockAuditor := &mockAuditor{}
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
	mockAuditor := &mockAuditor{}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT", eventUsers: []string{"bill", "bob", "external:james"}}
	Submit(auditable)

	expectedEventRealUserIds := []adt.RealUserId{
		adt.RealUserId{Source: "local", Username: "bill"},
		adt.RealUserId{Source: "local", Username: "bob"},
		adt.RealUserId{Source: "external", Username: "james"},
	}

	numExpected := len(expectedEventRealUserIds)
	numFound := len(mockAuditor.recordedEvents)
	if numExpected != numFound {
		t.Fatalf("Expected %d events, found %d", numExpected, numFound)
	}

	for i, expected := range expectedEventRealUserIds {
		found := mockAuditor.recordedEvents[i].event.RealUserid
		if expected != found {
			t.Fatalf("Expected user %v but found user %v", expected, found)
		}
	}
}

func TestAuditDisabled(t *testing.T) {
	mockAuditor := &mockAuditor{disabledAudit: true}
	_AUDITOR = mockAuditor

	auditable := &simpleAuditable{eventType: "SELECT"}
	Submit(auditable)

	numEvents := len(mockAuditor.recordedEvents)
	if numEvents != 0 {
		t.Fatalf("Expected 0 events, found %d", numEvents)
	}
}

func TestDisabledEvents(t *testing.T) {
	mockAuditor := &mockAuditor{disabledEvents: []uint32{28678, 28679}}
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
	mockAuditor := &mockAuditor{whitelistedUsers: []string{"nina", "nick", "neil"}}
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
		adt.RealUserId{Source: "local", Username: "bill"},
		adt.RealUserId{Source: "", Username: ""},
		adt.RealUserId{Source: "local", Username: "bob"},
	}

	numExpected := len(expectedEventRealUserIds)
	numFound := len(mockAuditor.recordedEvents)
	if numExpected != numFound {
		t.Fatalf("Expected %d events, found %d", numExpected, numFound)
	}

	for i, expected := range expectedEventRealUserIds {
		found := mockAuditor.recordedEvents[i].event.RealUserid
		if expected != found {
			t.Fatalf("Expected user %v but found user %v", expected, found)
		}
	}
}
