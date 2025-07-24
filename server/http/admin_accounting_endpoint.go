//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"bytes"
	"encoding/base64"
	go_errors "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/cbauth"
	json "github.com/couchbase/go_json"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	dictionary "github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/functions"
	functionsBridge "github.com/couchbase/query/functions/bridge"
	functionsResolver "github.com/couchbase/query/functions/resolver"
	functionsStorage "github.com/couchbase/query/functions/storage"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/memory"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/sequences"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http/router"
	"github.com/couchbase/query/system"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
	"github.com/golang/snappy"
)

const (
	accountingPrefix   = adminPrefix + "/stats"
	vitalsPrefix       = adminPrefix + "/vitals"
	preparedsPrefix    = adminPrefix + "/prepareds"
	requestsPrefix     = adminPrefix + "/active_requests"
	completedsPrefix   = adminPrefix + "/completed_requests"
	functionsPrefix    = adminPrefix + "/functions_cache"
	dictionaryPrefix   = adminPrefix + "/dictionary_cache"
	tasksPrefix        = adminPrefix + "/tasks_cache"
	indexesPrefix      = adminPrefix + "/indexes"
	expvarsRoute       = "/debug/vars"
	prometheusLow      = "/_prometheusMetrics"
	prometheusHigh     = "/_prometheusMetricsHigh"
	transactionsPrefix = adminPrefix + "/transactions"
	backupPrefix       = "/api/v1"
	sequencesPrefix    = adminPrefix + "/sequences_cache"
)

func expvarsHandler(w http.ResponseWriter, req *http.Request) {
	// Do not audit directly.
	// Will be handled and audited by /admin/stats auditing.
	http.Redirect(w, req, accountingPrefix, http.StatusFound)
}

