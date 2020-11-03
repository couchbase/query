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
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	mcc "github.com/couchbase/gomemcached/client"
	adt "github.com/couchbase/goutils/go-cbaudit"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server"
)

// We keep counters for four things. All of these counters are available from the /admin/stats API.
//
// 1. The total number of potentially auditable requests sent to the query engine. (AUDIT_REQUESTS_TOTAL)
// 2. The number of potentially auditable requests that cause no audit action to be taken. (AUDIT_REQUESTS_FILTERED)
// 3. The total number of audit records sent to the server. (AUDIT_ACTIONS)
// 4. The total number of audit records sent to the server that failed. (AUDIT_ACTIONS_FAILED)
//
// Note that AUDIT_ACTIONS can be higher than (AUDIT_REQUESTS_TOTAL - AUDIT_REQUESTS_FILTERED) because
// some requests cause more than one audit record to be emitted. In particular, requests
// with multiple RBAC credentails cause one record to be emitted for each record.
// Also, AUDIT_ACTIONS can be lower than (AUDIT_REQUESTS_TOTAL - AUDIT_REQUESTS_FILTERED) because records
// in the output queue that have not been sent the server yet, are not counted against AUDIT_ACTIONS.

type Auditable interface {
	// Standard fields used for all audit records.
	EventGenericFields() adt.GenericFields

	// Address from which the request originated.
	EventRemoteAddress() string

	// Address which accepted the request.
	EventLocalAddress() string

	// success/fatal/stopped/etc.
	EventStatus() string

	// The N1QL statement executed.
	EventStatement() string

	// The query context used to formalize collections
	EventQueryContext() string

	// Transaction Id
	EventTxId() string

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
	EventNamedArgs() map[string]interface{}
	EventPositionalArgs() []interface{}

	// From client_context_id input parameter.
	// Useful for separating system-generated queries from user-issued queries.
	ClientContextId() string

	IsAdHoc() bool
	PreparedId() string

	// Metrics
	ElapsedTime() time.Duration
	ExecutionTime() time.Duration
	EventResultCount() int
	EventResultSize() int
	MutationCount() uint64
	SortCount() uint64
	EventErrorCount() int
	EventWarningCount() int
}

type ApiAuditFields struct {
	GenericFields  adt.GenericFields
	RemoteAddress  string
	LocalAddress   string
	EventTypeId    uint32
	Users          []string
	HttpMethod     string
	HttpResultCode int
	ErrorCode      int
	ErrorMessage   string

	Stat    string
	Name    string
	Request string
	Cluster string
	Node    string
	Values  interface{}
	Body    interface{}
}

// An auditor is a component that can accept an audit record for processing.
// We create a formal interface, so we can have two Auditors: the regular one that
// talks to the audit daemon, and a mock that just stores audit records for testing.
// The mock is over in the test file.
type Auditor interface {
	auditInfo() *datastore.AuditInfo
	setAuditInfo(info *datastore.AuditInfo)

	submit(entry auditQueueEntry)
}

type standardAuditor struct {
	auditService     *adt.AuditSvc
	auditRecordQueue chan auditQueueEntry

	auditInfoLock sync.RWMutex // get read or write lock before modifying reference to
	info          *datastore.AuditInfo
}

type auditQueueEntry struct {
	eventId          uint32
	isQueryType      bool
	queryAuditRecord *n1qlAuditEvent
	apiAuditRecord   *n1qlAuditApiRequestEvent
}

func (sa *standardAuditor) auditInfo() *datastore.AuditInfo {
	sa.auditInfoLock.RLock()
	ret := sa.info
	sa.auditInfoLock.RUnlock()
	return ret
}

func (sa *standardAuditor) setAuditInfo(info *datastore.AuditInfo) {
	sa.auditInfoLock.Lock()
	sa.info = info
	sa.auditInfoLock.Unlock()
}

func eventIsDisabled(au *datastore.AuditInfo, eventId uint32) bool {
	// No real event number?
	if eventId == API_DO_NOT_AUDIT {
		return true
	}

	return au.EventDisabled[eventId]
}

func (sa *standardAuditor) submit(entry auditQueueEntry) {
	// Put the audit entry on the queue for processing.
	// If the queue is full, block until it clears.
	sa.auditRecordQueue <- entry
}

var _AUDITOR Auditor

