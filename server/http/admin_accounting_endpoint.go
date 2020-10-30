//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	dictionary "github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/scheduler"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/transactions"
	"github.com/couchbase/query/value"
	"github.com/gorilla/mux"
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
)

func expvarsHandler(w http.ResponseWriter, req *http.Request) {
	// Do not audit directly.
	// Will be handled and audited by /admin/stats auditing.
	http.Redirect(w, req, accountingPrefix, http.StatusFound)
}

func (this *HttpEndpoint) registerAccountingHandlers() {
	statsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStats)
	}
	statHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doStat)
	}
	notFoundHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doNotFound)
	}
	vitalsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doVitals)
	}
	preparedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepared)
	}
	preparedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrepareds)
	}
	requestsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequests)
	}
	requestHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doActiveRequest)
	}
	completedsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequests)
	}
	completedHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedRequest)
	}
	preparedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPreparedIndex)
	}
	requestIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doRequestIndex)
	}
	completedIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doCompletedIndex)
	}
	functionsIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctionsIndex)
	}
	functionHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunction)
	}
	functionsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doFunctions)
	}
	dictionaryIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionaryIndex)
	}
	dictionaryEntryHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionaryEntry)
	}
	dictionaryHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doDictionary)
	}
	tasksIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTasksIndex)
	}
	taskHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTask)
	}

	tasksHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTasks)
	}

	prometheusLowHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doPrometheusLow)
	}
	prometheusHighHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doEmpty)
	}
	transactionsIndexHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransactionsIndex)
	}
	transactionHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransaction)
	}
	transactionsHandler := func(w http.ResponseWriter, req *http.Request) {
		this.wrapAPI(w, req, doTransactions)
	}
	routeMap := map[string]struct {
		handler handlerFunc
		methods []string
	}{
		accountingPrefix:                      {handler: statsHandler, methods: []string{"GET"}},
		accountingPrefix + "/{stat}":          {handler: statHandler, methods: []string{"GET", "DELETE"}},
		vitalsPrefix:                          {handler: vitalsHandler, methods: []string{"GET"}},
		preparedsPrefix:                       {handler: preparedsHandler, methods: []string{"GET"}},
		preparedsPrefix + "/{name}":           {handler: preparedHandler, methods: []string{"GET", "POST", "DELETE", "PUT"}},
		requestsPrefix:                        {handler: requestsHandler, methods: []string{"GET"}},
		requestsPrefix + "/{request}":         {handler: requestHandler, methods: []string{"GET", "POST", "DELETE"}},
		completedsPrefix:                      {handler: completedsHandler, methods: []string{"GET"}},
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
		indexesPrefix + "/function_cache":     {handler: functionsIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/dictionary_cache":   {handler: dictionaryIndexHandler, methods: []string{"GET"}},
		indexesPrefix + "/tasks_cache":        {handler: tasksIndexHandler, methods: []string{"GET"}},
		prometheusLow:                         {handler: prometheusLowHandler, methods: []string{"GET"}},
		prometheusHigh:                        {handler: prometheusHighHandler, methods: []string{"GET"}},
		indexesPrefix + "/transactions":       {handler: transactionsIndexHandler, methods: []string{"GET"}},
	}

	for route, h := range routeMap {
		this.mux.HandleFunc(route, h.handler).Methods(h.methods...)
	}

	this.mux.HandleFunc(expvarsRoute, expvarsHandler).Methods("GET")

	this.mux.NotFoundHandler = http.HandlerFunc(notFoundHandler)
}