func (this *HttpEndpoint) registerAccountingHandlers() {
	statsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStats, false)
	}
	statHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStat, false)
	}
	notFoundHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNotFound, false)
	}
	vitalsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doVitals, false)
	}
	preparedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepared, false)
	}
	preparedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepareds, false)
	}
	requestsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequests, false)
	}
	requestHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequest, false)
	}
	completedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequests, false)
	}
	completedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequest, false)
	}
	preparedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPreparedIndex, false)
	}
	requestIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doRequestIndex, false)
	}
	completedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedIndex, false)
	}
	functionsIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctionsIndex, false)
	}
	functionHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunction, false)
	}
	functionsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctions, false)
	}
	dictionaryIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionaryIndex, false)
	}
	dictionaryEntryHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionaryEntry, false)
	}
	dictionaryHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionary, false)
	}
	tasksIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTasksIndex, false)
	}
	taskHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTask, false)
	}

	completedHistoryHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequestHistory, false)
	}
	completedsHistoryIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedHistoryIndex, false)
	}

	tasksHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTasks, false)
	}

	prometheusLowHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrometheusLow, false)
	}

	transactionsIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransactionsIndex, false)
	}
	transactionHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransaction, false)
	}
	transactionsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransactions, false)
	}
	globalBackupHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doGlobalBackup, true)
	}
	bucketBackupHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doBucketBackup, true)
	}
	forceGCHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doForceGC, false)
	}
	manualFFDCHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doManualFFDC, false)
	}
	logHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doLog, false)
	}
	sequenceIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSequenceIndex, false)
	}
	sequenceHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doSequence, false)
	}
	migrationHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doMigration, false)
	}
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:                      {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}":          {handler: statHandler, methods: []string{"GET", "DELETE"}},
		vitalsPrefix:                          {handler: vitalsHandler, methods: []string{"GET"}},
		preparedsPrefix:                       {handler: preparedsHandler, methods: []string{"GET", "POST"}},
		preparedsPrefix + "/{name}":           {handler: preparedHandler, methods: []string{"GET", "POST", "DELETE", "PUT"}},
		requestsPrefix:                        {handler: requestsHandler, methods: []string{"GET", "POST"}},
		requestsPrefix + "/{request}":         {handler: requestHandler, methods: []string{"GET", "POST", "DELETE"}},
		completedsPrefix:                      {handler: completedsHandler, methods: []string{"GET", "POST"}},
		completedsPrefix + "/{request}":       {handler: completedHandler, methods: []string{"GET", "POST", "DELETE"}},
		functionsPrefix:                       {handler: functionsHandler, methods: []string{"GET"}},
		functionsPrefix + "/{name}":           {handler: functionHandler, methods: []string{"GET", "POST", "DELETE"}},
		dictionaryPrefix:                      {handler: dictionaryHandler, methods: []string{"GET"}},
		dictionaryPrefix + "/{name}":          {handler: dictionaryEntryHandler, methods: []string{"GET", "POST", "DELETE"}},
		tasksPrefix:                           {handler: tasksHandler, methods: []string{"GET"}},
		tasksPrefix + "/{name}":               {handler: taskHandler, methods: []string{"GET", "POST", "DELETE"}},
		transactionsPrefix:                    {handler: transactionsHandler, methods: []string{"GET"}},
		transactionsPrefix + "/{txid}":        {handler: transactionHandler, methods: []string{"GET", "POST", "DELETE"}},
		indexesPrefix + "/prepareds":          {handler: preparedIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/active_requests":    {handler: requestIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/completed_requests": {handler: completedIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/functions_cache":    {handler: functionsIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/dictionary_cache":   {handler: dictionaryIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/tasks_cache":        {handler: tasksIndexHandler, methods: []string{"GET"}},
		prometheusLow:                         {handler: prometheusLowHandler, methods: []string{"GET"}},
		indexesPrefix + "/transactions":       {handler: transactionsIndexHandler, methods: []string{"GET"}},

		completedsPrefix + "_history/{request}":       {handler: completedHistoryHandler, methods: []string{"GET"}},
		indexesPrefix + "/completed_requests_history": {handler: completedsHistoryIndexHandler, methods: []string{"GET"}},

		backupPrefix + "/backup":                 {handler: globalBackupHandler, methods: []string{"GET", "POST"}},
		backupPrefix + "/bucket/{bucket}/backup": {handler: bucketBackupHandler, methods: []string{"GET", "POST"}},

		adminPrefix + "/gc":                {handler: forceGCHandler, methods: []string{"GET", "POST"}},
		adminPrefix + "/ffdc":              {handler: manualFFDCHandler, methods: []string{"POST"}},
		adminPrefix + "/log/{file}":        {handler: logHandler, methods: []string{"GET"}},
		adminPrefix + "/log/stream/{file}": {handler: logHandler, methods: []string{"GET"}},
		indexesPrefix + "/sequences":       {handler: sequenceIndexHandler, methods: []string{"GET"}},
		sequencesPrefix + "/{name}":        {handler: sequenceHandler, methods: []string{"GET"}},
		adminPrefix + "/migration":         {handler: migrationHandler, methods: []string{"DELETE"}},
	}

	for route, h := range routeMap {
		this.router.Map(route, h.handler, h.methods...)
	}

	// prometheus is a special case, as it may be handled by the tenant code
	if !tenant.IsServerless() {
		prometheusHighHandler := func(w http.ResponseWriter, req *http.Request) {
			this.wrapAPI(w, req, doEmpty, false)
		}
		this.router.Map(prometheusHigh, prometheusHighHandler, "GET")
	}

	this.router.Map(expvarsRoute, expvarsHandler, "GET")

	this.router.SetNotFoundHandler(notFoundHandler)
}

func doStats(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:stats", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		acctStore := endpoint.server.AccountingStore()
		reg := acctStore.MetricRegistry()
		af.EventTypeId = audit.API_DO_NOT_AUDIT
		stats := make(map[string]interface{})
		for name, metric := range reg.Counters() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Gauges() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Timers() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Meters() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name, metric := range reg.Histograms() {
			addMetricData(name, stats, getMetricData(metric))
		}
		for name := range localData {
			addMetricData(name, stats, getLocalData(endpoint.server, name))
		}
		return stats, nil
	default:
		af.EventTypeId = audit.API_ADMIN_STATS
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func addMetricData(name string, stats map[string]interface{}, metrics map[string]interface{}) {
	for metric_type, metric_value := range metrics {

		// MB-20521 avoid buffers that are reused
		stats[name+"."+metric_type] = metric_value
	}
}

func doStat(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "stat")
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

	af.EventTypeId = audit.API_ADMIN_STATS
	af.Stat = name

	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:stats", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		af.EventTypeId = audit.API_DO_NOT_AUDIT
		if isLocal(name) {
			return getLocalData(endpoint.server, name), nil
		} else {
			metric := reg.Get(name)
			if metric != nil {
				return getMetricData(metric), nil
			} else {
				return nil, nil
			}
		}
	case "DELETE":
		err, _ := endpoint.verifyCredentialsFromRequest("system:stats", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		return nil, reg.Unregister(name)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doPrometheusLow(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_QUERY_STATS, req, nil)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()
	for name, metric := range reg.Counters() {
		w.Write([]byte("# TYPE n1ql_" + name + " counter\n"))
		w.Write([]byte("n1ql_" + name + " "))
		w.Write([]byte(fmt.Sprintf("%v\n", metric.Count())))
	}
	for name, metric := range reg.Gauges() {
		w.Write([]byte("# TYPE n1ql_" + name + " gauge\n"))
		w.Write([]byte("n1ql_" + name + " "))
		w.Write([]byte(fmt.Sprintf("%v\n", metric.Value())))
	}
	for name, metric := range localData {
		w.Write([]byte("# TYPE n1ql_" + name + " " + metric + "\n"))
		w.Write([]byte("n1ql_" + name + " "))
		w.Write([]byte(fmt.Sprintf("%v\n", localValue(endpoint.server, name))))
	}

	bName := "bucket"
	if tenant.IsServerless() {
		tenant.Foreach(func(n string, m memory.MemoryManager) {
			w.Write([]byte("# TYPE n1ql_tenant_memory gauge\n"))
			w.Write([]byte("n1ql_tenant_memory{bucket=\"" + n + "\"} "))
			w.Write([]byte(fmt.Sprintf("%v\n", m.AllocatedMemory())))
		})
		bName = "tenant"
	}
	store, ok := datastore.GetDatastore().(datastore.Datastore2)
	if ok {
		store.ForeachBucket(func(b datastore.ExtendedBucket) {
			stats := b.GetIOStats(false, true, true, tenant.IsServerless())
			for n, s := range stats {
				statName := "n1ql_" + bName + "_" + n
				w.Write([]byte("# TYPE " + statName + " gauge\n"))
				w.Write([]byte(statName + "{bucket=\"" + b.Name() + "\"} "))
				w.Write([]byte(fmt.Sprintf("%v\n", s)))
			}
		})
	}

	return textPlain(""), nil
}

func doEmpty(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_QUERY_STATS, req, nil)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	return textPlain(""), nil
}

func doNotFound(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	accounting.UpdateCounter(accounting.INVALID_REQUESTS)
	return nil, nil
}

func doVitals(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_VITALS
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:vitals", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		acctStore := endpoint.server.AccountingStore()
		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		return acctStore.Vitals(durStyle)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func CaptureVitals(endpoint *HttpEndpoint, w io.Writer) error {
	acctStore := endpoint.server.AccountingStore()
	var err error
	var v map[string]interface{}
	v, err = acctStore.Vitals(util.SECONDS)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = json.MarshalNoEscapeToBuffer(v, &buf)
	if err == nil {
		_, err = buf.WriteTo(w)
	}
	return err
}

// Credentials only come from the basic username/password
func (endpoint *HttpEndpoint) getCredentialsFromRequest(ds datastore.Datastore, req *http.Request) (*auth.Credentials,
	errors.Error, bool) {

	isInternal := false

	// only avoid auditing for internal users
	if endpoint.internalUser == "" {
		endpoint.internalUser, _, _ = cbauth.Default.GetHTTPServiceAuth(distributed.RemoteAccess().WhoAmI())
	}
	creds := auth.NewCredentials()
	user, pass, ok := req.BasicAuth()
	if ok {
		creds.Users[user] = pass
		if endpoint.internalUser == user {
			isInternal = true
		}
	}
	creds.HttpRequest = req
	return creds, nil, isInternal
}

func (endpoint *HttpEndpoint) getImpersonate(req *http.Request) (string, errors.Error) {
	return req.FormValue("impersonate"), nil
}

// TODO this needs to be expanded when we support multiple tenants per each user
func (endpoint *HttpEndpoint) getImpersonateBucket(req *http.Request) (string, string, errors.Error) {
	impersonate := req.FormValue("impersonate")
	if len(impersonate) > 0 {
		userName, domain := datastore.DecodeName(impersonate)
		buckets := datastore.GetImpersonateBuckets(userName, domain)
		if len(buckets) > 0 {
			return impersonate, buckets[0], nil
		}

		return impersonate, "", nil
	}
	return "", "", nil
}

func (endpoint *HttpEndpoint) Authorize(req *http.Request) errors.Error {
	ds := datastore.GetDatastore()
	creds, err, _ := endpoint.getCredentialsFromRequest(ds, req)
	if err != nil {
		return err
	}

	privs := auth.NewPrivileges()
	privs.Add("", auth.PRIV_QUERY_STATS, auth.PRIV_PROPS_NONE)
	err = ds.Authorize(privs, creds)
	return err
}

func (endpoint *HttpEndpoint) verifyCredentialsFromRequest(api string, priv auth.Privilege, req *http.Request,
	af *audit.ApiAuditFields) (errors.Error, bool) {

	ds := datastore.GetDatastore()
	creds, err, isInternal := endpoint.getCredentialsFromRequest(ds, req)
	if err != nil {
		return err, false
	}

	if af != nil {
		users := make([]string, 0, len(creds.Users))
		for user := range creds.Users {
			users = append(users, user)
		}
		af.Users = users
	}

	privs := auth.NewPrivileges()
	privs.Add(api, priv, auth.PRIV_PROPS_NONE)
	err = ds.Authorize(privs, creds)
	return err, isInternal
}

func doRedact(req *http.Request) bool {
	val := req.FormValue("redact")
	r, _ := strconv.ParseBool(val)
	return r
}

func doPrepared(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "name")

	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	af.Name = name

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		err = prepareds.DeletePreparedFunc(name, func(e *prepareds.CacheEntry) bool {

			// for serverless user access, we treat entries not owned by the user as not existent
			userName, tenantName, err1 := endpoint.getImpersonateBucket(req)
			return err1 == nil && (userName == "" || e.Prepared.Tenant() == tenantName)
		})
		if err != nil {
			return nil, err
		}
		return true, nil
	} else if req.Method == "PUT" {
		body, err1 := ioutil.ReadAll(req.Body)
		defer req.Body.Close()

		// http.BasicAuth eats the body, so verify credentials after getting the body.
		err, _ := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		if err1 != nil {
			return nil, errors.NewAdminBodyError(err1)
		}

		prepared, _ := prepareds.GetPrepared(name, nil, logging.NULL_LOG)

		// nothing to do if the prepared is there and the plan matches
		if prepared != nil && !prepared.MismatchingEncodedPlan(string(body)) {
			return "", nil
		}
		_, err = prepareds.DecodePrepared(name, string(body), true, logging.NULL_LOG)
		if err != nil {
			return nil, err
		}
		return "", nil
	} else if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:prepareds, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		profiling := req.Method == "POST"
		var res interface{}
		prepareds.PreparedDo(name, func(entry *prepareds.CacheEntry) {
			// for serverless user access, we treat entries not owned by the user as not existent
			userName, tenantName, err1 := endpoint.getImpersonateBucket(req)
			if err1 != nil || (userName != "" && entry.Prepared.Tenant() != tenantName) {
				return
			}
			res = preparedWorkHorse(entry, profiling, doRedact(req), durStyle)
		})
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func preparedWorkHorse(entry *prepareds.CacheEntry, profiling bool, redact bool, durStyle util.DurationStyle) interface{} {
	itemMap := map[string]interface{}{
		"name":            entry.Prepared.Name(),
		"uses":            entry.Uses,
		"statement":       util.Redacted(entry.Prepared.Text(), redact),
		"indexApiVersion": entry.Prepared.IndexApiVersion(),
		"featureControls": entry.Prepared.FeatureControls(),
	}
	if entry.Prepared.QueryContext() != "" {
		itemMap["queryContext"] = entry.Prepared.QueryContext()
	}
	if entry.Prepared.EncodedPlan() != "" {
		itemMap["encoded_plan"] = entry.Prepared.EncodedPlan()
	}
	if entry.Prepared.OptimHints() != nil {
		itemMap["optimizer_hints"] = value.NewMarshalledValue(entry.Prepared.OptimHints())
	}
	isks := entry.Prepared.IndexScanKeyspaces()
	if len(isks) > 0 {
		itemMap["indexScanKeyspaces"] = isks
	}
	txPrepareds, txPlans := entry.Prepared.TxPrepared()
	if len(txPrepareds) > 0 {
		itemMap["txPrepareds"] = txPrepareds
	}
	if profiling {
		itemMap["plan"] = entry.Prepared.Operator
		if len(txPlans) > 0 {
			itemMap["txPlans"] = txPlans
		}
		planVersion := entry.Prepared.PlanVersion()
		if planVersion >= util.MIN_PLAN_VERSION {
			itemMap["planVersion"] = planVersion
		}

		// Subquery plans
		sqPlans := entry.Prepared.GetSubqueryPlansEntry()
		if len(sqPlans) > 0 {
			itemMap["subqueryPlans"] = sqPlans
		}
	}

	if !entry.Prepared.PreparedTime().IsZero() {
		itemMap["planPreparedTime"] = entry.Prepared.PreparedTime().Format(util.DEFAULT_FORMAT)
	}

	// only give times for entries that have completed at least one execution
	if entry.Uses > 0 && entry.RequestTime > 0 {
		itemMap["lastUse"] = entry.LastUse.Format(util.DEFAULT_FORMAT)
		itemMap["avgElapsedTime"] = util.FormatDuration((time.Duration(entry.RequestTime) / time.Duration(entry.Uses)), durStyle)
		itemMap["avgServiceTime"] = util.FormatDuration((time.Duration(entry.ServiceTime) / time.Duration(entry.Uses)), durStyle)
		itemMap["minElapsedTime"] = util.FormatDuration(time.Duration(entry.MinRequestTime), durStyle)
		itemMap["minServiceTime"] = util.FormatDuration(time.Duration(entry.MinServiceTime), durStyle)
		itemMap["maxElapsedTime"] = util.FormatDuration(time.Duration(entry.MaxRequestTime), durStyle)
		itemMap["maxServiceTime"] = util.FormatDuration(time.Duration(entry.MaxServiceTime), durStyle)
	}
	return itemMap
}

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	switch req.Method {
	case "GET", "POST":
		err, _ := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numPrepareds := prepareds.CountPrepareds()
		data := make([]interface{}, 0, numPrepareds)
		profiling := req.Method == "POST"

		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		redact := doRedact(req)
		prepareds.PreparedsForeach(func(name string, d *prepareds.CacheEntry) bool {
			p := preparedWorkHorse(d, profiling, redact, durStyle)
			data = append(data, p)
			return true
		}, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunction(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "name")

	af.EventTypeId = audit.API_ADMIN_FUNCTIONS
	af.Name = name

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:functions_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		functions.FunctionClear(name, nil)
		return true, nil
	} else if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:functions_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:functions_cache, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		var res interface{}

		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		functions.FunctionDo(name, func(entry *functions.FunctionEntry) {
			itemMap := map[string]interface{}{
				"uses": entry.Uses,
			}
			entry.Signature(itemMap)
			entry.Body(itemMap)

			// only give times for entries that have completed at least one execution
			if entry.Uses > 0 && entry.ServiceTime > 0 {
				itemMap["lastUse"] = entry.LastUse.Format(util.DEFAULT_FORMAT)
				itemMap["avgServiceTime"] =
					util.FormatDuration((time.Duration(entry.ServiceTime) / time.Duration(entry.Uses)), durStyle)
				itemMap["minServiceTime"] = util.FormatDuration(time.Duration(entry.MinServiceTime), durStyle)
				itemMap["maxServiceTime"] = util.FormatDuration(time.Duration(entry.MaxServiceTime), durStyle)
			}
			res = itemMap
		})
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunctions(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_FUNCTIONS
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:functions_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numFunctions := functions.CountFunctions()
		data := make([]map[string]interface{}, numFunctions)
		i := 0

		snapshot := func(name string, d *functions.FunctionEntry) bool {

			// FIXME quick hack to avoid overruns
			if i >= numFunctions {
				return false
			}
			data[i] = map[string]interface{}{}
			data[i]["uses"] = d.Uses
			d.Signature(data[i])
			d.Body(data[i])
			if d.Uses > 0 {
				data[i]["lastUse"] = d.LastUse.Format(util.DEFAULT_FORMAT)
			}
			i++
			return true
		}

		functions.FunctionsForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doDictionaryEntry(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "name")

	af.EventTypeId = audit.API_ADMIN_DICTIONARY
	af.Name = name

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:dictionary_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		dictionary.DropDictCacheEntry(name, true)
		return true, nil
	} else if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:dictionary_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:functions_cache, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		var res interface{}

		dictionary.DictCacheEntryDo(name, func(d interface{}) {
			entry := d.(dictionary.DictCacheEntry)

			itemMap := map[string]interface{}{}
			entry.Target(itemMap)
			entry.Dictionary(itemMap)
			res = itemMap
		})
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doDictionary(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_DICTIONARY
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:dictionary_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numKeyspaces := dictionary.CountDictCacheEntries()
		data := make([]map[string]interface{}, numKeyspaces)
		i := 0

		snapshot := func(name string, d interface{}) bool {

			// FIXME quick hack to avoid overruns
			if i >= numKeyspaces {
				return false
			}
			data[i] = map[string]interface{}{}
			entry := d.(dictionary.DictCacheEntry)
			entry.Target(data[i])
			entry.Dictionary(data[i])
			i++
			return true
		}

		dictionary.DictCacheEntriesForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doTask(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "name")

	af.EventTypeId = audit.API_ADMIN_TASKS
	af.Name = name

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:tasks_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		scheduler.DeleteTask(name)
		return true, nil
	} else if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:tasks_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:tasks_cache, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		var res interface{}
		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))

		scheduler.TaskDo(name, func(entry *scheduler.TaskEntry) {
			itemMap := map[string]interface{}{
				"class":        entry.Class,
				"subClass":     entry.SubClass,
				"name":         entry.Name,
				"id":           entry.Id,
				"state":        entry.State,
				"queryContext": entry.QueryContext,
				"submitTime":   entry.PostTime.Format(util.DEFAULT_FORMAT),
				"delay":        util.FormatDuration(entry.Delay, durStyle),
			}
			if entry.Results != nil {
				itemMap["results"] = entry.Results
			}
			if entry.Errors != nil {
				errors := make([]interface{}, 0, len(entry.Errors))
				for _, err := range entry.Errors {
					if err != nil {
						errors = append(errors, err.Object())
					}
				}
				itemMap["errors"] = errors
			}
			if !entry.StartTime.IsZero() {
				itemMap["startTime"] = entry.StartTime.Format(util.DEFAULT_FORMAT)
			}
			if !entry.EndTime.IsZero() {
				itemMap["stopTime"] = entry.EndTime.Format(util.DEFAULT_FORMAT)
			}
			if entry.Description != "" {
				itemMap["description"] = entry.Description
			}
			res = itemMap
		})
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doTasks(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_TASKS
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:tasks_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numTasks := scheduler.CountTasks()
		data := make([]map[string]interface{}, numTasks)
		i := 0

		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		snapshot := func(name string, d *scheduler.TaskEntry) bool {

			// FIXME quick hack to avoid overruns
			if i >= numTasks {
				return false
			}
			data[i] = map[string]interface{}{}
			data[i]["class"] = d.Class
			data[i]["subClass"] = d.SubClass
			data[i]["name"] = d.Name
			data[i]["id"] = d.Id
			data[i]["state"] = d.State
			data[i]["queryContext"] = d.QueryContext
			if d.Results != nil {
				data[i]["results"] = d.Results
			}
			if d.Errors != nil {
				errors := make([]interface{}, 0, len(d.Errors))
				for _, err := range d.Errors {
					if err != nil {
						errors = append(errors, err.Object())
					}
				}
				data[i]["errors"] = errors
			}
			data[i]["submitTime"] = d.PostTime.Format(util.DEFAULT_FORMAT)
			data[i]["delay"] = util.FormatDuration(d.Delay, durStyle)
			if !d.StartTime.IsZero() {
				data[i]["startTime"] = d.StartTime.Format(util.DEFAULT_FORMAT)
			}
			if !d.EndTime.IsZero() {
				data[i]["stopTime"] = d.EndTime.Format(util.DEFAULT_FORMAT)
			}
			if d.Description != "" {
				data[i]["description"] = d.Description
			}
			i++
			return true
		}

		scheduler.TasksForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doGlobalBackup(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_BACKUP
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_BACKUP_CLUSTER, req, af)
		if err != nil {
			return nil, err
		}

		numFunctions, err1 := functionsStorage.Count("")
		if err1 != nil {
			return nil, errors.NewStorageAccessError("backup", err1)
		}
		data := make([]interface{}, 0, numFunctions)

		// this check doesn't force waiting if the migration is active
		version := dictionary.SupportedBackupVersion()
		if version == datastore.BACKUP_NOT_POSSIBLE {
			return nil, errors.NewBackupNotPossible()
		}

		snapshot := func(name string, v value.Value) error {
			path := algebra.ParsePath(name)
			if len(path) == 2 {
				data = append(data, v)
			}
			return nil
		}

		functionsStorage.Foreach("", snapshot)

		if version == datastore.BACKUP_VERSION_1 {
			// even though global functions aren't changed between v1 & v2, write a v1 header when v1 is indicated
			return makeBackupHeaderV1(data), nil
		}
		return makeBackupHeader(data, nil, nil), nil

	case "POST":
		var iState json.IndexState

		// http.BasicAuth eats the body, so verify credentials after getting the body.
		bytes, e := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), "restore body")
		}

		err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_BACKUP_CLUSTER, req, af)
		if err != nil {
			return nil, err
		}

		fns, _, _, e := checkBackupHeader(bytes)
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(e, "restore body")
		}

		// if it's not an array, we'll try a single function
		if fns[0] != '[' {
			err := doFunctionRestore(fns, 2, "", nil, nil, nil)
			if err != nil {
				return nil, err
			}
		}
		index := 0
		json.SetIndexState(&iState, fns)
		for {
			v, err := iState.FindIndex(index)
			if err != nil {
				iState.Release()
				return nil, errors.NewServiceErrorBadValue(err, "restore body")
			}
			if string(v) == "" {
				break
			}
			index++
			err1 := doFunctionRestore(v, 2, "", nil, nil, nil)
			if err1 != nil {
				iState.Release()
				return nil, err1
			}
		}
		iState.Release()
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
	return "", nil
}