// numServicers is the number of worker threads we expect to see
// accessing the audit functionality. It is NOT the number of worker threads
// the audit system itself has.
func StartAuditService(server string, numServicers int) {
	// No support for auditing?
	// Set auditor to NIL for no auditing work at all.
	if !VERSION_SUPPORTS_AUDIT {
		_AUDITOR = nil
		return
	}

	var err error
	service, err := adt.NewAuditSvc(server)
	if err != nil {
		logging.Errorf("Audit service not started: %v", err)
		return
	}
	auditor := &standardAuditor{}
	auditor.auditService = service

	ds := datastore.GetDatastore()
	if ds == nil {
		logging.Errorf("Audit service not started: no data store available")
		return
	}

	// Fetch initial audit settings now. These will be refreshed periodically
	// by the auditSettings worker.
	auditInfo, err := ds.AuditInfo()
	if err != nil {
		logging.Errorf("Audit service not started: audit settings not available: %v", err)
		return
	}
	auditor.info = auditInfo

	// The queue should be of finite length, but ample,
	// to smooth out any bumps in service. The servicers
	// should be able to leave an audit entry and continue
	// unimpeded, with high probability.
	auditor.auditRecordQueue = make(chan auditQueueEntry, numServicers*25)

	for i := 1; i <= runtime.NumCPU(); i++ {
		go auditWorker(auditor, i)
	}

	go auditSettingsWorker(auditor, 1)

	_AUDITOR = auditor
}

func auditSettingsWorker(auditor *standardAuditor, num int) {
	// If this audit worker panics, start up a replacement.
	defer func() {
		r := recover()
		if r != nil {
			logging.Errorf("Audit settings worker %d: Panic: %v. Starting a replacement.", num, r)
			go auditSettingsWorker(auditor, num+1)
		}
	}()
	logging.Infof("Starting audit settings worker %d.", num)

	auditInfo := auditor.auditInfo()
	curUid := auditInfo.Uid
	ds := datastore.GetDatastore()

	for {
		time.Sleep(time.Second * 1)

		f := func(uid string) error {
			if uid == curUid {
				// No change. Do nothing.
				return nil
			}
			auditInfo, err := ds.AuditInfo()
			if err != nil {
				return fmt.Errorf("Audit update handler function %d: Unable to get audit settings: %v", num, err)
			}
			if curUid != auditInfo.Uid {
				logging.Infof("Audit update handler function %d: Got updated audit settings: %+v", num, stringifyauditInfo(*auditInfo))
				change := n1qlConfigurationChangeEvent{
					Timestamp:  time.Now().Format("2006-01-02T15:04:05.000Z07:00"),
					RealUserid: adt.RealUserId{Domain: "internal", Username: "couchbase"},
					Uuid:       auditInfo.Uid,
				}

				e := auditor.auditService.Write(28703, &change)
				if e != nil {
					return fmt.Errorf("Audit settings worker %d: Unable to send configuration change message: %v", num, err)
				}
				logging.Infof("Audit update handler function %d: wrote config change message.", num)

				curUid = auditInfo.Uid
				auditor.setAuditInfo(auditInfo)
			}
			return nil
		}

		logging.Infof("Starting audit update stream")
		err := ds.ProcessAuditUpdateStream(f)
		if err != nil {
			logging.Errorf("Audit update stream failed: %v", err)
		} else {
			logging.Infof("Audit update stream terminated normally.")
		}
	}
}

func auditWorker(auditor *standardAuditor, num int) {
	// If this audit worker panics, start up a replacement.
	defer func() {
		r := recover()
		if r != nil {
			logging.Errorf("Audit worker %d: Panic: %v. Starting a replacement.", num, r)
			go auditWorker(auditor, num+1)
		}
	}()
	logging.Infof("Starting audit worker %d", num)

	var client *mcc.Client // The audit worker holds on to one client.
	var err error

	// Main processing loop
	for {
		entry := <-auditor.auditRecordQueue

		// Dispose of an unhealthy client.
		if client != nil && !client.IsHealthy() {
			err = client.Close()
			if err != nil {
				logging.Warnf("Audit worker %d: closing unhealthy client produced error: %v", err)
			}
			client = nil
		}

		// Refresh the client if necessary.
		for client == nil {
			client, err = auditor.auditService.GetNonPoolClient()
			if err != nil {
				logging.Errorf("Audit worker %d: unable to get connection: %v. Will sleep and retry.", num, err)
				time.Sleep(time.Second * 2)
			}
		}

		// Send the audit record using the client.
		accounting.UpdateCounter(accounting.AUDIT_ACTIONS)
		if entry.isQueryType {
			err = auditor.auditService.WriteUsingNonPoolClient(client, entry.eventId, *entry.queryAuditRecord)
			if err != nil {
				accounting.UpdateCounter(accounting.AUDIT_ACTIONS_FAILED)
				logging.Errorf("Audit worker %d: unable to send audit record %+v to audit demon: %v", num, stringifyQueryAR(*entry.queryAuditRecord), err)
			}
		} else {
			err = auditor.auditService.WriteUsingNonPoolClient(client, entry.eventId, *entry.apiAuditRecord)
			if err != nil {
				accounting.UpdateCounter(accounting.AUDIT_ACTIONS_FAILED)
				logging.Errorf("Audit worker %d: unable to send audit record %+v to audit demon: %v", num, stringifyAPIAR(*entry.apiAuditRecord), err)
			}
		}
	}
}