func doStats(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

	switch req.Method {
	case "GET":
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

func doStat(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["stat"]
	acctStore := endpoint.server.AccountingStore()
	reg := acctStore.MetricRegistry()

	af.EventTypeId = audit.API_ADMIN_STATS
	af.Stat = name

	switch req.Method {
	case "GET":
		af.EventTypeId = audit.API_DO_NOT_AUDIT
		metric := reg.Get(name)
		if metric != nil {
			return getMetricData(metric), nil
		} else {
			return nil, nil
		}
	case "DELETE":
		return nil, reg.Unregister(name)
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doPrometheusLow(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
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

	return textPlain(""), nil
}

func doEmpty(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	err, _ := endpoint.verifyCredentialsFromRequest("", auth.PRIV_QUERY_STATS, req, nil)
	if err != nil {
		return nil, err
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	return textPlain(""), nil
}

func doNotFound(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	accounting.UpdateCounter(accounting.INVALID_REQUESTS)
	return nil, nil
}

func doVitals(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_VITALS
	switch req.Method {
	case "GET":
		acctStore := endpoint.server.AccountingStore()
		return acctStore.Vitals()
	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

// Credentials can come from two sources: the basic username/password
// from basic authorizatio, and from a "creds" value, which encodes
// in JSON an array of username/password pairs, like this:
//   [{"user":"foo", "pass":"foopass"}, {"user":"bar", "pass": "barpass"}]
func (endpoint *HttpEndpoint) getCredentialsFromRequest(ds datastore.Datastore, req *http.Request) (*auth.Credentials, errors.Error, bool) {
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
	creds_json := req.FormValue("creds")
	if creds_json != "" {
		cred_list := make([]map[string]string, 0, 2)
		err := json.Unmarshal([]byte(creds_json), &cred_list)
		if err != nil {
			return nil, errors.NewAdminCredsError(creds_json, err), false
		} else {
			for _, v := range cred_list {
				user, user_ok := v["user"]
				pass, pass_ok := v["pass"]
				if !user_ok || !pass_ok {
					return nil, errors.NewAdminCredsError(creds_json, nil), false
				}

				creds.Users[user] = pass
				if endpoint.internalUser == user {
					isInternal = true
				}
			}
		}
	}
	return creds, nil, isInternal
}

func (endpoint *HttpEndpoint) verifyCredentialsFromRequest(api string, priv auth.Privilege, req *http.Request, af *audit.ApiAuditFields) (errors.Error, bool) {
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
	_, err = ds.Authorize(privs, creds)
	return err, isInternal
}

func doPrepared(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	af.Name = name

	if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		err = prepareds.DeletePrepared(name)
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

		prepared, _ := prepareds.GetPrepared(name, nil)

		// nothing to do if the prepared is there and the plan matches
		if prepared != nil && !prepared.MismatchingEncodedPlan(string(body)) {
			return "", nil
		}
		_, err = prepareds.DecodePrepared(name, string(body))
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

		var itemMap map[string]interface{}

		prepareds.PreparedDo(name, func(entry *prepareds.CacheEntry) {
			itemMap = map[string]interface{}{
				"name":            entry.Prepared.Name(),
				"uses":            entry.Uses,
				"statement":       entry.Prepared.Text(),
				"indexApiVersion": entry.Prepared.IndexApiVersion(),
				"featureControls": entry.Prepared.FeatureControls(),
			}
			if entry.Prepared.QueryContext() != "" {
				itemMap["queryContext"] = entry.Prepared.QueryContext()
			}
			if entry.Prepared.EncodedPlan() != "" {
				itemMap["encoded_plan"] = entry.Prepared.EncodedPlan()
			}
			isks := entry.Prepared.IndexScanKeyspaces()
			if len(isks) > 0 {
				itemMap["indexScanKeyspaces"] = isks
			}
			txPrepards, txPlans := entry.Prepared.TxPrepared()
			if len(txPrepards) > 0 {
				itemMap["txPrepards"] = txPrepards
			}
			if req.Method == "POST" {
				itemMap["plan"] = entry.Prepared.Operator
				if len(txPlans) > 0 {
					itemMap["txPlans"] = txPlans
				}
			}

			// only give times for entries that have completed at least one execution
			if entry.Uses > 0 && entry.RequestTime > 0 {
				itemMap["lastUse"] = entry.LastUse.String()
				itemMap["avgElapsedTime"] = (time.Duration(entry.RequestTime) /
					time.Duration(entry.Uses)).String()
				itemMap["avgServiceTime"] = (time.Duration(entry.ServiceTime) /
					time.Duration(entry.Uses)).String()
				itemMap["minElapsedTime"] = time.Duration(entry.MinRequestTime).String()
				itemMap["minServiceTime"] = time.Duration(entry.MinServiceTime).String()
				itemMap["maxElapsedTime"] = time.Duration(entry.MaxRequestTime).String()
				itemMap["maxServiceTime"] = time.Duration(entry.MaxServiceTime).String()
			}
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doPrepareds(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	switch req.Method {
	case "GET":
		err, _ := endpoint.verifyCredentialsFromRequest("system:prepareds", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}

		numPrepareds := prepareds.CountPrepareds()
		data := make([]map[string]interface{}, numPrepareds)
		i := 0

		snapshot := func(name string, d *prepareds.CacheEntry) bool {

			// FIXME quick hack to avoid overruns
			if i >= numPrepareds {
				return false
			}
			data[i] = map[string]interface{}{}
			data[i]["name"] = d.Prepared.Name()
			if d.Prepared.QueryContext() != "" {
				data[i]["queryContext"] = d.Prepared.QueryContext()
			}
			if d.Prepared.EncodedPlan() != "" {
				data[i]["encoded_plan"] = d.Prepared.EncodedPlan()
			}
			isks := d.Prepared.IndexScanKeyspaces()
			if len(isks) > 0 {
				data[i]["indexScanKeyspaces"] = isks
			}
			txPrepards, _ := d.Prepared.TxPrepared()
			if len(txPrepards) > 0 {
				data[i]["txPrepards"] = txPrepards
			}
			data[i]["statement"] = d.Prepared.Text()
			data[i]["uses"] = d.Uses
			if d.Uses > 0 {
				data[i]["lastUse"] = d.LastUse.String()
			}
			i++
			return true
		}

		prepareds.PreparedsForeach(snapshot, nil)
		return data, nil

	default:
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunction(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

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

		var itemMap map[string]interface{}

		functions.FunctionDo(name, func(entry *functions.FunctionEntry) {
			itemMap = map[string]interface{}{
				"uses": entry.Uses,
			}
			entry.Signature(itemMap)
			entry.Body(itemMap)

			// only give times for entries that have completed at least one execution
			if entry.Uses > 0 && entry.ServiceTime > 0 {
				itemMap["lastUse"] = entry.LastUse.String()
				itemMap["avgServiceTime"] = (time.Duration(entry.ServiceTime) /
					time.Duration(entry.Uses)).String()
				itemMap["minServiceTime"] = time.Duration(entry.MinServiceTime).String()
				itemMap["maxServiceTime"] = time.Duration(entry.MaxServiceTime).String()
			}
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doFunctions(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
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
				data[i]["lastUse"] = d.LastUse.String()
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

func doDictionaryEntry(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

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

		var itemMap map[string]interface{}

		dictionary.DictCacheEntryDo(name, func(d interface{}) {
			entry := d.(dictionary.DictCacheEntry)

			itemMap = map[string]interface{}{}
			entry.Target(itemMap)
			entry.Dictionary(itemMap)
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doDictionary(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
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

func doTask(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	name := vars["name"]

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
		err, isInternal := endpoint.verifyCredentialsFromRequest("system:task_cache", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if isInternal {
			// Do not audit internal requests. They are an internal API used
			// only for queries to system:tasks_cache, and would cause too
			// many log messages to be generated.
			af.EventTypeId = audit.API_DO_NOT_AUDIT
		}

		var itemMap map[string]interface{}

		scheduler.TaskDo(name, func(entry *scheduler.TaskEntry) {
			itemMap = map[string]interface{}{
				"class":      entry.Class,
				"subClass":   entry.SubClass,
				"name":       entry.Name,
				"id":         entry.Id,
				"state":      entry.State,
				"submitTime": entry.PostTime.String(),
				"delay":      entry.Delay.String(),
			}
			if entry.Results != nil {
				itemMap["results"] = entry.Results
			}
			if entry.Errors != nil {
				itemMap["errors"] = entry.Errors
			}
			if !entry.StartTime.IsZero() {
				itemMap["startTime"] = entry.StartTime.String()
			}
			if !entry.EndTime.IsZero() {
				itemMap["stopTime"] = entry.EndTime.String()
			}
		})
		return itemMap, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func doTasks(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
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
			if d.Results != nil {
				data[i]["results"] = d.Results
			}
			if d.Errors != nil {
				data[i]["errors"] = d.Errors
			}
			data[i]["submitTime"] = d.PostTime.String()
			data[i]["delay"] = d.Delay.String()
			if !d.StartTime.IsZero() {
				data[i]["startTime"] = d.StartTime.String()
			}
			if !d.EndTime.IsZero() {
				data[i]["stopTime"] = d.EndTime.String()
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

func doTransaction(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request,
	af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	txId := vars["txid"]

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

		var itemMap map[string]interface{}
		transactions.TransactionEntryDo(txId, func(d interface{}) {
			entry := d.(*transactions.TranContext)
			itemMap = map[string]interface{}{}
			entry.Content(itemMap)
		})
		return itemMap, nil
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

func doActiveRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

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
		reqMap := activeRequestWorkHorse(endpoint, requestId, (req.Method == "POST"))

		return reqMap, nil
	} else if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		if endpoint.actives.Delete(requestId, true) {
			return nil, errors.NewServiceErrorHttpReq(requestId)
		}

		return true, nil
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func activeRequestWorkHorse(endpoint *HttpEndpoint, requestId string, profiling bool) map[string]interface{} {
	reqMap := map[string]interface{}{}
	_ = endpoint.actives.Get(requestId, func(request server.Request) {
		reqMap["requestId"] = request.Id().String()
		cId := request.ClientID().String()
		if cId != "" {
			reqMap["clientContextID"] = cId
		}
		if request.Statement() != "" {
			reqMap["statement"] = request.Statement()
		}
		if request.QueryContext() != "" {
			reqMap["queryContext"] = request.QueryContext()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			reqMap["preparedName"] = p.Name()
			reqMap["preparedText"] = p.Text()
		}
		if request.TxId() != "" {
			reqMap["txid"] = request.TxId()
		}
		reqMap["requestTime"] = request.RequestTime().Format(expression.DEFAULT_FORMAT)
		reqMap["elapsedTime"] = time.Since(request.RequestTime()).String()
		reqMap["executionTime"] = time.Since(request.ServiceTime()).String()
		reqMap["state"] = request.State().StateName()
		reqMap["scanConsistency"] = request.ScanConsistency()
		if request.UseFts() {
			reqMap["useFts"] = request.UseFts()
		}
		if request.UseCBO() {
			reqMap["useCBO"] = request.UseCBO()
		}

		p := request.Output().FmtPhaseCounts()
		if p != nil {
			reqMap["phaseCounts"] = p
		}
		p = request.Output().FmtPhaseOperators()
		if p != nil {
			reqMap["phaseOperators"] = p
		}
		p = request.Output().FmtPhaseTimes()
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
			t := request.GetTimings()

			// TODO - check lifetime of entry
			// by the time we marshal, is this still valid?
			if (prof == server.ProfOn || prof == server.ProfBench) && t != nil {
				reqMap["timings"] = t
				p = request.Output().FmtOptimizerEstimates(t)
				if p != nil {
					reqMap["optimizerEstimates"] = p
				}
			}

			var ctrl bool
			ctr := request.Controls()
			if ctr == value.NONE {
				ctrl = endpoint.server.Controls()
			} else {
				ctrl = (ctr == value.TRUE)
			}
			if ctrl {
				na := request.NamedArgs()
				if na != nil {
					reqMap["namedArgs"] = na
				}
				pa := request.PositionalArgs()
				if pa != nil {
					reqMap["positionalArgs"] = pa
				}
				memoryQuota := request.MemoryQuota()
				if memoryQuota != 0 {
					reqMap["memoryQuota"] = memoryQuota
				}
			}
		}
		credsString := datastore.CredsString(request.Credentials())
		if credsString != "" {
			reqMap["users"] = credsString
		}
		remoteAddr := request.RemoteAddr()
		if remoteAddr != "" {
			reqMap["remoteAddr"] = remoteAddr
		}
		userAgent := request.UserAgent()
		if userAgent != "" {
			reqMap["userAgent"] = userAgent
		}
	})
	return reqMap
}

func doActiveRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_PREPAREDS
	err, _ := endpoint.verifyCredentialsFromRequest("system:active_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}

	numRequests, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}

	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request server.Request) bool {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return false
		}
		requests[i] = map[string]interface{}{}
		requests[i]["requestId"] = request.Id().String()
		cId := request.ClientID().String()
		if cId != "" {
			requests[i]["clientContextID"] = cId
		}
		if request.Statement() != "" {
			requests[i]["statement"] = request.Statement()
		}
		if request.Prepared() != nil {
			p := request.Prepared()
			requests[i]["preparedName"] = p.Name()
			requests[i]["preparedStatement"] = p.Text()
		}
		if request.QueryContext() != "" {
			requests[i]["queryContext"] = request.QueryContext()
		}
		if request.TxId() != "" {
			requests[i]["txid"] = request.TxId()
		}
		requests[i]["requestTime"] = request.RequestTime().Format(expression.DEFAULT_FORMAT)
		requests[i]["elapsedTime"] = time.Since(request.RequestTime()).String()
		requests[i]["executionTime"] = time.Since(request.ServiceTime()).String()
		requests[i]["state"] = request.State().StateName()
		requests[i]["scanConsistency"] = request.ScanConsistency()

		credsString := datastore.CredsString(request.Credentials())
		if credsString != "" {
			requests[i]["users"] = credsString
		}

		p := request.Output().FmtPhaseCounts()
		if p != nil {
			requests[i]["phaseCounts"] = p
		}
		p = request.Output().FmtPhaseOperators()
		if p != nil {
			requests[i]["phaseOperators"] = p
		}
		p = request.Output().FmtPhaseTimes()
		if p != nil {
			requests[i]["phaseTimes"] = p
		}
		i++
		return true
	}
	endpoint.actives.ForEach(snapshot, nil)
	return requests, nil
}

func doCompletedRequest(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	vars := mux.Vars(req)
	requestId := vars["request"]

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
		reqMap := completedRequestWorkHorse(requestId, (req.Method == "POST"))
		return reqMap, nil
	} else if req.Method == "DELETE" {
		err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
		if err != nil {
			return nil, err
		}
		err = server.RequestDelete(requestId)
		if err != nil {
			return nil, err
		} else {
			return true, nil
		}
	} else {
		return nil, errors.NewServiceErrorHttpMethod(req.Method)
	}
}

func completedRequestWorkHorse(requestId string, profiling bool) map[string]interface{} {
	reqMap := map[string]interface{}{}
	server.RequestDo(requestId, func(request *server.RequestLogEntry) {
		reqMap["requestId"] = request.RequestId
		if request.ClientId != "" {
			reqMap["clientContextID"] = request.ClientId
		}
		reqMap["state"] = request.State
		reqMap["scanConsistency"] = request.ScanConsistency
		if request.UseFts {
			reqMap["useFts"] = request.UseFts
		}
		if request.UseCBO {
			reqMap["useCBO"] = request.UseCBO
		}
		if request.QueryContext != "" {
			reqMap["queryContext"] = request.QueryContext
		}
		if request.Statement != "" {
			reqMap["statement"] = request.Statement
		}
		if request.PreparedName != "" {
			reqMap["preparedName"] = request.PreparedName
			reqMap["preparedText"] = request.PreparedText
		}
		if request.TxId != "" {
			reqMap["txid"] = request.TxId
		}
		reqMap["requestTime"] = request.Time.Format(expression.DEFAULT_FORMAT)
		reqMap["elapsedTime"] = request.ElapsedTime.String()
		reqMap["serviceTime"] = request.ServiceTime.String()
		reqMap["resultCount"] = request.ResultCount
		reqMap["resultSize"] = request.ResultSize
		reqMap["errorCount"] = request.ErrorCount
		if request.Mutations != 0 {
			reqMap["mutations"] = request.Mutations
		}
		if request.PhaseCounts != nil {
			reqMap["phaseCounts"] = request.PhaseCounts
		}
		if request.PhaseOperators != nil {
			reqMap["phaseOperators"] = request.PhaseOperators
		}
		if request.PhaseTimes != nil {
			reqMap["phaseTimes"] = request.PhaseTimes
		}
		if request.UsedMemory != 0 {
			reqMap["usedMemory"] = request.UsedMemory
		}
		if request.Tag != "" {
			reqMap["~tag"] = request.Tag
		}

		if profiling {
			if request.NamedArgs != nil {
				reqMap["namedArgs"] = request.NamedArgs
			}
			if request.PositionalArgs != nil {
				reqMap["positionalArgs"] = request.PositionalArgs
			}
			if request.Timings != nil {
				reqMap["timings"] = request.Timings
				if request.OptEstimates != nil {
					reqMap["optimizerEstimates"] = request.OptEstimates
				}
			}
			if request.Errors != nil {
				errors := make([]map[string]interface{}, len(request.Errors))
				for i, e := range request.Errors {
					errors[i] = e.Object()
				}
				reqMap["errors"] = errors
			}
			if request.MemoryQuota != 0 {
				reqMap["memoryQuota"] = request.MemoryQuota
			}
		}
		if request.Users != "" {
			reqMap["users"] = request.Users
		}
		if request.RemoteAddr != "" {
			reqMap["remoteAddr"] = request.RemoteAddr
		}
		if request.UserAgent != "" {
			reqMap["userAgent"] = request.UserAgent
		}
	})
	return reqMap
}

func doCompletedRequests(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_ADMIN_COMPLETED_REQUESTS
	err, _ := endpoint.verifyCredentialsFromRequest("system:completed_requests", auth.PRIV_SYSTEM_READ, req, af)
	if err != nil {
		return nil, err
	}

	numRequests := server.RequestsCount()
	requests := make([]map[string]interface{}, numRequests)
	i := 0

	snapshot := func(requestId string, request *server.RequestLogEntry) bool {

		// FIXME quick hack to avoid overruns
		if i >= numRequests {
			return false
		}
		requests[i] = map[string]interface{}{}
		requests[i]["requestId"] = request.RequestId
		if request.ClientId != "" {
			requests[i]["clientContextID"] = request.ClientId
		}
		requests[i]["state"] = request.State
		requests[i]["scanConsistency"] = request.ScanConsistency
		if request.Statement != "" {
			requests[i]["statement"] = request.Statement
		}
		if request.QueryContext != "" {
			requests[i]["queryContext"] = request.QueryContext
		}
		if request.PreparedName != "" {
			requests[i]["preparedName"] = request.PreparedName
			requests[i]["preparedText"] = request.PreparedText
		}
		if request.TxId != "" {
			requests[i]["txid"] = request.TxId
		}
		requests[i]["requestTime"] = request.Time.Format(expression.DEFAULT_FORMAT)
		requests[i]["elapsedTime"] = request.ElapsedTime.String()
		requests[i]["serviceTime"] = request.ServiceTime.String()
		requests[i]["resultCount"] = request.ResultCount
		requests[i]["resultSize"] = request.ResultSize
		requests[i]["errorCount"] = request.ErrorCount
		if request.Mutations != 0 {
			requests[i]["mutations"] = request.Mutations
		}
		if request.PhaseCounts != nil {
			requests[i]["phaseCounts"] = request.PhaseCounts
		}
		if request.PhaseOperators != nil {
			requests[i]["phaseOperators"] = request.PhaseOperators
		}
		if request.PhaseTimes != nil {
			requests[i]["phaseTimes"] = request.PhaseTimes
		}
		if request.Users != "" {
			requests[i]["users"] = request.Users
		}
		if request.Tag != "" {
			requests[i]["~tag"] = request.Tag
		}

		// FIXME more stats
		i++
		return true
	}
	server.RequestsForeach(snapshot, nil)
	return requests, nil
}

func doPreparedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	return prepareds.NamePrepareds(), nil
}

func doRequestIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	numEntries, err := endpoint.actives.Count()
	if err != nil {
		return nil, err
	}
	requests := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request server.Request) bool {
		if i >= numEntries {
			requests = append(requests, requestId)
		} else {
			requests[i] = requestId
		}
		i++
		return true
	}
	endpoint.actives.ForEach(snapshot, nil)
	return requests, nil
}

func doCompletedIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	numEntries := server.RequestsCount()
	completed := make([]string, numEntries)
	i := 0
	snapshot := func(requestId string, request *server.RequestLogEntry) bool {
		if i >= numEntries {
			completed = append(completed, requestId)
		} else {
			completed[i] = requestId
		}
		i++
		return true
	}
	server.RequestsForeach(snapshot, nil)
	return completed, nil
}

func doFunctionsIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	return functions.NameFunctions(), nil
}

func doDictionaryIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	return dictionary.NameDictCacheEntries(), nil
}

func doTasksIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request, af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	return scheduler.NameTasks(), nil
}

func doTransactionsIndex(endpoint *HttpEndpoint, w http.ResponseWriter, req *http.Request,
	af *audit.ApiAuditFields) (interface{}, errors.Error) {
	af.EventTypeId = audit.API_DO_NOT_AUDIT
	return transactions.NameTransactions(), nil
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