func doBucketBackup(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, bucket := router.RequestValue(req, "bucket")
	af.EventTypeId = audit.API_ADMIN_BACKUP

	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest(bucket, auth.PRIV_BACKUP_BUCKET, req, af)
		if err != nil {
			return nil, err
		}

		include, err := newFilter(req.FormValue("include"))
		if err != nil {
			return nil, err
		}
		exclude, err := newFilter(req.FormValue("exclude"))
		if err != nil {
			return nil, err
		}
		// this check doesn't force waiting if the migration is active
		version := dictionary.SupportedBackupVersion()
		if version == datastore.BACKUP_NOT_POSSIBLE {
			return nil, errors.NewBackupNotPossible()
		}
		fns := make([]interface{}, 0)
		// do not archive functions if the metadata is already stored in KV
		snapshot := func(name string, v value.Value) error {
			path := algebra.ParsePath(name)
			if len(path) == 4 && path[1] == bucket && filterEval(path, include, exclude) {
				fns = append(fns, v)
			}
			return nil
		}
		err1 := functionsStorage.Foreach(bucket, snapshot)
		if err1 != nil {
			if ue, ok := err1.(errors.Error); ok {
				return nil, ue
			}
			return nil, errors.NewServiceErrorBadValue(err1, "UDF backup")
		}

		if version == datastore.BACKUP_VERSION_1 {
			return makeBackupHeaderV1(fns), nil
		}

		seqs, err := sequences.BackupSequences("default", bucket, func(name string) bool {
			path := algebra.ParsePath(name)
			return filterEval(path, include, exclude)
		})
		if err != nil {
			return nil, err
		}

		keys := make([]string, 1)
		res := make(map[string]value.AnnotatedValue, 1)
		cbo := make([]interface{}, 0)
		err = datastore.ScanSystemCollection(bucket, "cbo::", nil,
			func(key string, systemCollection datastore.Keyspace) errors.Error {
				// Exclude index records as they're bound to the index ID which will differ when restored. Furthermore, indices
				// will be rebuilt following a restore rendering these stats obsolete.
				if strings.Contains(key, "(index)") {
					return nil
				}
				parts := strings.Split(key, "::")
				path, _, _, _ := dictionary.GetCBOKeyspaceFromKey(parts[len(parts)-1])
				p := algebra.ParsePath(path)
				if !filterEval(p, include, exclude) {
					return nil
				}
				keys[0] = key
				errs := systemCollection.Fetch(keys, res, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
				if errs != nil && len(errs) > 0 {
					return errs[0]
				}
				av, ok := res[key]
				if ok {
					b, err := av.MarshalJSON()
					if err == nil {
						d := make(map[string]interface{})
						d["key"] = key
						sn := snappy.Encode(nil, b)
						b64 := base64.StdEncoding.EncodeToString(sn)
						d["value"] = b64
						cbo = append(cbo, d)
					}
				}
				return nil
			}, nil)
		if err != nil {
			return nil, err
		}

		return makeBackupHeader(fns, seqs, cbo), nil

	case "POST":
		var iState json.IndexState

		// http.BasicAuth eats the body, so verify credentials after getting the body.
		body, e := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), "restore body")
		}

		err, _ := endpoint.verifyCredentialsFromRequest(bucket, auth.PRIV_BACKUP_BUCKET, req, af)
		if err != nil {
			return nil, err
		}
		fns, seqs, cbo, e := checkBackupHeader(body)
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(e, "restore body")
		}

		include, err := newFilter(req.FormValue("include"))
		if err != nil {
			return nil, err
		}
		exclude, err := newFilter(req.FormValue("exclude"))
		if err != nil {
			return nil, err
		}
		remap, err := newRemapper(req.FormValue("remap"), "UDF")
		if err != nil {
			return nil, err
		}

		// if it's not an array, we'll try a single function
		if fns[0] != '[' {
			err := doFunctionRestore(fns, 4, bucket, include, exclude, remap)
			if err != nil {
				return nil, err
			}
			return "", nil
		}
		index := 0
		json.SetIndexState(&iState, fns)
		for {
			v, err := iState.FindIndex(index)
			if err != nil {
				iState.Release()
				return nil, errors.NewServiceErrorBadValue(err, "restore body")
			}
			if string(v) == "" {
				break
			}
			index++
			err1 := doFunctionRestore(v, 4, bucket, include, exclude, remap)
			if err1 != nil {
				iState.Release()
				return nil, err1
			}
		}
		iState.Release()

		if seqs != nil {
			remap, err = newRemapper(req.FormValue("remap"), "sequence")
			if err != nil {
				return nil, err
			}
			index = 0
			json.SetIndexState(&iState, seqs)
			for {
				v, err := iState.FindIndex(index)
				if err != nil {
					iState.Release()
					return nil, errors.NewServiceErrorBadValue(err, "restore sequence")
				}
				if string(v) == "" {
					break
				}
				index++
				err1 := doSequenceRestore(v, bucket, include, exclude, remap)
				if err1 != nil {
					iState.Release()
					return nil, err1
				}
			}
			iState.Release()
		}

		if cbo != nil {
			remap, err = newRemapper(req.FormValue("remap"), "cbo")
			if err != nil {
				return nil, err
			}
			store := datastore.GetDatastore()
			if store == nil {
				return nil, errors.NewServiceErrorBadValue(errors.NewNoDatastoreError(), "restore cbo")
			}
			systemCollection, err := store.GetSystemCollection(bucket)
			if err != nil {
				return nil, errors.NewServiceErrorBadValue(err, "restore cbo")
			}

			// gather a list of target keyspaces
			purge := make(map[string]bool)
			index = 0
			json.SetIndexState(&iState, cbo)
			for {
				v, err := iState.FindIndex(index)
				if err != nil {
					iState.Release()
					return nil, errors.NewServiceErrorBadValue(err, "restore cbo")
				}
				if string(v) == "" {
					break
				}
				index++
				ks, ok := getCBORestoreKeyspace(v, bucket, include, exclude, remap)
				if ok {
					purge[ks] = true
				}
			}
			iState.Release()

			// ensure we've cleared existing stats
			for k, _ := range purge {
				dictionary.DropDictEntryAndAllCache(k, datastore.NULL_QUERY_CONTEXT, false)
			}

			// restore the stats
			index = 0
			json.SetIndexState(&iState, cbo)
			for {
				v, err := iState.FindIndex(index)
				if err != nil {
					iState.Release()
					return nil, errors.NewServiceErrorBadValue(err, "restore cbo")
				}
				if string(v) == "" {
					break
				}
				index++
				err1 := doCBORestore(v, bucket, include, exclude, remap, systemCollection)
				if err1 != nil {
					iState.Release()
					return nil, err1
				}
			}
			iState.Release()
		}

		// after restoring cleanup any stale entries in the system collection
		go dictionary.CleanupSystemCollection("default", bucket)

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
	return "", nil
}