func stringifyauditInfo(entry datastore.AuditInfo) string {
	str := fmt.Sprintf("AuditEnabled: %v ", entry.AuditEnabled)
	str += fmt.Sprintf("EventDisabled: %v ", entry.EventDisabled)
	str += fmt.Sprintf("UserWhitelisted: <ud>%v</ud> ", entry.UserWhitelisted)
	str += fmt.Sprintf("Uid: <ud>%v</ud> ", entry.Uid)
	return str
}

func stringifyQueryAR(entry n1qlAuditEvent) string {

	str := fmt.Sprintf("RequestID: <ud>%v</ud> ", entry.RequestId)
	str += fmt.Sprintf("Statement: <ud>%v</ud> ", entry.Statement)
	str += fmt.Sprintf("QueryContext: <ud>%v</ud> ", entry.QueryContext)
	str += fmt.Sprintf("NamedArgs: %v ", entry.NamedArgs)
	str += fmt.Sprintf("PositionalArgs: %v ", entry.PositionalArgs)
	str += fmt.Sprintf("ClientContextId: <ud>%v</ud> ", entry.ClientContextId)
	str += fmt.Sprintf("TxId: <ud>%v</ud> ", entry.TxId)
	str += fmt.Sprintf("IsAdHoc: %v ", entry.IsAdHoc)
	str += fmt.Sprintf("PreparedId: %v ", entry.PreparedId)
	str += fmt.Sprintf("UserAgent: %v ", entry.UserAgent)
	str += fmt.Sprintf("Node: %v ", entry.Node)
	str += fmt.Sprintf("Status: %v ", entry.Status)
	str += fmt.Sprintf("Metrics: %v ", entry.Metrics)
	return str

}

func stringifyAPIAR(entry n1qlAuditApiRequestEvent) string {

	str := fmt.Sprintf("HttpMethod: %v ", entry.HttpMethod)
	str += fmt.Sprintf("HttpResultCode: %v ", entry.HttpResultCode)
	str += fmt.Sprintf("ErrorCode: %v ", entry.ErrorCode)
	str += fmt.Sprintf("ErrorMessage: %v ", entry.ErrorMessage)
	str += fmt.Sprintf("Stat: %v ", entry.Stat)
	str += fmt.Sprintf("Name: <ud>%v</ud> ", entry.Name)
	str += fmt.Sprintf("Request: <ud>%v</ud> ", entry.Request)
	str += fmt.Sprintf("Cluster: %v ", entry.Cluster)
	str += fmt.Sprintf("Node: %v ", entry.Node)
	str += fmt.Sprintf("Values: <ud>%v</ud> ", entry.Values)
	str += fmt.Sprintf("Body: <ud>%v</ud> ", entry.Body)

	return str
}

// Event types are described in /query/etc/audit_descriptor.json
var _EVENT_TYPE_MAP = map[string]uint32{
	"SELECT":                    28672,
	"EXPLAIN":                   28673,
	"PREPARE":                   28674,
	"INFER":                     28675,
	"INSERT":                    28676,
	"UPSERT":                    28677,
	"DELETE":                    28678,
	"UPDATE":                    28679,
	"MERGE":                     28680,
	"CREATE_INDEX":              28681,
	"DROP_INDEX":                28682,
	"ALTER_INDEX":               28683,
	"BUILD_INDEX":               28684,
	"GRANT_ROLE":                28685,
	"REVOKE_ROLE":               28686,
	"CREATE_PRIMARY_INDEX":      28688,
	"CREATE_FUNCTION":           28706,
	"DROP_FUNCTION":             28707,
	"EXECUTE_FUNCTION":          28708,
	"CREATE_SCOPE":              28713,
	"DROP_SCOPE":                28714,
	"CREATE_COLLECTION":         28715,
	"DROP_COLLECTION":           28716,
	"FLUSH_COLLECTION":          28717,
	"UPDATE_STATISTICS":         28718,
	"ADVISE":                    28719,
	"START_TRANSACTION":         28720,
	"COMMIT":                    28721,
	"ROLLBACK":                  28722,
	"ROLLBACK_SAVEPOINT":        28723,
	"SET_TRANSACTION_ISOLATION": 28724,
	"SAVEPOINT":                 28725,
}