const _MAGIC_KEY = "udfMagic"
const _MAGIC = "4D6172636F2072756C6573"
const _VERSION_KEY = "version"
const _VERSION = "0x02"
const _VERSION_1 = "0x01"
const _VERSION_MIN = 1
const _VERSION_MAX = 2
const _UDF_KEY = "udfs"
const _SEQ_KEY = "seqs"
const _CBO_KEY = "cbo"

func makeBackupHeader(v interface{}, s interface{}, c interface{}) interface{} {
	data := make(map[string]interface{}, 4)
	data[_MAGIC_KEY] = _MAGIC
	data[_VERSION_KEY] = _VERSION
	data[_UDF_KEY] = v
	data[_SEQ_KEY] = s
	data[_CBO_KEY] = c
	return data
}

func makeBackupHeaderV1(v interface{}) interface{} {
	data := make(map[string]interface{}, 4)
	data[_MAGIC_KEY] = _MAGIC
	data[_VERSION_KEY] = _VERSION_1
	data[_UDF_KEY] = v
	return data
}

func checkBackupHeader(d []byte) ([]byte, []byte, []byte, errors.Error) {
	var oState json.KeyState
	json.SetKeyState(&oState, d)
	magic, err := oState.FindKey(_MAGIC_KEY)
	if err != nil || string(magic) != "\""+_MAGIC+"\"" {
		oState.Release()
		return nil, nil, nil, errors.NewServiceErrorBadValue(err, "restore: invalid magic")
	}
	version, err := oState.FindKey(_VERSION_KEY)
	var ver uint64
	if err == nil {
		trimmed := strings.Trim(string(version), "\"")
		if !strings.HasPrefix(trimmed, "0x") || len(trimmed) <= 2 {
			return nil, nil, nil, errors.NewServiceErrorBadValue(nil, "restore: invalid version")
		}
		ver, err = strconv.ParseUint(trimmed[2:], 16, 64)
	}
	if err != nil || ver < _VERSION_MIN || ver > _VERSION_MAX {
		oState.Release()
		return nil, nil, nil, errors.NewServiceErrorBadValue(err, "restore: invalid version")
	}
	udfs, err := oState.FindKey(_UDF_KEY)
	if err != nil {
		oState.Release()
		return nil, nil, nil, errors.NewServiceErrorBadValue(err, "restore: missing UDF field")
	}
	// only expect sequences & cbo for version 2+ backup images
	var seqs []byte
	var cbo []byte
	if ver >= 2 {
		seqs, err = oState.FindKey(_SEQ_KEY)
		if err != nil {
			oState.Release()
			return nil, nil, nil, errors.NewServiceErrorBadValue(err, "restore: missing sequences field")
		}
		cbo, err = oState.FindKey(_CBO_KEY)
		if err != nil {
			oState.Release()
			return nil, nil, nil, errors.NewServiceErrorBadValue(err, "restore: missing cbo field")
		}
	}
	oState.Release()
	return udfs, seqs, cbo, nil
}

type matcher map[string]map[string]bool
type remapper map[string]map[string][]string

func parsePath(p string) []string {
	var out []string

	if strings.IndexByte(p, '\\') < 0 {
		return strings.Split(p, ".")
	}
	wasSlash := false
	elem := ""
	for _, c := range p {
		switch c {
		case '\\':
			wasSlash = true
		case '.':
			if wasSlash {
				elem = elem + string(c)
			} else {
				out = append(out, elem)
				elem = ""
			}
			wasSlash = false
		default:
			elem = elem + string(c)
			wasSlash = false
		}
	}
	if len(elem) > 0 {
		out = append(out, elem)
	}
	return out
}

// filter implements the backup service include and exclude form entries
// only considers scopes and collections
// duplicates entries are not allowed
func newFilter(p string) (matcher, errors.Error) {
	if len(p) == 0 {
		return nil, nil
	}
	paths := strings.Split(p, ",")
	if len(paths) == 0 {
		return nil, nil
	}
	m := make(matcher, len(paths))
	for _, entry := range paths {
		elems := parsePath(entry)
		s, ok := m[elems[0]]

		// found, check and add to scope
		if ok {
			if len(s) > 0 && len(elems) > 1 {
				s[elems[1]] = true
			} else {
				return nil, errors.NewServiceErrorBadValue(go_errors.New("filter includes duplicate entries"),
					"restore parameters")
			}

			// currently we ignore overlapping and duplicates
		} else if len(elems) > 1 {

			// not found, entry has bucket and scope
			m[elems[0]] = map[string]bool{elems[1]: true}
		} else {

			// not found, entry has just scope
			m[elems[0]] = map[string]bool{}
		}
	}
	return m, nil
}

func (f matcher) match(p []string) bool {
	scope, ok := f[p[2]]
	if !ok {
		return false
	}
	if len(scope) == 0 {
		return true
	}

	// any collections matches are ignored
	return false
}

func filterEval(path []string, include, exclude matcher) bool {
	if len(include) == 0 && len(exclude) == 0 {
		return true
	}
	if len(include) > 0 {
		return include.match(path)
	}
	if len(exclude) > 0 {
		return !exclude.match(path)
	}

	// should never reach this
	return false
}

func newRemapper(p string, serv string) (remapper, errors.Error) {
	if len(p) == 0 {
		return nil, nil
	}
	paths := strings.Split(p, ",")
	if len(paths) == 0 {
		return nil, nil
	}
	m := make(remapper, len(paths))
	for _, entry := range paths {
		rm := strings.Split(entry, ":")
		if len(rm) != 2 {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("remapper is incomplete"), serv+" restore parameter")
		}
		inElems := parsePath(rm[0])
		outElems := parsePath(rm[1])
		if len(inElems) != len(outElems) || len(inElems) > 2 {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("remapper has mismatching or invalid entries"),
				serv+" restore parameter")
		}
		s, ok := m[inElems[0]]

		// found, check and add to bucket
		if ok {
			_, foundEmpty := s[""]
			if len(inElems) > 0 && !foundEmpty {
				s[inElems[1]] = outElems
			} else {
				return nil, errors.NewServiceErrorBadValue(go_errors.New("filter includes duplicate entries"),
					serv+" restore parameter")
			}

			// currently we ignore overlapping and duplicates
		} else if len(inElems) > 1 {

			// not found, entry has scope and collection
			m[inElems[0]] = map[string][]string{inElems[1]: outElems}
		} else {

			// not found, entry has just scope
			m[inElems[0]] = map[string][]string{"": outElems}
		}
	}
	return m, nil
}

func (r remapper) remap(bucket string, path []string) {
	if bucket == "" || len(path) != 4 {
		return
	}
	path[1] = bucket
	scope, ok := r[path[2]]

	// in order to remap functions, we must be remapping scopes, not individual collections
	if ok {
		target, ok := scope[""]
		if ok {
			path[2] = target[0]
		}
	}
}

// Restore semantics:
//   - for global functions, no include, exclude remap is possible.
//     any non global functions passed will simply be skipped
//   - for scope functions, include, exclude and remap will only operate at scope level
//     currently no check is made that the target scope exist, which may very well leave stale function definitions
//   - for both cases the only thing that counts is the name, and not the signature, it is therefore possible to go back in
//     time and restore function definitions with different parameter lists
//   - remapping to an existing scope will replace existing functions, with the same or different signature, which may not be
//     intended, however a different conflict resolution would prevent going back in time
//   - be aware that remapping may have other side effects: for query context based statements contained within functions, the new
//     targets will be under the new bucket / scope query context, while accesses with full path will remain unchanged.
//     this makes perfect sense, but may not necessarely be what the user intended
func doFunctionRestore(v []byte, l int, b string, include, exclude matcher, remap remapper) errors.Error {
	var oState json.KeyState

	json.SetKeyState(&oState, v)
	identity, err := oState.FindKey("identity")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "UDF missing identity")
	}
	definition, err := oState.FindKey("definition")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "UDF missing body")
	}
	oState.Release()
	if string(identity) == "" || string(definition) == "" {
		return errors.NewServiceErrorBadValue(go_errors.New("missing function definition or body"), "UDF restore")
	}
	path, err1 := functionsResolver.MakePath(identity)
	if err1 != nil {
		return errors.NewServiceErrorBadValue(err1, "UDF restore")
	}

	// we skip the entries that do no apply
	if len(path) != l {
		return nil
	}

	if l == 2 || filterEval(path, include, exclude) {
		remap.remap(b, path)

		name, err1 := functionsBridge.NewFunctionName(path, path[0], "")
		if err1 != nil {
			return errors.NewServiceErrorBadValue(err1, "UDF restore")
		}
		body, err1 := functionsResolver.MakeBody(name.Name(), definition)
		if err1 != nil {
			return errors.NewServiceErrorBadValue(err1, "UDF restore")
		}
		return name.Save(body, true)
	}
	return nil
}

func doTransaction(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, txId := router.RequestValue(req, "txid")

	af.EventTypeId = audit.API_ADMIN_TRANSACTIONS
	af.Name = txId

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:transactions", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		return true, transactions.DeleteTransContext(txId, true)
	} else if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:transactions", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:transactions, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		var res interface{}
		transactions.TransactionEntryDo(txId, func(d interface{}) {
			entry := d.(*transactions.TranContext)
			itemMap := map[string]interface{}{}
			entry.Content(itemMap)
			res = itemMap
		})
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doTransactions(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request,
	af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_TRANSACTIONS
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:transactions", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numTransactions := transactions.CountTransContext()
		data := make([]map[string]interface{}, 0, numTransactions)

		snapshot := func(name string, d interface{}) bool {
			tranContext := d.(*transactions.TranContext)
			entry := map[string]interface{}{}
			tranContext.Content(entry)
			data = append(data, entry)
			return true
		}

		transactions.TransactionEntriesForeach(snapshot, nil)
		return data, nil
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, requestId := router.RequestValue(req, "request")

	af.EventTypeId = audit.API_ADMIN_ACTIVE_REQUESTS
	af.Request = requestId

	if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:active_requests, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		impersonate, err1 := endpoint.getImpersonate(req)
		if err1 != nil {
			return nil, err1
		}
		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		return processActiveRequest(endpoint, requestId, impersonate, (req.Method == "POST"), doRedact(req), durStyle), nil

	} else if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		impersonate, err1 := endpoint.getImpersonate(req)
		if err1 != nil {
			return nil, err1
		}
		if endpoint.actives.Delete(requestId, true, func(r server.Request) bool {
			if impersonate != "" {
				users := datastore.CredsArray(r.Credentials())
				return len(users) > 0 && impersonate == users[0]
			}
			return true
		}) {
			return nil, errors.NewServiceErrorHttpReq(requestId)
		}

		return true, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func processActiveRequest(endpoint *HttpEndpoint, requestId string, userName string, profiling bool, redact bool,
	durStyle util.DurationStyle) interface{} {

	var res interface{}

	_ = endpoint.actives.Get(requestId, func(request server.Request) {
		if userName != "" {
			users := datastore.CredsArray(request.Credentials())
			if len(users) == 0 || userName != users[0] {
				return
			}
		}
		res = activeRequestWorkHorse(endpoint, request, profiling, redact, durStyle)
	})
	return res
}