func Submit(event Auditable) {
	if _AUDITOR == nil {
		return // Nothing configured. Nothing to be done.
	}
	accounting.UpdateCounter(accounting.AUDIT_REQUESTS_TOTAL)

	auditInfo := _AUDITOR.auditInfo()
	if auditInfo == nil {
		logging.Errorf("Unable to audit. Audit specification is not available.")
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	if !auditInfo.AuditEnabled {
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	eventType := event.EventType()
	eventTypeId := _EVENT_TYPE_MAP[eventType]

	// Handle unrecognized events.
	if eventTypeId == 0 {
		eventTypeId = 28687
	}

	if eventIsDisabled(auditInfo, eventTypeId) {
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	// We build the audit record from the request in the main thread
	// because the request will be destroyed soon after the call to Submit(),
	// and we don't want to cause a race condition.
	auditEntries := buildAuditEntries(eventTypeId, event, auditInfo)
	submitAuditEntries(auditEntries)
}

func submitAuditEntries(entries []auditQueueEntry) {
	for _, entry := range entries {
		_AUDITOR.submit(entry)
	}
}

// the ...INDEX... events are there for completeness as they are used internally and not currently audited
const (
	API_DO_NOT_AUDIT                     = 0
	API_ADMIN_STATS                      = 28689
	API_ADMIN_VITALS                     = 28690
	API_ADMIN_PREPAREDS                  = 28691
	API_ADMIN_ACTIVE_REQUESTS            = 28692
	API_ADMIN_INDEXES_PREPAREDS          = 28693
	API_ADMIN_INDEXES_ACTIVE_REQUESTS    = 28694
	API_ADMIN_INDEXES_COMPLETED_REQUESTS = 28695
	API_ADMIN_PING                       = 28697
	API_ADMIN_CONFIG                     = 28698
	API_ADMIN_SSL_CERT                   = 28699
	API_ADMIN_SETTINGS                   = 28700
	API_ADMIN_CLUSTERS                   = 28701
	API_ADMIN_COMPLETED_REQUESTS         = 28702
	API_ADMIN_FUNCTIONS                  = 28704
	API_ADMIN_INDEXES_FUNCTIONS          = 28705
	API_ADMIN_TASKS                      = 28709
	API_ADMIN_INDEXES_TASKS              = 28710
	API_ADMIN_DICTIONARY                 = 28711
	API_ADMIN_INDEXES_DICTIONARY         = 28712
	API_ADMIN_TRANSACTIONS               = 28726
	API_ADMIN_INDEXES_TRANSACTIONS       = 28727
)

func SubmitApiRequest(event *ApiAuditFields) {
	if _AUDITOR == nil {
		return // Nothing configured. Nothing to be done.
	}
	accounting.UpdateCounter(accounting.AUDIT_REQUESTS_TOTAL)

	auditInfo := _AUDITOR.auditInfo()
	if auditInfo == nil {
		logging.Errorf("Unable to audit. Audit specification is not available.")
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	if !auditInfo.AuditEnabled {
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	eventTypeId := event.EventTypeId

	if eventIsDisabled(auditInfo, eventTypeId) {
		accounting.UpdateCounter(accounting.AUDIT_REQUESTS_FILTERED)
		return
	}

	// We build the audit entry from the request in the main thread
	// because the request will be destroyed soon after the call to SubmitApiRequest(),
	// and we don't want to cause a race condition.
	auditEntries := buildApiRequestAuditEntries(eventTypeId, event, auditInfo)
	submitAuditEntries(auditEntries)
}

func parseAddress(addr string) *addressFields {
	if addr == "" {
		return nil
	}
	host, port := server.HostNameandPort(addr)
	p, err := strconv.Atoi(port)
	if err != nil {
		logging.Errorf("Auditing: unable to parse port %s of address %s", port, addr)
		return &addressFields{Ip: addr}
	}
	return &addressFields{Ip: host, Port: p}
}

// Returns a list of audit entries, because each user credential submitted as part of
// the requests generates a separate audit record.
func buildAuditEntries(eventTypeId uint32, event Auditable, auditInfo *datastore.AuditInfo) []auditQueueEntry {
	// Grab the data from the event, so we don't query the duplicated data
	// multiple times.
	genericFields := event.EventGenericFields()
	remote := parseAddress(event.EventRemoteAddress())
	local := parseAddress(event.EventLocalAddress())
	requestId := event.EventId()
	statement := event.EventStatement()
	queryContext := event.EventQueryContext()
	namedArgs := event.EventNamedArgs()
	positionalArgs := event.EventPositionalArgs()
	clientContextId := event.ClientContextId()
	txId := event.EventTxId()
	isAdHoc := event.IsAdHoc()
	preparedId := event.PreparedId()
	userAgent := event.UserAgent()
	node := event.EventNodeName()
	status := event.EventStatus()
	metrics := &n1qlMetrics{
		ElapsedTime:   fmt.Sprintf("%v", event.ElapsedTime()),
		ExecutionTime: fmt.Sprintf("%v", event.ExecutionTime()),
		ResultCount:   event.EventResultCount(),
		ResultSize:    event.EventResultSize(),
		MutationCount: event.MutationCount(),
		SortCount:     event.SortCount(),
		ErrorCount:    event.EventErrorCount(),
		WarningCount:  event.EventWarningCount(),
	}

	// No credentials at all? Generate one record.
	usernames := event.EventUsers()
	if len(usernames) == 0 {
		record := &n1qlAuditEvent{
			GenericFields:   genericFields,
			Remote:          remote,
			Local:           local,
			RequestId:       requestId,
			Statement:       statement,
			QueryContext:    queryContext,
			NamedArgs:       namedArgs,
			PositionalArgs:  positionalArgs,
			ClientContextId: clientContextId,
			TxId:            txId,
			IsAdHoc:         isAdHoc,
			PreparedId:      preparedId,
			UserAgent:       userAgent,
			Node:            node,
			Status:          status,
			Metrics:         metrics,
		}
		entry := auditQueueEntry{
			eventId:          eventTypeId,
			isQueryType:      true,
			queryAuditRecord: record,
		}
		return []auditQueueEntry{entry}
	}

	// Figure out which users to generate events for.
	auditableUsers := make([]datastore.UserInfo, 0, len(usernames))
	for _, username := range usernames {
		user := userInfoFromUsername(username)
		if !auditInfo.UserWhitelisted[user] {
			auditableUsers = append(auditableUsers, user)
		}
	}

	// Generate one record per user.
	entries := make([]auditQueueEntry, len(auditableUsers))
	for i, user := range auditableUsers {
		record := &n1qlAuditEvent{
			GenericFields:   genericFields,
			Remote:          remote,
			Local:           local,
			RequestId:       requestId,
			Statement:       statement,
			QueryContext:    queryContext,
			NamedArgs:       namedArgs,
			PositionalArgs:  positionalArgs,
			ClientContextId: clientContextId,
			TxId:            txId,
			IsAdHoc:         isAdHoc,
			PreparedId:      preparedId,
			UserAgent:       userAgent,
			Node:            node,
			Status:          status,
			Metrics:         metrics,
		}
		record.GenericFields.RealUserid.Domain = user.Domain
		record.GenericFields.RealUserid.Username = user.Name

		entries[i] = auditQueueEntry{
			eventId:          eventTypeId,
			isQueryType:      true,
			queryAuditRecord: record,
		}
	}
	return entries
}

func userInfoFromUsername(user string) datastore.UserInfo {
	domain := "local"
	userName := user
	// Handle non-local users, e.g. "external:dtrump"
	if strings.Contains(user, ":") {
		parts := strings.SplitN(user, ":", 2)
		domain = parts[0]
		userName = parts[1]
	}
	return datastore.UserInfo{Name: userName, Domain: domain}
}

// Returns a list of audit entries, because each user credential submitted as part of
// the requests generates a separate audit record.
func buildApiRequestAuditEntries(eventTypeId uint32, event *ApiAuditFields, auditInfo *datastore.AuditInfo) []auditQueueEntry {

	remote := parseAddress(event.RemoteAddress)
	local := parseAddress(event.LocalAddress)
	// No credentials at all? Generate one record.
	usernames := event.Users
	if len(usernames) == 0 {
		record := &n1qlAuditApiRequestEvent{
			GenericFields:  event.GenericFields,
			Remote:         remote,
			Local:          local,
			HttpMethod:     event.HttpMethod,
			HttpResultCode: event.HttpResultCode,
			ErrorCode:      event.ErrorCode,
			ErrorMessage:   event.ErrorMessage,
			Stat:           event.Stat,
			Name:           event.Name,
			Request:        event.Request,
			Values:         event.Values,
			Cluster:        event.Cluster,
			Node:           event.Node,
			Body:           event.Body,
		}
		entry := auditQueueEntry{
			eventId:        eventTypeId,
			isQueryType:    false,
			apiAuditRecord: record,
		}
		return []auditQueueEntry{entry}
	}

	// Figure out which users to generate events for.
	auditableUsers := make([]datastore.UserInfo, 0, len(usernames))
	for _, username := range usernames {
		user := userInfoFromUsername(username)
		if !auditInfo.UserWhitelisted[user] {
			auditableUsers = append(auditableUsers, user)
		}
	}

	// Generate one entry per user.
	entries := make([]auditQueueEntry, len(auditableUsers))
	for i, user := range auditableUsers {
		record := &n1qlAuditApiRequestEvent{
			GenericFields:  event.GenericFields,
			Remote:         remote,
			Local:          local,
			HttpMethod:     event.HttpMethod,
			HttpResultCode: event.HttpResultCode,
			ErrorCode:      event.ErrorCode,
			ErrorMessage:   event.ErrorMessage,
			Stat:           event.Stat,
			Name:           event.Name,
			Request:        event.Request,
			Values:         event.Values,
			Cluster:        event.Cluster,
			Node:           event.Node,
			Body:           event.Body,
		}
		record.GenericFields.RealUserid.Domain = user.Domain
		record.GenericFields.RealUserid.Username = user.Name

		entries[i] = auditQueueEntry{
			eventId:        eventTypeId,
			isQueryType:    false,
			apiAuditRecord: record,
		}
	}
	return entries
}

// If possible, use whatever field names are used elsewhere in the N1QL system.
// Follow whatever naming scheme (under_scores/camelCase/What.Ever) is standard for each field.
// If no standard exists for the field, use camelCase.
type n1qlAuditEvent struct {
	adt.GenericFields
	Remote *addressFields `json:"remote,omitempty"`
	Local  *addressFields `json:"local,omitempty"`

	RequestId       string                 `json:"requestId"`
	Statement       string                 `json:"statement"`
	QueryContext    string                 `json:"queryContext,omitempty"`
	NamedArgs       map[string]interface{} `json:"namedArgs,omitempty"`
	PositionalArgs  []interface{}          `json:"positionalArgs,omitempty"`
	ClientContextId string                 `json:"clientContextId,omitempty"`
	TxId            string                 `json:"txId,omitempty"`

	IsAdHoc    bool   `json:"isAdHoc"`
	PreparedId string `json:"preparedId,omitempty"`
	UserAgent  string `json:"userAgent"`
	Node       string `json:"node"`

	Status string `json:"status"`

	Metrics *n1qlMetrics `json:"metrics"`
}

type addressFields struct {
	Ip   string `json:"ip"`
	Port int    `json:"port,omitempty"`
}

type n1qlMetrics struct {
	ElapsedTime   string `json:"elapsedTime"`
	ExecutionTime string `json:"executionTime"`
	ResultCount   int    `json:"resultCount"`
	ResultSize    int    `json:"resultSize"`
	MutationCount uint64 `json:"mutationCount,omitempty"`
	SortCount     uint64 `json:"sortCount,omitempty"`
	ErrorCount    int    `json:"errorCount,omitempty"`
	WarningCount  int    `json:"warningCount,omitempty"`
}

type n1qlAuditApiRequestEvent struct {
	adt.GenericFields
	Remote *addressFields `json:"remote,omitempty"`
	Local  *addressFields `json:"local,omitempty"`

	HttpMethod     string `json:"httpMethod"`
	HttpResultCode int    `json:"httpResultCode"`
	ErrorCode      int    `json:"errorCode,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`

	Stat    string      `json:"stat,omitempty"`
	Name    string      `json:"name,omitempty"`
	Request string      `json:"request,omitempty"`
	Cluster string      `json:"cluster,omitempty"`
	Node    string      `json:"node,omitempty"`
	Values  interface{} `json:"values,omitempty"`
	Body    interface{} `json:"body,omitempty"`
}

type n1qlConfigurationChangeEvent struct {
	Timestamp  string         `json:"timestamp"`
	RealUserid adt.RealUserId `json:"real_userid"`
	Uuid       string         `json:"uuid"`
}