func activeRequestWorkHorse(endpoint *HttpEndpoint, request server.Request, profiling bool, redact bool,
	durStyle util.DurationStyle) interface{} {

	reqMap := map[string]interface{}{
		"requestId": request.Id().String(),
	}
	cId := request.ClientID().String()
	if cId != "" {
		reqMap["clientContextID"] = cId
	}
	if request.Statement() != "" {
		reqMap["statement"] = util.Redacted(request.Statement(), redact)
	}
	if request.Type() != "" {
		reqMap["statementType"] = request.Type()
	}
	if request.QueryContext() != "" {
		reqMap["queryContext"] = request.QueryContext()
	}
	if request.Prepared() != nil {
		p := request.Prepared()
		reqMap["preparedName"] = p.Name()
		reqMap["preparedText"] = util.Redacted(p.Text(), redact)
	}
	if request.TxId() != "" {
		reqMap["txid"] = request.TxId()
	}
	reqMap["requestTime"] = request.RequestTime().Format(expression.DEFAULT_FORMAT)
	reqMap["elapsedTime"] = util.FormatDuration(time.Since(request.RequestTime()), durStyle)
	if request.Timeout() > time.Duration(0) {
		reqMap["timeout"] = util.FormatDuration(request.Timeout(), durStyle)
	}
	if request.ServiceTime().IsZero() {
		reqMap["executionTime"] = util.FormatDuration(0, durStyle)
	} else {
		reqMap["executionTime"] = util.FormatDuration(time.Since(request.ServiceTime()), durStyle)
	}
	if !request.TransactionStartTime().IsZero() {
		reqMap["transactionElapsedTime"] = util.FormatDuration(time.Since(request.TransactionStartTime()), durStyle)
		remTime := request.TxTimeout() - time.Since(request.TransactionStartTime())
		if remTime > 0 {
			reqMap["transactionRemainingTime"] = util.FormatDuration(remTime, durStyle)
		}
	}
	reqMap["state"] = request.State().StateName()
	reqMap["scanConsistency"] = request.ScanConsistency()
	if request.UseFts() {
		reqMap["useFts"] = request.UseFts()
	}
	if request.UseCBO() {
		reqMap["useCBO"] = request.UseCBO()
	}
	if request.UseReplica() == value.TRUE {
		reqMap["useReplica"] = value.TristateToString(request.UseReplica())
	}
	reqMap["n1qlFeatCtrl"] = request.FeatureControls()

	p := request.Output().FmtPhaseCounts()
	if p != nil {
		reqMap["phaseCounts"] = p
	}
	p = request.Output().FmtPhaseOperators()
	if p != nil {
		reqMap["phaseOperators"] = p
	}
	p = request.Output().FmtPhaseTimes(durStyle)
	if p != nil {
		reqMap["phaseTimes"] = p
	}
	usedMemory := request.UsedMemory()
	if usedMemory != 0 {
		reqMap["usedMemory"] = usedMemory
	}
	if profiling {

		prof := request.Profile()
		if prof == server.ProfUnset {
			prof = endpoint.server.Profile()
		}

		// TODO - check lifetime of entry
		// by the time we marshal, is this still valid?
		if prof == server.ProfOn || prof == server.ProfBench {
			timings := request.GetTimings()
			if timings != nil {
				reqMap["timings"] = util.InterfaceRedacted(value.ApplyDurationStyleToValue(durStyle, value.NewMarshalledValue(timings)),
					redact)
				p = request.Output().FmtOptimizerEstimates(timings)
				if p != nil {
					reqMap["optimizerEstimates"] = value.NewValue(util.InterfaceRedacted(p, redact))
				}
			}
		}
		cpuTime := request.CpuTime()
		if cpuTime > time.Duration(0) {
			reqMap["cpuTime"] = util.FormatDuration(cpuTime, durStyle)
		}

		var ctrl bool
		ctr := request.Controls()
		if ctr == value.NONE {
			ctrl = endpoint.server.Controls()
		} else {
			ctrl = (ctr == value.TRUE)
		}
		if ctrl {
			na := request.FormattedNamedArgs()
			if na != nil {
				reqMap["namedArgs"] = util.InterfaceRedacted(na, redact)
			}
			pa := request.PositionalArgs()
			if pa != nil {
				reqMap["positionalArgs"] = util.InterfaceRedacted(pa, redact)
			}
			memoryQuota := request.MemoryQuota()
			if memoryQuota != 0 {
				reqMap["memoryQuota"] = memoryQuota
			}
		}
	}
	credsString := datastore.CredsString(request.Credentials())
	if credsString != "" {
		reqMap["users"] = util.Redacted(credsString, redact)
	}
	remoteAddr := request.RemoteAddr()
	if remoteAddr != "" {
		reqMap["remoteAddr"] = remoteAddr
	}
	userAgent := request.UserAgent()
	if userAgent != "" {
		reqMap["userAgent"] = userAgent
	}
	throttleTime := request.ThrottleTime()
	if throttleTime > time.Duration(0) {
		reqMap["throttleTime"] = util.FormatDuration(throttleTime, durStyle)
	}

	return reqMap
}

func doActiveRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}

	numRequests, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}

	requests := make([]interface{}, 0, numRequests)
	profiling := req.Method == "POST"
	durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))

	redact := doRedact(req)
	endpoint.actives.ForEach(func(requestId string, request server.Request) bool {
		r := activeRequestWorkHorse(endpoint, request, profiling, redact, durStyle)
		requests = append(requests, r)
		return true
	}, nil)
	return requests, nil
}

func CaptureActiveRequests(endpoint *HttpEndpoint, w io.Writer) error {
	var buf bytes.Buffer
	var err error
	first := true
	_, err = w.Write([]byte{'['})
	if err != nil {
		return err
	}
	endpoint.actives.ForEach(func(requestId string, request server.Request) bool {
		if !first {
			_, err = w.Write([]byte{','})
		}
		if err == nil {
			r := activeRequestWorkHorse(endpoint, request, true, true, util.GetDurationStyle())
			err = json.MarshalNoEscapeToBuffer(r, &buf)
			if err == nil {
				_, err = buf.WriteTo(w)
				buf.Reset()
			}
		}
		first = false
		return err == nil
	}, nil)
	if err == nil {
		_, err = w.Write([]byte{']', '\n'})
	}
	return err
}

func doCompletedRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, requestId := router.RequestValue(req, "request")

	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	af.Request = requestId

	if req.Method == "GET" || req.Method == "POST" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:completed_requests, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		impersonate, err1 := endpoint.getImpersonate(req)
		if err1 != nil {
			return nil, err1
		}
		durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
		return processCompletedRequest(requestId, impersonate, (req.Method == "POST"), doRedact(req), durStyle), nil
	} else if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		impersonate, err1 := endpoint.getImpersonate(req)
		if err1 != nil {
			return nil, err1
		}
		err = server.RequestDelete(requestId, func(request *server.RequestLogEntry) bool {
			return impersonate == "" || impersonate == request.Users
		})
		if err != nil {
			return nil, err
		} else {
			return true, nil
		}
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func processCompletedRequest(requestId string, userName string, profiling bool, redact bool,
	durStyle util.DurationStyle) interface{} {

	var res interface{}

	server.RequestDo(requestId, func(request *server.RequestLogEntry) {
		if userName != "" && userName != request.Users {
			return
		}
		res = request.Format(profiling, redact, durStyle)
	})
	return res
}

func doCompletedRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}

	numRequests := server.RequestsCount()
	requests := make([]interface{}, 0, numRequests)
	profiling := req.Method == "POST"

	durStyle, _ := util.IsDurationStyle(req.FormValue("duration_style"))
	redact := doRedact(req)
	server.RequestsForeach(func(requestId string, request *server.RequestLogEntry) bool {
		r := request.Format(profiling, redact, durStyle)
		requests = append(requests, r)
		return true
	}, nil)
	return requests, nil
}

func CaptureCompletedRequests(w io.Writer) error {
	var buf bytes.Buffer
	var err error
	first := true
	_, err = w.Write([]byte{'['})
	if err != nil {
		return err
	}
	server.RequestsForeach(func(requestId string, request *server.RequestLogEntry) bool {
		if !first {
			_, err = w.Write([]byte{','})
		}
		if err == nil {
			r := request.Format(true, true, util.GetDurationStyle())
			err = json.MarshalNoEscapeToBuffer(r, &buf)
			if err == nil {
				_, err = buf.WriteTo(w)
				buf.Reset()
			}
		}
		first = false
		return err == nil
	}, nil)
	if err == nil {
		_, err = w.Write([]byte{']', '\n'})
	}
	return err
}

func doPreparedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	userName, tenantName, err := endpoint.getImpersonateBucket(req)
	if err != nil {
		return nil, err
	}
	numEntries := prepareds.CountPrepareds()
	keys := make([]string, 0, numEntries)
	var snapshot func(key string, request *prepareds.CacheEntry) bool

	if userName == "" {
		snapshot = func(key string, prepared *prepareds.CacheEntry) bool {
			keys = append(keys, key)
			return true
		}
	} else {
		snapshot = func(key string, prepared *prepareds.CacheEntry) bool {
			if prepared.Prepared.Tenant() == tenantName {
				keys = append(keys, key)
			}
			return true
		}
	}
	prepareds.PreparedsForeach(snapshot, nil)
	return keys, nil
}

func doRequestIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	userName, err := endpoint.getImpersonate(req)
	if err != nil {
		return nil, err
	}
	numEntries, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}
	requests := make([]string, 0, numEntries)
	var snapshot func(requestId string, request server.Request) bool

	if userName == "" {
		snapshot = func(requestId string, request server.Request) bool {
			requests = append(requests, requestId)
			return true
		}
	} else {
		snapshot = func(requestId string, request server.Request) bool {

			// for ease of processing we ignore the tenant and
			// we expect the request to only have the one user
			// this could be expanded later
			users := datastore.CredsArray(request.Credentials())
			if len(users) > 0 && userName == users[0] {
				requests = append(requests, requestId)
			}
			return true
		}
	}
	endpoint.actives.ForEach(snapshot, nil)
	return requests, nil
}

func doCompletedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	userName, err := endpoint.getImpersonate(req)
	if err != nil {
		return nil, err
	}
	numEntries := server.RequestsCount()
	completed := make([]string, 0, numEntries)
	var snapshot func(requestId string, request *server.RequestLogEntry) bool

	if userName == "" {
		snapshot = func(requestId string, request *server.RequestLogEntry) bool {
			completed = append(completed, requestId)
			return true
		}
	} else {
		snapshot = func(requestId string, request *server.RequestLogEntry) bool {

			// ditto
			if userName == request.Users {
				completed = append(completed, requestId)
			}
			return true
		}
	}
	server.RequestsForeach(snapshot, nil)
	return completed, nil
}

func doFunctionsIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:functions_cache", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	return functions.NameFunctions(), nil
}

func doDictionaryIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:dictionary_cache", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	return dictionary.NameDictCacheEntries(), nil
}

func doTasksIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:tasks", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	return scheduler.NameTasks(), nil
}

func doTransactionsIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request,
	af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:transactions", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	return transactions.NameTransactions(), nil
}

var localData = map[string]string{
	"load":             "gauge",
	"load_factor":      "gauge",
	"active_requests":  "gauge",
	"queued_requests":  "gauge",
	"allocated_values": "counter",
	"node_memory":      "gauge",
	"node_rss":         "gauge",
}

func isLocal(metric string) bool {
	return localData[metric] != ""
}

func getLocalData(serv *server.Server, metric string) map[string]interface{} {
	values := make(map[string]interface{})
	switch metric {
	case "load":
		values["value"] = serv.Load()
	case "load_factor":
		values["value"] = serv.LoadFactor()
	case "active_requests":
		values["count"] = serv.ActiveRequests()
	case "queued_requests":
		values["count"] = serv.QueuedRequests()
	case "allocated_values":
		values["value"] = value.AllocatedValuesCount()
	case "node_memory":
		values["value"] = memory.AllocatedMemory()
	case "node_rss":
		if stats, err := system.NewSystemStats(); err == nil {
			if _, rss, err := stats.ProcessRSS(); err == nil {
				values["value"] = rss
			} else {
				values["value"] = 0
			}
			stats.Close()
		}
	}
	return values
}

func localValue(serv *server.Server, metric string) interface{} {
	switch metric {
	case "load":
		return serv.Load()
	case "load_factor":
		return serv.LoadFactor()
	case "active_requests":
		return serv.ActiveRequests()
	case "queued_requests":
		return serv.QueuedRequests()
	case "allocated_values":
		return value.AllocatedValuesCount()
	case "node_memory":
		return memory.AllocatedMemory()
	case "node_rss":
		var rss uint64
		if stats, err := system.NewSystemStats(); err == nil {
			if _, rss, err = stats.ProcessRSS(); err != nil {
				rss = 0
			}
			stats.Close()
		}
		return rss
	}
	return nil
}

func getMetricData(metric accounting.Metric) map[string]interface{} {
	values := make(map[string]interface{})
	switch metric := metric.(type) {
	case accounting.Counter:
		values["count"] = metric.Count()
	case accounting.Gauge:
		values["value"] = metric.Value()
	case accounting.Meter:
		values["count"] = metric.Count()
		values["1m.rate"] = metric.Rate1()
		values["5m.rate"] = metric.Rate5()
		values["15m.rate"] = metric.Rate15()
		values["mean.rate"] = metric.RateMean()
	case accounting.Timer:
		ps := metric.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = metric.Count()
		values["min"] = metric.Min()
		values["max"] = metric.Max()
		values["mean"] = metric.Mean()
		values["stddev"] = metric.StdDev()
		values["median"] = ps[0]
		values["75%"] = ps[1]
		values["95%"] = ps[2]
		values["99%"] = ps[3]
		values["99.9%"] = ps[4]
		values["1m.rate"] = metric.Rate1()
		values["5m.rate"] = metric.Rate5()
		values["15m.rate"] = metric.Rate15()
		values["mean.rate"] = metric.RateMean()
	case accounting.Histogram:
		ps := metric.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
		values["count"] = metric.Count()
		values["min"] = metric.Min()
		values["max"] = metric.Max()
		values["mean"] = metric.Mean()
		values["stddev"] = metric.StdDev()
		values["median"] = ps[0]
		values["75%"] = ps[1]
		values["95%"] = ps[2]
		values["99%"] = ps[3]
		values["99.9%"] = ps[4]
	}
	return values
}

func doForceGC(ep *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_GC
	err, _ := ep.verifyCredentialsFromRequest("", auth.PRIV_CLUSTER_ADMIN, req, af)
	if err != nil {
		return nil, err
	}

	var before runtime.MemStats
	var after runtime.MemStats
	resp := make(map[string]interface{}, 3)
	switch req.Method {
	case "GET":
		runtime.ReadMemStats(&before)
		runtime.GC()
		runtime.ReadMemStats(&after)
		logging.Warnf("Admin endpoint forced GC. Freed: %v", ffdc.Human(before.HeapAlloc-after.HeapAlloc))
		resp["status"] = "GC invoked"
		resp["freed"] = (before.HeapAlloc - after.HeapAlloc)
	case "POST":
		var before runtime.MemStats
		var after runtime.MemStats
		runtime.ReadMemStats(&before)
		debug.FreeOSMemory()
		runtime.ReadMemStats(&after)
		logging.Warnf("Admin endpoint forced GC. Freed: %v Released: %v",
			ffdc.Human(before.HeapAlloc-after.HeapAlloc),
			ffdc.Human(after.HeapReleased-before.HeapReleased))
		resp["status"] = "GC invoked and memory released"
		resp["freed"] = (before.HeapAlloc - after.HeapAlloc)
		resp["released"] = (after.HeapReleased - before.HeapReleased)
	}
	return resp, nil
}

var earliest time.Time

func doManualFFDC(ep *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_FFDC
	err, _ := ep.verifyCredentialsFromRequest("", auth.PRIV_CLUSTER_ADMIN, req, af)
	if err != nil {
		return nil, err
	}

	logging.Warnf("Manual FFDC collection invoked.")
	if ffdc.Capture(ffdc.Manual) {
		ffdc.Reset(ffdc.Manual)
		resp := make(map[string]interface{}, 2)
		resp["status"] = "FFDC invoked."
		earliest = time.Now().Add(ffdc.FFDC_MIN_INTERVAL)
		resp["next_earliest"] = earliest.Format(util.DEFAULT_FORMAT)
		return resp, nil
	} else {
		if time.Now().Before(earliest) {
			return nil, errors.NewAdminManualFFDCError("Ensure sufficient interval between invocations.",
				int(earliest.Sub(time.Now()).Seconds()))
		} else {
			return nil, errors.NewAdminManualFFDCError("", 0)
		}
	}
}

func doLog(ep *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_STREAM_LOG
	err, _ := ep.verifyCredentialsFromRequest("", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}

	_, fileParam := router.RequestValue(req, "file")

	// The fileParam is already URL decoded
	if fileParam == "" {
		fileParam = "query.log"
	}
	stream := strings.Contains(req.URL.Path, "/stream/")

	var file *os.File
	var e error
	var n int64

	logDir := ffdc.GetPath()
	if logDir == "" {
		return nil, errors.NewAdminLogError(fmt.Errorf("Couchbase log directory not set"))
	}

	fileName := filepath.Clean(filepath.Join(logDir, fileParam))

	if logDir != filepath.Dir(fileName) {
		return nil, errors.NewAdminLogError(fmt.Errorf("Attempt to access a file outside of Couchbase log directory"))
	}

	adt := make(map[string]interface{}, 2)
	adt["file"] = fileName
	adt["stream"] = stream
	af.Values = adt

	if file, e = os.Open(fileName); e != nil {
		return nil, errors.NewAdminLogError(e)
	}

	for {
		n, e = io.Copy(w, file)
		if e != nil && e != io.EOF {
			return nil, errors.NewAdminLogError(e)
		}
		if !stream {
			return textPlain(""), nil
		}
		if n == 0 && req.Context().Err() != nil {
			return textPlain(""), nil
		}
		pos, _ := file.Seek(0, os.SEEK_CUR)
		file.Close()
		file = nil
		time.Sleep(time.Second)
		if file, e = os.Open(fileName); e != nil {
			return nil, errors.NewAdminLogError(e)
		}
		if info, e := file.Stat(); e == nil && info.Size() >= pos {
			file.Seek(pos, os.SEEK_SET)
		}
	}
}

func doSequenceIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_SEQUENCES
	err, _ := endpoint.verifyCredentialsFromRequest("system:sequences", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	return sequences.ListCachedSequences(), nil
}

func doSequence(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, name := router.RequestValue(req, "name")
	af.EventTypeId = audit.API_ADMIN_SEQUENCES
	af.Name = name

	if req.Method == "GET" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:sequences", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		var res interface{}
		av, err := sequences.FetchSequence(name, true)
		if av != nil {
			res = av.Actual()
		}
		return res, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doMigration(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_ADMIN_MIGRATION
	err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_ADMIN, req, af)
	if err != nil {
		return nil, err
	}

	if req.Method == "DELETE" {
		res, err := datastore.AbortMigration()
		if err != nil {
			return textPlain(err.Error()), err
		}
		return textPlain(res), nil
	}
	return nil, errors.NewServiceErrorHttpMethod(req.Method)
}

func doSequenceRestore(v []byte, b string, include, exclude matcher, remap remapper) errors.Error {
	var oState json.KeyState
	json.SetKeyState(&oState, v)
	bidentity, err := oState.FindKey("identity")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing identity")
	}
	bstart, err := oState.FindKey("start")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing start")
	}
	binitial, err := oState.FindKey("initial")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing initial")
	}
	bcache, err := oState.FindKey("cache")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing cache")
	}
	bcycle, err := oState.FindKey("cycle")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing cycle")
	}
	bincrement, err := oState.FindKey("increment")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing increment")
	}
	bmax, err := oState.FindKey("max")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing max")
	}
	bmin, err := oState.FindKey("min")
	if err != nil {
		oState.Release()
		return errors.NewServiceErrorBadValue(err, "sequence restore: missing min")
	}
	oState.Release()
	name := strings.Trim(string(bidentity), "\"")
	if name == "" {
		return errors.NewServiceErrorBadValue(err, "sequence restore: name invalid")
	}
	path := algebra.ParsePath(name)
	start, err := strconv.ParseInt(strings.Trim(string(bstart), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: start invalid")
	}
	initial, err := strconv.ParseInt(strings.Trim(string(binitial), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: initial invalid")
	}
	cache, err := strconv.ParseUint(strings.Trim(string(bcache), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: cache invalid")
	}
	cycle, err := strconv.ParseBool(string(bcycle))
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: cycle invalid")
	}
	increment, err := strconv.ParseInt(strings.Trim(string(bincrement), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: increment invalid")
	}
	max, err := strconv.ParseInt(strings.Trim(string(bmax), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: max invalid")
	}
	min, err := strconv.ParseInt(strings.Trim(string(bmin), "\""), 10, 64)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "sequence restore: min invalid")
	}
	if filterEval(path, include, exclude) {
		remap.remap(b, path)
		m := make(map[string]interface{}, 2)
		m[sequences.OPT_CACHE] = cache
		m[sequences.OPT_CYCLE] = cycle
		m[sequences.OPT_START] = start
		m[sequences.OPT_INCR] = increment
		m[sequences.OPT_MAX] = max
		m[sequences.OPT_MIN] = min
		m[sequences.INTERNAL_OPT_INITIAL] = initial
		with := value.NewValue(m)
		p := algebra.NewPathFromElements(path)
		err1 := sequences.CreateSequence(p, with)
		if err1 != nil {
			if err1.Code() == errors.E_SEQUENCE_ALREADY_EXISTS {
				if sequences.DropSequence(p, true) == nil {
					err1 = sequences.CreateSequence(p, with)
				}
			}
			if err1 != nil && !err1.IsWarning() {
				return errors.NewServiceErrorBadValue(err1, "sequence restore")
			}
		}
	}
	return nil
}

var escQuote = regexp.MustCompile(`(\\\\)*(\\\")`)

func getCBORestoreKeyValue(v []byte, doValue bool) (string, []byte, error) {
	var oState json.KeyState
	json.SetKeyState(&oState, v)
	bkey, err := oState.FindKey("key")
	if err != nil {
		oState.Release()
		return "", nil, err
	}
	var bvalue []byte
	if doValue {
		bvalue, err = oState.FindKey("value")
		if err != nil {
			oState.Release()
			return string(bkey), nil, err
		}
	}
	oState.Release()

	var key string
	err = json.Unmarshal(bkey, &key)
	if err != nil || key == "" {
		return key, bvalue, err
	}

	// replace potential problematic document keys from previous backup/restore that contains
	// (pairs of) multiple escaped quotes with just quotes
	strIdx := escQuote.FindAllStringIndex(key, -1)
	if strIdx != nil && (len(strIdx)%2) == 0 {
		key = escQuote.ReplaceAllString(key, "\"")
	}

	return key, bvalue, nil
}

func getCBORestoreKeyspace(v []byte, b string, include, exclude matcher, remap remapper) (string, bool) {
	key, _, err := getCBORestoreKeyValue(v, false)
	if err != nil || key == "" {
		return "", false
	}
	parts := strings.Split(key, "::")
	path, _, _, _ := dictionary.GetCBOKeyspaceFromKey(parts[len(parts)-1])
	fullPath := "default:" + b + "." + path
	p := algebra.ParsePath(fullPath)
	if !filterEval(p, include, exclude) {
		return "", false
	}
	remap.remap(b, p)
	return algebra.NewPathFromElements(p).SimpleString(), true
}

func doCBORestore(v []byte, b string, include, exclude matcher, remap remapper, systemCollection datastore.Keyspace) errors.Error {
	key, bvalue, err := getCBORestoreKeyValue(v, true)
	if err != nil {
		if key == "" {
			return errors.NewServiceErrorBadValue(err, "cbo restore: missing key")
		} else if bvalue == nil {
			return errors.NewServiceErrorBadValue(err, "cbo restore: missing value")
		}
		return errors.NewServiceErrorBadValue(err, "cbo restore: invalid key or value")
	}
	if key == "" {
		return errors.NewServiceErrorBadValue(err, "cbo restore: invalid key")
	}
	if len(bvalue) < 3 || bvalue[0] != '"' || bvalue[len(bvalue)-1] != '"' {
		return errors.NewServiceErrorBadValue(err, "cbo restore: invalid value")
	}

	parts := strings.Split(key, "::")
	if len(parts) != 3 {
		return errors.NewServiceErrorBadValue(err, "cbo restore: invalid key")
	}
	path, _, _, _ := dictionary.GetCBOKeyspaceFromKey(parts[len(parts)-1])
	fullPath := "default:" + b + "." + path
	p := algebra.ParsePath(fullPath)
	if len(p) != 4 {
		return errors.NewServiceErrorBadValue(err, "cbo restore: invalid key")
	}
	if !filterEval(p, include, exclude) {
		return nil
	}

	remap.remap(b, p)
	key = strings.Replace(key, path, p[2]+"."+p[3], 1)
	uid, err := datastore.GetScopeUid(p[:3]...)
	if err != nil {
		return errors.NewServiceErrorBadValue(err, "cbo restore: error determining scope UID")
	}
	key = strings.Replace(key, parts[1], uid, 1)

	data := make([]byte, base64.StdEncoding.DecodedLen(len(bvalue)-2))
	n, err := base64.StdEncoding.Decode(data, []byte(bvalue[1:len(bvalue)-1]))
	raw, cerr := snappy.Decode(nil, data[:n])
	if cerr != nil {
		return errors.NewServiceErrorBadValue(cerr, "cbo restore: error decoding value"+":"+key)
	}

	pairs := make([]value.Pair, 1)
	pairs[0].Name = key
	pairs[0].Value = value.NewValue(raw)
	_, _, errs := systemCollection.Upsert(pairs, datastore.NULL_QUERY_CONTEXT, true)
	if errs != nil && len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func doCompletedRequestHistory(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	_, requestId := router.RequestValue(req, "request")

	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	af.Request = requestId

	if req.Method == "GET" {
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:completed_requests_history", auth.PRIV_SYSTEM_READ, req,
			af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:completed_requests, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}
		userName, err := endpoint.getImpersonate(req)
		if err != nil {
			return nil, err
		}

		i := strings.LastIndexByte(requestId, '-')
		if i == -1 {
			return nil, errors.NewServiceErrorBadValue(nil, "history request id")
		}
		fileNum, e := strconv.ParseUint(requestId[:i], 10, 64)
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(e, "history request id")
		}
		recNum, e := strconv.ParseUint(requestId[i+1:], 10, 64)
		if e != nil {
			return nil, errors.NewServiceErrorBadValue(e, "history request id")
		}
		var rv interface{}
		e = server.RequestsFileStreamRead(fileNum, recNum, 1, userName, func(m map[string]interface{}) bool {
			m["~file"] = fileNum
			rv = m
			return true
		})
		if e != nil && e != io.EOF {
			return nil, errors.NewServiceErrorBadValue(e, "history request")
		}
		return rv, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doCompletedHistoryIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (
	interface{}, errors.Error) {

	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests_history", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}
	userName, err := endpoint.getImpersonate(req)
	if err != nil {
		return nil, err
	}
	info := server.RequestsFileStreamFileInfo()
	if len(info) == 0 {
		return []string{}, nil
	}
	completed := make([]string, 0, len(info)/2)

	if userName == "" {
		for i := 0; i < len(info); i += 2 {
			completed = append(completed, fmt.Sprintf("%d-%d", info[i], info[i+1]))
		}
	} else {
		for i := 0; i < len(info); i += 2 {
			n := 0
			server.RequestsFileStreamRead(info[i], 0, 0, userName, func(m map[string]interface{}) bool { n++; return true })
			if n > 0 {
				completed = append(completed, fmt.Sprintf("%d-%d", info[i], n))
			}
		}
	}
	return completed, nil
}
