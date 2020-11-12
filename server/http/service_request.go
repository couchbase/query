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
	"encoding/base64"
	go_errors "errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/couchbase/go_json"
	adt "github.com/couchbase/goutils/go-cbaudit"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/prepareds"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type httpRequest struct {
	server.BaseRequest
	resp        http.ResponseWriter
	req         *http.Request
	consistency scanConfigImpl

	httpCloseNotify <-chan bool
	writer          bufferedWriter
	httpRespCode    int
	resultCount     int
	resultSize      int
	errorCount      int
	warningCount    int

	sync.WaitGroup
	prefix string
	indent string

	elapsedTime   time.Duration
	executionTime time.Duration

	stmtCnt  int
	consCnt  int
	jsonArgs jsonArgs // ESCAPE analysis workaround
	urlArgs  urlArgs  // ESCAPE analysis workaround
}

var zeroScanVectorSource = &ZeroScanVectorSource{}

func newHttpRequest(rv *httpRequest, resp http.ResponseWriter, req *http.Request, bp BufferPool, size int, namespace string) {
	var httpArgs httpRequestArgs
	var err errors.Error

	// This is literally when we become aware of the request
	reqTime := time.Now()

	// Limit body size in case of denial-of-service attack
	req.Body = http.MaxBytesReader(resp, req.Body, int64(size))

	if req.Method != "GET" && req.Method != "POST" {
		err = errors.NewServiceErrorHTTPMethod(req.Method)
	}

	err = contentNegotiation(resp, req)

	if err == nil {
		const (
			URL_CONTENT  = "application/x-www-form-urlencoded"
			JSON_CONTENT = "application/json"
		)
		content_types := req.Header["Content-Type"]
		content_type := URL_CONTENT

		if len(content_types) > 0 {
			content_type = content_types[0]
		}

		if strings.HasPrefix(content_type, JSON_CONTENT) {
			err = newJsonArgs(req, &rv.jsonArgs)
			httpArgs = &rv.jsonArgs
		} else {
			err = newUrlArgs(req, &rv.urlArgs)
			httpArgs = &rv.urlArgs
		}
	}

	rv.resp = resp
	rv.req = req
	server.NewBaseRequest(&rv.BaseRequest)
	rv.SetRequestTime(reqTime)

	// for GET method, only readonly access
	if req.Method == "GET" {
		rv.SetReadonly(value.TRUE)
	}

	userAgent := req.UserAgent()
	cbUserAgent := req.Header.Get("CB-User-Agent")
	if cbUserAgent != "" {
		userAgent = userAgent + " (" + cbUserAgent + ")"
	}
	rv.SetUserAgent(userAgent)
	rv.SetRemoteAddr(req.RemoteAddr)

	if err == nil {
		err = httpArgs.processParameters(rv)
	}

	if err == nil {
		if rv.stmtCnt == 0 {
			err = errors.NewServiceErrorMissingValue("statement or prepared")
		} else if rv.stmtCnt > 1 {
			err = errors.NewServiceErrorMultipleValues("statement and prepared")
		}
	}

	// handle parameters that can't be handled dynamically
	if err == nil {
		var creds = &auth.Credentials{}
		creds.Users = make(map[string]string, 0)

		rv.SetNamedArgs(httpArgs.getNamedArgs())
		creds, err = getCredentials(httpArgs, req.Header["Authorization"])

		if err == nil {
			creds.HttpRequest = req
			rv.SetCredentials(creds)

			if rv.consCnt > 0 {
				defaultNamespace := rv.Namespace()
				if defaultNamespace == "" {
					defaultNamespace = namespace
				}
				err = getScanConfiguration(rv.TxId(), &rv.consistency, httpArgs, defaultNamespace)
				if err == nil {
					rv.SetScanConfiguration(&rv.consistency)
				}
			} else {
				getEmptyScanConfiguration(&rv.consistency)
				rv.SetScanConfiguration(&rv.consistency)
			}
		}

	}

	NewBufferedWriter(&rv.writer, rv, bp)

	// Abort if client closes connection; alternatively, return when request completes.
	rv.httpCloseNotify = resp.(http.CloseNotifier).CloseNotify()

	// Prevent operator to send results until the prefix is done
	rv.Add(1)

	if err != nil {
		rv.Fail(err)
	}
	rv.jsonArgs = jsonArgs{}
	rv.urlArgs = urlArgs{}
}

// For audit.Auditable interface.
// Nodename is dependent on the protocol and not the base request
func (this *httpRequest) EventNodeName() string {
	ret := distributed.RemoteAccess().WhoAmI()
	if ret == "" {
		ret = "local_node"
	}
	return ret
}

func handleStatement(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	statement, err := httpArgs.getStringVal(parm, val)
	if err == nil {
		rv.SetStatement(statement)
		rv.stmtCnt++
	}
	return err
}

func handleEncodedPlan(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	return httpArgs.storeDirect(_ENCODED_PLAN, parm, val)
}

func handlePrepared(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var phaseTime time.Duration

	prepared_name, prepared, err := getPrepared(httpArgs, rv.QueryContext(), parm, val, &phaseTime)

	// MB-18841 (encoded_plan processing affects latency)
	// MB-19509 (encoded_plan may corrupt cache)
	// MB-19659 (spurious 4080 on multi node reprepare)
	// MB-27355 / MB-27778 (distrubute plans / deprecate encoded_plan)
	// If an encoded_plan has been supplied, only decode it
	// when the prepared statement can't be found, for backwards
	// compatibility with older SDKs
	if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_ENCODED_PLAN) &&
		err != nil && err.Code() == errors.NO_SUCH_PREPARED {
		encoded_plan, plan_err := httpArgs.getString(_ENCODED_PLAN, "")
		if plan_err == nil && encoded_plan != "" && encoded_plan != prepareds.EmptyPlan {
			var decoded_plan *plan.Prepared

			// Monitoring API: we only need to track the prepared
			// statement if we couldn't do it in getPrepared()
			decoded_plan, plan_err = prepareds.DecodePreparedWithContext(prepared_name, rv.QueryContext(), encoded_plan, (prepared == nil), &phaseTime)
			if plan_err != nil {
				err = plan_err
			} else if decoded_plan != nil {
				prepared = decoded_plan
				err = nil
			}
		}
		if err == nil && plan_err != nil {
			err = plan_err
		}
	}
	if prepared != nil {
		if phaseTime != 0 {
			rv.Output().AddPhaseTime(execution.REPREPARE, phaseTime)
		}
		rv.SetPrepared(prepared)
		rv.stmtCnt++
	}
	return err
}

func handlePositionalArgs(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	positionalArgs, err := httpArgs.getPositionalArgs(parm, val)
	if err == nil {
		rv.SetPositionalArgs(positionalArgs)
	}
	return err
}

func handleNamespace(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	namespace, err := httpArgs.getStringVal(parm, val)
	if err == nil {
		rv.SetNamespace(namespace)
	}
	return err
}

func handleTimeout(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var timeout time.Duration

	t, err := httpArgs.getStringVal(parm, val)
	if err == nil && t != "" {
		timeout, err = newDuration(t)
		if err == nil {
			rv.SetTimeout(timeout)
		}
	}
	return err
}

func handleMaxParallelism(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		max_parallelism, e := strconv.Atoi(param)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("max_parallelism is invalid"), "max parallelism")
		} else {
			rv.SetMaxParallelism(max_parallelism)
		}
	}
	return err
}

func handleScanCap(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		scan_cap, e := strconv.ParseInt(param, 10, 64)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("scan_cap is invalid"), "scan cap")
		} else {
			rv.SetScanCap(scan_cap)
		}
	}
	return err
}

func handlePipelineCap(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		var e error
		var pipeline_cap int64

		pipeline_cap, e = strconv.ParseInt(param, 10, 64)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("pipeline_cap is invalid"), "pipeline cap")
		} else {
			rv.SetPipelineCap(pipeline_cap)
		}
	}
	return err
}

func handlePipelineBatch(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		pipeline_batch, e := strconv.Atoi(param)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("pipeline_batch is invalid"), "pipeline batch")
		} else {
			rv.SetPipelineBatch(pipeline_batch)
		}
	}
	return err
}

func handleReadonly(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	readonly, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		if rv.req.Method == "GET" && readonly == value.FALSE {
			err = errors.NewServiceErrorReadonly(
				fmt.Sprintf("%s=false cannot be used with HTTP GET method.", READONLY))
		} else {
			rv.SetReadonly(readonly)
		}
	}
	return err
}

func handleMetrics(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	metrics, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetMetrics(metrics)
	}
	return err
}

func handleFormat(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	format_field, err := httpArgs.getStringVal(parm, val)
	if err == nil && format_field != "" {
		format := newFormat(format_field)
		if format == UNDEFINED_FORMAT {
			err = errors.NewServiceErrorUnrecognizedValue(FORMAT, format_field)
		} else if format != JSON {
			err = errors.NewServiceErrorNotImplemented(FORMAT, format_field)
		}
	}
	return err
}

func handleSignature(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	signature, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetSignature(signature)
	}
	return err
}

func handleCompression(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	compression_field, err := httpArgs.getStringVal(parm, val)
	if err == nil && compression_field != "" {
		compression := newCompression(compression_field)
		if compression == UNDEFINED_COMPRESSION {
			err = errors.NewServiceErrorUnrecognizedValue(COMPRESSION, compression_field)
		} else if compression != NONE {
			err = errors.NewServiceErrorNotImplemented(COMPRESSION, compression_field)
		}
	}
	return err
}

func handleEncoding(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	encoding_field, err := httpArgs.getStringVal(parm, val)
	if err == nil && encoding_field != "" {
		encoding := newEncoding(encoding_field)
		if encoding == UNDEFINED_ENCODING {
			err = errors.NewServiceErrorUnrecognizedValue(ENCODING, encoding_field)
		} else if encoding != UTF8 {
			err = errors.NewServiceErrorNotImplemented(ENCODING, encoding_field)
		}
	}
	return err
}

func handlePretty(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	pretty, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetPretty(pretty)
	}
	return err
}

func handleAutoPrepare(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	autoPrepare, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		if rv.AutoExecute() == value.TRUE {
			return errors.NewServiceErrorMultipleValues("auto_execute and auto_prepare")
		} else {
			rv.SetAutoPrepare(autoPrepare)
		}
	}
	return err
}

func handleAutoExecute(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	autoExecute, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		if rv.AutoPrepare() == value.TRUE {
			return errors.NewServiceErrorMultipleValues("auto_execute and auto_prepare")
		} else {
			rv.SetAutoExecute(autoExecute)
		}
	}
	return err
}

func handleCreds(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	return httpArgs.storeDirect(_CREDS, parm, val)
}

func handleConsistency(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	rv.consCnt++
	return httpArgs.storeDirect(_SCAN_CONSISTENCY, parm, val)
}

func handleScanWait(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	rv.consCnt++
	return httpArgs.storeDirect(_SCAN_WAIT, parm, val)
}

func handleScanVector(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	rv.consCnt++
	return httpArgs.storeDirect(_SCAN_VECTOR, parm, val)
}

func handleScanVectors(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	rv.consCnt++
	return httpArgs.storeDirect(_SCAN_VECTORS, parm, val)
}

func handleClientContextID(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	client_id, err := httpArgs.getStringVal(parm, val)
	if err != nil {
		return err
	}
	client_id, err = getClientID(client_id)
	if err == nil {
		rv.SetClientID(client_id)
	}
	return err
}

func handleControls(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	controls, err := getControlsRequest(httpArgs, parm, val)
	if err == nil {
		rv.SetControls(controls)
	}
	return err
}

func handleProfile(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	prof, err := getProfileRequest(httpArgs, parm, val)
	if err == nil {
		rv.SetProfile(prof)
	}
	return err
}

func handleN1QLFeatCtrl(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		n1qlFeatureControl, e := strconv.ParseUint(param, 0, 64)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("n1ql_feat_ctrl is invalid"), N1QL_FEAT_CTRL)
		} else {
			rv.SetFeatureControls(n1qlFeatureControl)
		}
	}
	return err
}

func handleMaxIndexAPI(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		indexApiVer, e := strconv.Atoi(param)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("max_index_api is invalid"), MAX_INDEX_API)
		} else {
			rv.SetIndexApiVersion(indexApiVer)
		}
	}
	return err
}

func handleQueryContext(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	queryContext, err := httpArgs.getStringVal(QUERY_CONTEXT, val)
	if err == nil {
		err = algebra.ValidateQueryContext(queryContext)
		if err == nil {
			rv.SetQueryContext(queryContext)
		}
	}
	return err
}

func handleUseFts(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	useFts, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetUseFts(useFts == value.TRUE)
	}
	return err
}

func handleMemoryQuota(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		memoryQuota, e := strconv.ParseUint(param, 0, 64)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("memory quota is invalid"), MEMORY_QUOTA)
		} else {
			rv.SetMemoryQuota(memoryQuota)
		}
	}
	return err
}

func handleUseCBO(rv *httpRequest, httpsArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	useCBO, err := httpsArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetUseCBO(useCBO == value.TRUE)
	}
	return err
}

func handleTxId(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	txid, err := httpArgs.getStringVal(parm, val)
	if err == nil {
		rv.SetTxId(txid)
	}
	return err
}

func handleTxImplicit(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	tximplicit, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetTxImplicit(tximplicit == value.TRUE)
	}

	return err
}

func handleTxStmtNum(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		txstmtnum, e := strconv.ParseInt(param, 10, 64)
		if e != nil || txstmtnum < 0 {
			err = errors.NewServiceErrorBadValue(fmt.Errorf("%s is invalid", parm), parm)
		} else {
			rv.SetTxStmtNum(txstmtnum)
		}
	}
	return err
}

func handleTxTimeout(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var timeout time.Duration

	t, err := httpArgs.getStringVal(parm, val)
	if err == nil && t != "" {
		timeout, err = newDuration(t)
		if err == nil {
			rv.SetTxTimeout(timeout)
		}
	}

	return err
}

func handleDurabilityLevel(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	dLevel, err := httpArgs.getStringVal(parm, val)

	if err == nil {
		if l := datastore.DurabilityNameToLevel(dLevel); l >= 0 {
			rv.SetDurabilityLevel(l)
		} else {
			err = errors.NewServiceErrorBadValue(fmt.Errorf("%s is invalid", parm), parm)
		}
	}

	return err
}

func handleDurabilityTimeout(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var timeout time.Duration

	t, err := httpArgs.getStringVal(parm, val)
	if err == nil && t != "" {
		timeout, err = newDuration(t)
		if err == nil {
			rv.SetDurabilityTimeout(timeout)
		}
	}
	return err
}

func handleTxData(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	txData, err := httpArgs.getTxData(parm, val)
	if err == nil && len(txData) > 0 {
		rv.SetTxData(txData)
	}
	return err
}

func handleKvTimeout(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var timeout time.Duration

	t, err := httpArgs.getStringVal(parm, val)
	if err == nil && t != "" {
		timeout, err = newDuration(t)
		if err == nil {
			rv.SetKvTimeout(timeout)
		}
	}

	return err
}

func handleAtrCollection(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	s, err := httpArgs.getStringVal(parm, val)
	if err == nil && s != "" {
		rv.SetAtrCollection(s)
	}

	return err
}

func handleNumAtrs(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	param, err := httpArgs.getStringVal(parm, val)
	if err == nil && param != "" {
		n, e := strconv.ParseUint(param, 0, 64)
		if e != nil || n <= 0 || n >= (1<<16) {
			err = errors.NewServiceErrorBadValue(go_errors.New("number of Atrs is invalid"), param)
		} else {
			rv.SetNumAtrs(int(n))
		}
	}
	return err
}

// For audit.Auditable interface.
func (this *httpRequest) ElapsedTime() time.Duration {
	return this.elapsedTime
}

// For audit.Auditable interface.
func (this *httpRequest) ExecutionTime() time.Duration {
	return this.executionTime
}

// For audit.Auditable interface.
func (this *httpRequest) EventResultCount() int {
	return this.resultCount
}

// For audit.Auditable interface.
func (this *httpRequest) EventResultSize() int {
	return this.resultSize
}

// For audit.Auditable interface.
func (this *httpRequest) EventErrorCount() int {
	return this.errorCount
}

// For audit.Auditable interface.
func (this *httpRequest) EventWarningCount() int {
	return this.warningCount
}

// For audit.Auditable interface.
func (this *httpRequest) EventStatus() string {
	state := this.State()
	if state == server.COMPLETED {
		if this.errorCount == 0 {
			state = server.SUCCESS
		} else {
			state = server.ERRORS
		}
	}

	return state.StateName()
}

// For audit.Auditable interface.
func (this *httpRequest) EventGenericFields() adt.GenericFields {
	return adt.GetAuditBasicFields(this.req)
}

// for audit.Auditable interface.
func (this *httpRequest) EventRemoteAddress() string {
	return this.req.RemoteAddr
}

// for audit.Auditable interface.
func (this *httpRequest) EventLocalAddress() string {
	ctx := this.req.Context()
	if ctx != nil {
		addr, ok := ctx.Value(http.LocalAddrContextKey).(net.Addr)
		if ok && addr != nil {
			return addr.String()
		}
	}
	return ""
}

const ( // Request argument names
	MAX_PARALLELISM    = "max_parallelism"
	SCAN_CAP           = "scan_cap"
	PIPELINE_CAP       = "pipeline_cap"
	PIPELINE_BATCH     = "pipeline_batch"
	READONLY           = "readonly"
	METRICS            = "metrics"
	NAMESPACE          = "namespace"
	TIMEOUT            = "timeout"
	ARGS               = "args"
	PREPARED           = "prepared"
	ENCODED_PLAN       = "encoded_plan"
	STATEMENT          = "statement"
	FORMAT             = "format"
	ENCODING           = "encoding"
	COMPRESSION        = "compression"
	SIGNATURE          = "signature"
	PRETTY             = "pretty"
	SCAN_CONSISTENCY   = "scan_consistency"
	SCAN_WAIT          = "scan_wait"
	SCAN_VECTOR        = "scan_vector"
	SCAN_VECTORS       = "scan_vectors"
	CREDS              = "creds"
	CLIENT_CONTEXT_ID  = "client_context_id"
	PROFILE            = "profile"
	CONTROLS           = "controls"
	N1QL_FEAT_CTRL     = "n1ql_feat_ctrl"
	MAX_INDEX_API      = "max_index_api"
	AUTO_PREPARE       = "auto_prepare"
	AUTO_EXECUTE       = "auto_execute"
	QUERY_CONTEXT      = "query_context"
	USE_FTS            = "use_fts"
	MEMORY_QUOTA       = "memory_quota"
	USE_CBO            = "use_cbo"
	TXID               = "txid"
	TXIMPLICIT         = "tximplicit"
	TXSTMTNUM          = "txstmtnum"
	TXTIMEOUT          = "txtimeout"
	TXDATA             = "txdata"
	DURABILITY_LEVEL   = "durability_level"
	DURABILITY_TIMEOUT = "durability_timeout"
	KVTIMEOUT          = "kvtimeout"
	ATRCOLLECTION      = "atrcollection"
	NUMATRS            = "numatrs"
)

type argHandler struct {
	fn      func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error
	initial bool
}

var _PARAMETERS = map[string]*argHandler{
	N1QL_FEAT_CTRL: {handleN1QLFeatCtrl, true},
	MAX_INDEX_API:  {handleMaxIndexAPI, true},
	QUERY_CONTEXT:  {handleQueryContext, true},
	NAMESPACE:      {handleNamespace, true},
	ENCODED_PLAN:   {handleEncodedPlan, true},
	TXID:           {handleTxId, true},

	STATEMENT:          {handleStatement, false},
	PREPARED:           {handlePrepared, false},
	CREDS:              {handleCreds, false},
	ARGS:               {handlePositionalArgs, false},
	TIMEOUT:            {handleTimeout, false},
	SCAN_CONSISTENCY:   {handleConsistency, false},
	SCAN_WAIT:          {handleScanWait, false},
	SCAN_VECTOR:        {handleScanVector, false},
	SCAN_VECTORS:       {handleScanVectors, false},
	MAX_PARALLELISM:    {handleMaxParallelism, false},
	SCAN_CAP:           {handleScanCap, false},
	PIPELINE_CAP:       {handlePipelineCap, false},
	PIPELINE_BATCH:     {handlePipelineBatch, false},
	READONLY:           {handleReadonly, false},
	METRICS:            {handleMetrics, false},
	FORMAT:             {handleFormat, false},
	ENCODING:           {handleEncoding, false},
	COMPRESSION:        {handleCompression, false},
	SIGNATURE:          {handleSignature, false},
	PRETTY:             {handlePretty, false},
	CLIENT_CONTEXT_ID:  {handleClientContextID, false},
	PROFILE:            {handleProfile, false},
	CONTROLS:           {handleControls, false},
	AUTO_PREPARE:       {handleAutoPrepare, false},
	AUTO_EXECUTE:       {handleAutoExecute, false},
	USE_FTS:            {handleUseFts, false},
	MEMORY_QUOTA:       {handleMemoryQuota, false},
	USE_CBO:            {handleUseCBO, false},
	TXIMPLICIT:         {handleTxImplicit, false},
	TXSTMTNUM:          {handleTxStmtNum, false},
	TXTIMEOUT:          {handleTxTimeout, false},
	TXDATA:             {handleTxData, false},
	DURABILITY_LEVEL:   {handleDurabilityLevel, false},
	DURABILITY_TIMEOUT: {handleDurabilityTimeout, false},
	KVTIMEOUT:          {handleKvTimeout, false},
	ATRCOLLECTION:      {handleAtrCollection, false},
	NUMATRS:            {handleNumAtrs, false},
}

// common storage for the httpArgs implementations
const (
	_ENCODED_PLAN = int(iota)
	_CREDS
	_SCAN_CONSISTENCY
	_SCAN_WAIT
	_SCAN_VECTOR
	_SCAN_VECTORS

	_DIRECT_MAX
)

type directAccess [_DIRECT_MAX]interface{}

type parmValue struct {
	name string
	val  interface{}
	fn   func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error
}

// this is a bit of a hack because golang doesn't allow map constants (can't size an array at compile time based on a map)
// and is a bit clumsy with statically allocated slice buffers
// remember to resize initialArray and argsArray approriately if _PARAMETERS changes
type parmList interface {
	add(name string, val interface{}, fn func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error)
	slice() []parmValue
}

type initialArray struct {
	count int
	parms [10]parmValue
}

func (ia *initialArray) slice() []parmValue {
	return ia.parms[0:ia.count]
}

func (ia *initialArray) add(name string, val interface{}, fn func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error) {
	ia.parms[ia.count].name = name
	ia.parms[ia.count].val = val
	ia.parms[ia.count].fn = fn
	ia.count++
}

type argsArray struct {
	count int
	parms [30]parmValue
}

func (aa *argsArray) slice() []parmValue {
	return aa.parms[0:aa.count]
}

func (aa *argsArray) add(name string, val interface{}, fn func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error) {
	aa.parms[aa.count].name = name
	aa.parms[aa.count].val = val
	aa.parms[aa.count].fn = fn
	aa.count++
}

func getPrepared(a httpRequestArgs, queryContext string, parm string, val interface{},
	phaseTime *time.Duration) (string, *plan.Prepared, errors.Error) {
	prepared_name, err := a.getPreparedName(parm, val)
	if err != nil || prepared_name == "" {
		return "", nil, err
	}

	// Monitoring API: track prepared statement access
	prepared, err := prepareds.GetPreparedWithContext(prepared_name, queryContext, nil,
		prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY, phaseTime)
	if err != nil || prepared == nil {
		return prepared_name, nil, err
	}

	return prepared_name, prepared, err
}

func getEmptyScanConfiguration(rv *scanConfigImpl) {
	rv.scan_level = newScanConsistency("NOT_SET")
	rv.scan_vector_source = zeroScanVectorSource
}

func getScanConfiguration(txId string, rv *scanConfigImpl, a httpRequestArgs, namespace string) errors.Error {
	scan_consistency_field, err := a.getString(_SCAN_CONSISTENCY, "NOT_SET")
	if err != nil {
		return err
	}

	scan_level := newScanConsistency(scan_consistency_field)
	if scan_level == server.UNDEFINED_CONSISTENCY {
		return errors.NewServiceErrorUnrecognizedValue(SCAN_CONSISTENCY, scan_consistency_field)
	}

	t, err := a.getString(_SCAN_WAIT, "0s")
	if err != nil {
		return err
	}
	scan_wait, err := newDuration(t)
	if err != nil {
		return err
	}

	scan_vector, err := a.getScanVector()
	if err != nil {
		return err
	}

	scan_vectors, err := a.getScanVectors()
	if err != nil {
		return err
	}

	var scan_vector_source timestamp.ScanVectorSource
	if scan_vector == nil {
		if scan_vectors == nil {
			if scan_level == server.AT_PLUS && scan_vector == nil && scan_vectors == nil {
				return errors.NewServiceErrorMissingValue(SCAN_VECTOR)
			}
			scan_vector_source = zeroScanVectorSource
		} else {
			scan_vector_source = newMultipleScanVectorSource(namespace, scan_vectors)
		}
	} else {
		if scan_vectors == nil {
			scan_vector_source = &singleScanVectorSource{scan_vector: scan_vector}
		} else {
			// Not both scan_vector and scan_vectors.
			return errors.NewServiceErrorMultipleValues("scan_vector and scan_vectors")
		}
	}

	rv.scan_level = scan_level
	rv.scan_wait = scan_wait
	rv.scan_vector_source = scan_vector_source
	return nil
}

func getCredentials(a httpRequestArgs, auths []string) (*auth.Credentials, errors.Error) {
	// Cred_data retrieves credentials from either the URL parameters or from the body of the JSON request.
	cred_data, err := a.getCredentials()
	if err != nil {
		return nil, err
	}

	// Credentials can come from the cred_data, from the Basic authorization field
	// in  the request, both, or neither. If from both, the credentials are combined.
	// If neither, this function should return nil, nil.
	var creds = &auth.Credentials{}
	creds.Users = make(map[string]string, 0)

	if len(cred_data) > 0 {
		// Credentials are in request parameters:
		for _, cred := range cred_data {
			user, user_ok := cred["user"]
			pass, pass_ok := cred["pass"]
			if user_ok && pass_ok {
				creds.Users[user] = pass
			} else {
				return nil, errors.NewServiceErrorMissingValue("user or pass")
			}
		}
	}

	if len(auths) > 0 {
		// Credentials are in http header:
		curAuth := auths[0]
		if strings.HasPrefix(curAuth, "Basic ") {
			encoded_creds := strings.Split(curAuth, " ")[1]
			decoded_creds, err := base64.StdEncoding.DecodeString(encoded_creds)
			if err != nil {
				return nil, errors.NewServiceErrorBadValue(go_errors.New("credentials not base64-encoded"), CREDS)
			}
			// Authorization header is in format "user:pass"
			// per http://tools.ietf.org/html/rfc1945#section-10.2
			u_details := strings.Split(string(decoded_creds), ":")
			if creds.Users == nil {
				creds = &auth.Credentials{}
				creds.Users = make(map[string]string, 0)
			}
			switch len(u_details) {
			case 0, 1:
				// Authorization header format is incorrect
				return nil, errors.NewServiceErrorBadValue(nil, CREDS)
			case 2:
				creds.Users[u_details[0]] = u_details[1]
			default:
				// Support passwords like "local:xxx" or "admin:xxx"
				creds.Users[u_details[0]] = strings.Join(u_details[1:], ":")
			}
		}
	}

	// If we have credentials from neither source, creds will be uninitialized, i.e. nil.
	return creds, nil
}

const MAX_CLIENTID = 64

// Ensure that client context id is no more than 64 characters.
// Also ensure that client context id does not contain characters that would
// break json syntax.
func getClientID(client_id string) (string, errors.Error) {
	if len(client_id) > MAX_CLIENTID {
		id_trunc := make([]byte, MAX_CLIENTID)
		copy(id_trunc[:], client_id)
		client_id = string(id_trunc)
	}
	l := len(client_id)
	for i := 0; i < l; i++ {
		switch client_id[i] {
		case '"', '\\':
			return client_id, errors.NewServiceErrorClientID(client_id)
		default:
			continue
		}
	}
	return client_id, nil
}

const acceptType = "application/json"
const versionTag = "version="
const version = acceptType + "; " + versionTag + util.VERSION

func contentNegotiation(resp http.ResponseWriter, req *http.Request) errors.Error {
	// set content type to current version
	resp.Header().Set("Content-Type", version)
	accept := req.Header["Accept"]
	// if no media type specified, default to current version
	if accept == nil || accept[0] == "*/*" {
		return nil
	}
	desiredContent := accept[0]
	// media type must be application/json at least
	if !strings.HasPrefix(desiredContent, acceptType) {
		return errors.NewServiceErrorMediaType(desiredContent)
	}
	versionIndex := strings.Index(desiredContent, versionTag)
	// no version specified, default to current version
	if versionIndex == -1 {
		return nil
	}
	// check if requested version is supported
	requestVersion := desiredContent[versionIndex+len(versionTag):]
	if requestVersion >= util.MIN_VERSION && requestVersion <= util.VERSION {
		resp.Header().Set("Content-Type", desiredContent)
		return nil
	}
	return errors.NewServiceErrorMediaType(desiredContent)
}

// httpRequestArgs is an interface for getting the arguments in a http request
type httpRequestArgs interface {
	processParameters(rv *httpRequest) errors.Error
	storeDirect(int, string, interface{}) errors.Error
	getString(int, string) (string, errors.Error)
	getStringVal(field string, v interface{}) (string, errors.Error)
	getTristateVal(field string, v interface{}) (value.Tristate, errors.Error)
	getPreparedName(field string, v interface{}) (string, errors.Error)
	getNamedArgs() map[string]value.Value
	getPositionalArgs(parm string, val interface{}) (value.Values, errors.Error)
	getCredentials() ([]map[string]string, errors.Error)
	getScanVector() (timestamp.Vector, errors.Error)
	getScanVectors() (map[string]timestamp.Vector, errors.Error)
	getTxData(parm string, val interface{}) ([]byte, errors.Error)
}

// urlArgs is an implementation of httpRequestArgs that reads
// request arguments from a url-encoded http request
type urlArgs struct {
	req     *http.Request
	initial initialArray
	named   map[string]value.Value
	direct  directAccess
}

func newUrlArgs(req *http.Request, urlArgs *urlArgs) errors.Error {
	var named map[string]value.Value

	e := req.ParseForm()
	if e != nil {
		return errors.NewServiceErrorBadValue(go_errors.New("unable to parse form"), "request form")
	}

	for arg, val := range req.Form {
		newArg := util.TrimSpace(arg)

		// ignore empty parameters
		if newArg == "" {
			delete(req.Form, arg)
			continue
		}
		if newArg[0] == '$' {
			delete(req.Form, arg)
			switch len(val) {
			case 0:
				//This is an error - there _has_ to be a value for a named argument
				return errors.NewServiceErrorMissingValue(fmt.Sprintf("named argument %s", arg))
			case 1:
				named = addNamedArg(named, newArg, value.NewValue([]byte(util.TrimSpace(val[0]))))
			default:
				return errors.NewServiceErrorMultipleValues(arg)
			}
			continue
		}

		lowerArg := strings.ToLower(newArg)
		pType := _PARAMETERS[lowerArg]
		if pType == nil {
			return errors.NewServiceErrorUnrecognizedParameter(lowerArg)
		} else if pType.initial {
			delete(req.Form, arg)
			urlArgs.initial.add(lowerArg, val, pType.fn)
		} else if arg != lowerArg {
			delete(req.Form, arg)
			req.Form[lowerArg] = val
		}
	}
	if req.Form[STATEMENT] == nil && req.Method == "POST" {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), STATEMENT)
		}
		if len(bytes) > 0 {
			req.Form[STATEMENT] = []string{string(bytes)}
		}
	}

	urlArgs.req = req
	urlArgs.named = named
	return nil
}

func (this *urlArgs) processParameters(rv *httpRequest) errors.Error {
	var err errors.Error

	// certain parameters need to be handled before all others (eg query_context), because processing others depends on them
	for _, p := range this.initial.slice() {
		err = p.fn(rv, this, p.name, p.val)
		if err != nil {
			return err
		}
	}
	for parm, val := range this.req.Form {
		err = _PARAMETERS[parm].fn(rv, this, parm, val)
		if err != nil {
			break
		}
	}
	return err
}

func (this *urlArgs) storeDirect(f int, parm string, val interface{}) errors.Error {
	s, err := this.checkFormValue(parm, val)
	if err == nil {
		this.direct[f] = s
	}
	return err
}

// A named argument is an argument of the form: $<identifier>=json_value
func (this *urlArgs) getNamedArgs() map[string]value.Value {
	return this.named
}

func getJsonDecoder(r io.Reader) (*json.Decoder, errors.Error) {
	if r == nil {
		return nil, errors.NewServiceErrorDecodeNil()
	}
	return json.NewDecoder(r), nil
}

// Positional args are of the form: args=json_list
func (this *urlArgs) getPositionalArgs(parm string, val interface{}) (value.Values, errors.Error) {
	var positionalArgs value.Values

	args_field, err := this.checkFormValue(parm, val)
	if err != nil || args_field == "" {
		return positionalArgs, err
	}

	var args []interface{}

	e := json.Unmarshal([]byte(args_field), &args)
	if e != nil {
		return positionalArgs, errors.NewServiceErrorBadValue(go_errors.New("unable to parse args parameter as array"), ARGS)
	}

	positionalArgs = make([]value.Value, len(args))

	// Put each element of args into positionalArgs
	for i, arg := range args {
		positionalArgs[i] = value.NewValue(arg)
	}

	return positionalArgs, nil
}

func (this *urlArgs) getTxData(parm string, val interface{}) ([]byte, errors.Error) {
	txData, err := this.checkFormValue(parm, val)
	if err != nil {
		return nil, err
	}

	var target interface{}
	err1 := json.Unmarshal([]byte(txData), &target)
	if err1 != nil {
		return nil, errors.NewServiceErrorBadValue(err1, TXDATA)
	}

	if rval, ok := target.(map[string]interface{}); ok {
		if err1 := txDataValidation(rval); err1 != nil {
			return nil, errors.NewServiceErrorBadValue(err1, TXDATA)
		}
	} else {
		return nil, errors.NewServiceErrorBadValue(go_errors.New("txdata is invalid"), TXDATA)
	}

	return []byte(txData), nil
}

// Note: This function has no receiver, which makes it easier to test.
// The "json" input parameter should be a parsed JSON object structure, the output of a decoder.
func getScanVectorsFromJSON(json interface{}) (map[string]timestamp.Vector, errors.Error) {
	bucketMap, ok := json.(map[string]interface{})
	if !ok {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTORS, "map of strings to vectors")
	}

	if len(bucketMap) == 0 {
		return nil, nil
	}
	out := make(map[string]timestamp.Vector)
	for k, v := range bucketMap {
		// Is it a sparse vector?
		mapVector, ok := v.(map[string]interface{})
		if ok {
			entries, err := makeSparseVector(mapVector)
			if err != nil {
				return nil, err
			}
			out[k] = entries
			continue
		}
		// Is it a full vector?
		arrayVector, ok := v.([]interface{})
		if ok {
			entries, err := makeFullVector(arrayVector)
			if err != nil {
				return nil, err
			}
			out[k] = entries
			continue
		}
		return nil, errors.NewServiceErrorTypeMismatch(k, "full or sparse vector")
	}

	return out, nil
}

// Note: This function has no receiver, which makes it easier to test.
// The "json" input parameter should be a parsed JSON object structure, the output of a decoder.
func getScanVectorFromJSON(json interface{}) (timestamp.Vector, errors.Error) {
	// Is it a sparse vector?
	mapVector, ok := json.(map[string]interface{})
	if ok {
		return makeSparseVector(mapVector)
	}

	// Is it a full vector?
	arrayVector, ok := json.([]interface{})
	if ok {
		return makeFullVector(arrayVector)
	}

	return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "full or sparse vector")
}

func (this *urlArgs) getScanVector() (timestamp.Vector, errors.Error) {
	scan_vector_data_field, err := this.formValue(_SCAN_VECTOR)

	if err != nil || scan_vector_data_field == "" {
		return nil, err
	}

	var target interface{}
	e := json.Unmarshal([]byte(scan_vector_data_field), &target)
	if e != nil {
		return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to parse scan vector"), SCAN_VECTOR)
	}

	return getScanVectorFromJSON(target)
}

func (this *urlArgs) getScanVectors() (map[string]timestamp.Vector, errors.Error) {
	scan_vectors_data_field, err := this.formValue(_SCAN_VECTORS)

	if err != nil || scan_vectors_data_field == "" {
		return nil, err
	}

	var target interface{}
	e := json.Unmarshal([]byte(scan_vectors_data_field), &target)
	if e != nil {
		return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to parse scan vectors"), SCAN_VECTORS)
	}

	return getScanVectorsFromJSON(target)
}

func (this *urlArgs) getStringVal(field string, v interface{}) (string, errors.Error) {
	values, ok := v.([]string)

	// this never happens
	if !ok {
		return "", errors.NewServiceErrorBadValue(go_errors.New("unexpected value type"), field)
	}
	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return util.TrimSpace(values[0]), nil
	default:
		return "", errors.NewServiceErrorMultipleValues(field)
	}
}

func (this *urlArgs) getString(f int, dflt string) (string, errors.Error) {
	value := dflt

	value_field, err := this.formValue(f)
	if err == nil && value_field != "" {
		value = value_field
	}
	return value, err
}

func (this *urlArgs) getTristateVal(field string, v interface{}) (value.Tristate, errors.Error) {
	tristate_value := value.NONE

	value_field, err := this.getStringVal(field, v)
	if err != nil {
		return tristate_value, err
	}
	if value_field == "" {
		return tristate_value, nil
	}
	bool_value, e := strconv.ParseBool(value_field)
	if e != nil {
		return tristate_value, errors.NewServiceErrorBadValue(go_errors.New("unable to parse tristate as a boolean"), field)
	}
	tristate_value = value.ToTristate(bool_value)
	return tristate_value, nil
}

func (this *urlArgs) getCredentials() ([]map[string]string, errors.Error) {
	var creds_data []map[string]string

	creds_field, err := this.formValue(_CREDS)
	if err == nil && creds_field != "" {
		e := json.Unmarshal([]byte(creds_field), &creds_data)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("unable to parse creds"), CREDS)
		}
	}
	return creds_data, err
}

func (this *urlArgs) getPreparedName(field string, v interface{}) (string, errors.Error) {
	value_field, err := this.getStringVal(field, v)
	if value_field == "" || err != nil {
		return "", err
	}

	// MB-25351: accept non quoted prepared names (much like the actual PREPARE statement)
	if value_field[0] == '"' {
		value_field = strings.Trim(value_field, "\"")
	}

	return value_field, nil
}

func (this *urlArgs) formValue(field int) (string, errors.Error) {
	v := this.direct[field]
	if v == nil {
		return "", nil
	}
	return v.(string), nil
}

func (this *urlArgs) checkFormValue(field string, val interface{}) (string, errors.Error) {
	values, ok := val.([]string)

	// this should never happen
	if !ok {
		return "", errors.NewServiceErrorBadValue(go_errors.New("unexpected value type"), field)
	}

	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return util.TrimSpace(values[0]), nil
	default:
		return "", errors.NewServiceErrorMultipleValues(field)
	}
}

// jsonArgs is an implementation of httpRequestArgs that reads
// request arguments from a json-encoded http request
type jsonArgs struct {
	initial initialArray
	args    argsArray
	named   map[string]value.Value
	direct  directAccess
	req     *http.Request
	state   json.ScanState // ESCAPE analysis workaround
}

// create a jsonArgs structure from the given http request.
func newJsonArgs(req *http.Request, p *jsonArgs) errors.Error {
	var bytes []byte
	var err error

	if req.Method == "POST" {
		bytes, err = ioutil.ReadAll(req.Body)
		if err != nil {
			return errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), "JSON request body")
		}
	}

	json.SetScanState(&p.state, bytes)
	for {
		key, err := p.state.ScanKeys()
		if err != nil {
			return errors.NewServiceErrorBadValue(err, "getting key")
		}
		if key == nil {
			break
		}

		val, err := p.state.NextUnmarshaledValue()
		if err != nil {
			return errors.NewServiceErrorBadValue(err, "getting value")
		}
		newArg := util.TrimSpace(string(key))

		// ignore empty parameters
		if newArg == "" {
			continue
		}
		if newArg[0] == '$' {
			p.named = addNamedArg(p.named, newArg, value.NewValue(val))
			continue
		}
		lowerArg := strings.ToLower(newArg)
		pType := _PARAMETERS[lowerArg]
		if pType == nil {
			return errors.NewServiceErrorUnrecognizedParameter(newArg)
		} else if pType.initial {
			p.initial.add(lowerArg, val, pType.fn)
		} else {
			p.args.add(lowerArg, val, pType.fn)
		}
	}
	p.state.Release()
	p.req = req
	return nil
}

func (this *jsonArgs) processParameters(rv *httpRequest) errors.Error {
	var err errors.Error

	// certain parameters need to be handled before all others (eg query_context), because processing others depends on them
	for _, p := range this.initial.slice() {
		err = p.fn(rv, this, p.name, p.val)
		if err != nil {
			return err
		}
	}
	for _, p := range this.args.slice() {
		err = p.fn(rv, this, p.name, p.val)
		if err != nil {
			break
		}
	}
	return err
}

func (this *jsonArgs) storeDirect(f int, parm string, val interface{}) errors.Error {
	if val == nil {
		return errors.NewServiceErrorBadValue(go_errors.New("value is not specified"), parm)
	}
	this.direct[f] = val
	return nil
}

func (this *jsonArgs) getNamedArgs() map[string]value.Value {
	return this.named
}

func (this *jsonArgs) getPositionalArgs(parm string, val interface{}) (value.Values, errors.Error) {
	var positionalArgs value.Values

	args, type_ok := val.([]interface{})
	if !type_ok {
		return positionalArgs, errors.NewServiceErrorTypeMismatch(ARGS, "array")
	}

	positionalArgs = make([]value.Value, len(args))
	// Put each element of args into positionalArgs
	for i, arg := range args {
		positionalArgs[i] = value.NewValue(arg)
	}

	return positionalArgs, nil
}

func (this *jsonArgs) getTxData(parm string, val interface{}) (txData []byte, err errors.Error) {
	var err1 error
	if rval, ok := val.(map[string]interface{}); ok {
		err1 = txDataValidation(rval)
		if err1 == nil {
			txData, err1 = json.Marshal(rval)
			if err1 == nil {
				return txData, nil
			}
		}
	} else {
		err1 = go_errors.New("txdata is invalid")
	}

	return nil, errors.NewServiceErrorBadValue(err1, TXDATA)
}

func (this *jsonArgs) getCredentials() ([]map[string]string, errors.Error) {
	creds_field := this.direct[_CREDS]
	if creds_field == nil {
		return nil, nil
	}

	creds_items, arr_ok := creds_field.([]interface{})
	if arr_ok {
		creds_data := make([]map[string]string, len(creds_items))
		for i, item := range creds_items {
			map_item, map_ok := item.(map[string]interface{})
			if map_ok {
				map_new := make(map[string]string, len(map_item))
				for k, v := range map_item {
					vs, v_ok := v.(string)
					if v_ok {
						map_new[k] = vs
					} else {
						return nil, errors.NewServiceErrorTypeMismatch(CREDS,
							"array of { user, pass }")
					}
				}
				creds_data[i] = map_new
			} else {
				return nil, errors.NewServiceErrorTypeMismatch(CREDS, "array of { user, pass }")
			}
		}
		return creds_data, nil
	}
	return nil, errors.NewServiceErrorTypeMismatch(CREDS, "array of { user, pass }")
}

func (this *jsonArgs) getScanVectors() (map[string]timestamp.Vector, errors.Error) {
	scan_vectors_data := this.direct[_SCAN_VECTORS]
	if scan_vectors_data == nil {
		return nil, nil
	}
	return getScanVectorsFromJSON(scan_vectors_data)
}

func (this *jsonArgs) getScanVector() (timestamp.Vector, errors.Error) {
	scan_vector_data := this.direct[_SCAN_VECTOR]
	if scan_vector_data == nil {
		return nil, nil
	}
	return getScanVectorFromJSON(scan_vector_data)
}

func makeVectorEntry(index int, args interface{}) (*scanVectorEntry, errors.Error) {
	data, is_map := args.(map[string]interface{})
	if !is_map {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "array or map of { number, string }")
	}
	value, has_value := data["value"]
	if !has_value {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "array or map of { number, string }")
	}
	value_val, is_number := value.(float64)
	if !is_number {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "array or map of { number, string }")
	}
	guard, has_guard := data["guard"]
	if !has_guard {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "array or map of { number, string }")
	}
	guard_val, guard_ok := guard.(string)
	if !guard_ok {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR, "array or map of { number, string }")
	}
	return &scanVectorEntry{
		position: uint32(index),
		value:    uint64(value_val),
		guard:    guard_val,
	}, nil
}

func (this *jsonArgs) getTristateVal(field string, v interface{}) (value.Tristate, errors.Error) {
	value_tristate := value.NONE

	b, type_ok := v.(bool)
	if !type_ok {
		return value_tristate, errors.NewServiceErrorTypeMismatch(field, "boolean")
	}

	value_tristate = value.ToTristate(b)
	return value_tristate, nil
}

// helper function to get a string type argument
func (this *jsonArgs) getStringVal(field string, v interface{}) (string, errors.Error) {
	value, type_ok := v.(string)
	if !type_ok {
		return value, errors.NewServiceErrorTypeMismatch(field, "string")
	}
	return value, nil
}

// helper function to get a string type argument
func (this *jsonArgs) getString(f int, dflt string) (string, errors.Error) {
	value_field := this.direct[f]
	if value_field == nil {
		return dflt, nil
	}

	return value_field.(string), nil
}

func (this *jsonArgs) getPreparedName(field string, v interface{}) (string, errors.Error) {
	switch v := v.(type) {
	case string:
		return v, nil
	default:
		return "", errors.NewUnrecognizedPreparedError(fmt.Errorf("Invalid prepared stmt %v", v))
	}
}

type Encoding int

const (
	UTF8 Encoding = iota
	UNDEFINED_ENCODING
)

func newEncoding(s string) Encoding {
	switch strings.ToUpper(s) {
	case "UTF-8":
		return UTF8
	default:
		return UNDEFINED_ENCODING
	}
}

func (e Encoding) String() string {
	var s string
	switch e {
	case UTF8:
		s = "UTF-8"
	default:
		s = "UNDEFINED_ENCODING"
	}
	return s
}

type Format int

const (
	JSON Format = iota
	XML
	CSV
	TSV
	UNDEFINED_FORMAT
)

func newFormat(s string) Format {
	switch strings.ToUpper(s) {
	case "JSON":
		return JSON
	case "XML":
		return XML
	case "CSV":
		return CSV
	case "TSV":
		return TSV
	default:
		return UNDEFINED_FORMAT
	}
}

func (f Format) String() string {
	var s string
	switch f {
	case JSON:
		s = "JSON"
	case XML:
		s = "XML"
	case CSV:
		s = "CSV"
	case TSV:
		s = "TSV"
	default:
		s = "UNDEFINED_FORMAT"
	}
	return s
}

type Compression int

const (
	NONE Compression = iota
	ZIP
	RLE
	LZMA
	LZO
	UNDEFINED_COMPRESSION
)

func newCompression(s string) Compression {
	switch strings.ToUpper(s) {
	case "NONE":
		return NONE
	case "ZIP":
		return ZIP
	case "RLE":
		return RLE
	case "LZMA":
		return LZMA
	case "LZO":
		return LZO
	default:
		return UNDEFINED_COMPRESSION
	}
}

func (c Compression) String() string {
	var s string
	switch c {
	case NONE:
		s = "NONE"
	case ZIP:
		s = "ZIP"
	case RLE:
		s = "RLE"
	case LZMA:
		s = "LZMA"
	case LZO:
		s = "LZO"
	default:
		s = "UNDEFINED_COMPRESSION"
	}
	return s
}

// scanVectorEntry implements timestamp.Entry
type scanVectorEntry struct {
	position uint32
	value    uint64
	guard    string
}

func (this *scanVectorEntry) Position() uint32 {
	return this.position
}

func (this *scanVectorEntry) Value() uint64 {
	return this.value
}

func (this *scanVectorEntry) Guard() string {
	return this.guard
}

// scanVectorEntries implements timestamp.Vector
type scanVectorEntries struct {
	entries []timestamp.Entry
}

func (this *scanVectorEntries) Entries() []timestamp.Entry {
	return this.entries
}

// A scan vector entry is an array of length two: [number, string] holding a sequence number and a guard string.
func extractValues(arr []interface{}) (uint64, string, errors.Error) {
	if len(arr) != 2 {
		return 0, "", errors.NewServiceErrorScanVectorBadLength(arr)
	}
	// JSON parser maps numbers to float64 or int64.
	// Only integers are valid here.
	sequenceNum, found := arr[0].(int64)
	if !found || sequenceNum < 0 {
		return 0, "", errors.NewServiceErrorScanVectorBadSequenceNumber(arr[0])
	}
	uuid, found := arr[1].(string)
	if !found {
		return 0, "", errors.NewServiceErrorScanVectorBadUUID(arr[1])
	}
	return uint64(sequenceNum), uuid, nil
}

func makeSparseVector(args map[string]interface{}) (*scanVectorEntries, errors.Error) {
	entries := make([]timestamp.Entry, len(args))
	i := 0
	for key, arg := range args {
		index, err := strconv.Atoi(key)
		if err != nil {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("Key value is not integer: "+key), SCAN_VECTOR)
		}
		array, ok := arg.([]interface{})
		if !ok {
			return nil, errors.NewServiceErrorTypeMismatch("scan vector entry", "two-element array")
		}
		sequenceNum, uuid, error := extractValues(array)
		if err != nil {
			return nil, error
		}
		entries[i] = &scanVectorEntry{
			position: uint32(index),
			value:    sequenceNum,
			guard:    uuid,
		}
		i = i + 1
	}
	return &scanVectorEntries{
		entries: entries,
	}, nil
}

func makeFullVector(args []interface{}) (*scanVectorEntries, errors.Error) {
	if len(args) != SCAN_VECTOR_SIZE {
		return nil, errors.NewServiceErrorTypeMismatch(SCAN_VECTOR,
			fmt.Sprintf("array of %d entries", SCAN_VECTOR_SIZE))
	}
	entries := make([]timestamp.Entry, len(args))
	for i, arg := range args {
		array, ok := arg.([]interface{})
		if !ok {
			return nil, errors.NewServiceErrorTypeMismatch("entry of a full scan vector",
				"array of length 2")
		}
		sequenceNum, uuid, err := extractValues(array)
		if err != nil {
			return nil, err
		}
		entries[i] = &scanVectorEntry{
			position: uint32(i),
			value:    sequenceNum,
			guard:    uuid,
		}
	}
	return &scanVectorEntries{
		entries: entries,
	}, nil
}

const SCAN_VECTOR_SIZE = 1024

type scanConfigImpl struct {
	scan_level         server.ScanConsistency
	scan_wait          time.Duration
	scan_vector_source timestamp.ScanVectorSource
}

func (this *scanConfigImpl) ScanConsistency() datastore.ScanConsistency {
	if this == nil {
		return datastore.NOT_SET
	}
	switch this.scan_level {
	case server.NOT_SET:
		return datastore.NOT_SET
	case server.NOT_BOUNDED:
		return datastore.UNBOUNDED
	case server.REQUEST_PLUS, server.STATEMENT_PLUS:
		return datastore.SCAN_PLUS
	case server.AT_PLUS:
		return datastore.AT_PLUS
	default:
		return datastore.UNBOUNDED
	}
}

func (this *scanConfigImpl) SetScanConsistency(consistency datastore.ScanConsistency) interface{} {
	if this == nil {
		this = &scanConfigImpl{}
	}
	switch consistency {
	case datastore.NOT_SET:
		this.scan_level = server.NOT_SET
	case datastore.UNBOUNDED:
		this.scan_level = server.NOT_BOUNDED
	case datastore.SCAN_PLUS:
		this.scan_level = server.REQUEST_PLUS
	case datastore.AT_PLUS:
		this.scan_level = server.AT_PLUS
	}

	return this
}

func (this *scanConfigImpl) ScanWait() time.Duration {
	return this.scan_wait
}

func (this *scanConfigImpl) ScanVectorSource() timestamp.ScanVectorSource {
	return this.scan_vector_source
}

func newScanConsistency(s string) server.ScanConsistency {
	switch strings.ToUpper(s) {
	case "NOT_SET":
		return server.NOT_SET
	case "NOT_BOUNDED":
		return server.NOT_BOUNDED
	case "REQUEST_PLUS":
		return server.REQUEST_PLUS
	case "STATEMENT_PLUS":
		return server.STATEMENT_PLUS
	case "AT_PLUS":
		return server.AT_PLUS
	default:
		return server.UNDEFINED_CONSISTENCY
	}
}

// addNamedArgs is used by getNamedArgs implementations to add a named argument
func addNamedArg(args map[string]value.Value, name string, arg value.Value) map[string]value.Value {
	if args == nil {
		args = make(map[string]value.Value)
	}
	name = util.TrimSpace(name)
	// The '$' is trimmed from the argument name when added to args:
	args[strings.TrimPrefix(name, "$")] = arg
	return args
}

// helper function to create a time.Duration instance from a given string.
// There must be a unit - valid units are "ns", "us", "ms", "s", "m", "h"
func newDuration(s string) (duration time.Duration, err errors.Error) {

	// handle empty REST parameters by returning a zero duration
	if s == "" {
		return
	}

	// Error if given string has no unit
	last_char := s[len(s)-1]
	if last_char != 's' && last_char != 'm' && last_char != 'h' {
		err = errors.NewServiceErrorBadValue(nil,
			fmt.Sprintf("duration value %s: missing or incorrect unit "+
				"(valid units: ns, us, ms, s, m, h)", s))
	}
	if err == nil {
		d, e := time.ParseDuration(s)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("Unable to parse duration: "+s), "duration")
		} else {
			duration = d
		}
	}
	return
}

/*
   txdata is internal SDK query parameter object. No way to prevent REST API by setting.
   So validate the fields loosely so that random fields are not allowed.
   This is mainly for restore the suspended transaction. SDK should not allowed to control the
   behavior of the transaction. Also used for SDK transformed KV operations. Those are part of N1QL.
   Any additional fields must carefully thought through, so that future N1QL functionality will not impact.
   At present this approach may break backward compatibility (i.e. If N1QL decided move commit protocol different
   approach it makes impossible because it required document access. But due to performance reason it has been decided
   to keep this way. One alternative is read the documents xattr and replay on new approach). At the same time
   any additions should be evaluated keep this in mind.
   opaque is SDKs opaque data. No backward compatibility will be provided.
   txnMeta is generated by transactional library, skip the validation.
   (state, timeLeftMillis, config, numAtrs, durabilityLevel, kvTimeoutMs) are temporary and will be removed once SDK moved to
   query parameters.
*/

var validTxDataFields = map[string]bool{"id": true, "txn": true, "atmpt": true, "atr": true,
	"mutations": true, "bkt": true, "scp": true, "coll": true, "cas": true, "type": true,
	"kv": true, "scas": true, "txnMeta": false, "opaque": false,
	"state": true, "timeLeftMillis": true, "config": true, "numAtrs": true, "durabilityLevel": true, "kvTimeoutMs": true}

func txDataValidation(tgt interface{}) (err error) {

	return nil

	if obj, ok := tgt.(map[string]interface{}); ok {
		for s, v := range obj {
			mv, ok := validTxDataFields[s]
			if !ok {
				return fmt.Errorf("%s field is not allowed", s)
			} else if ok && mv {
				if err = txDataValidation(v); err != nil {
					return err
				}
			}
		}
	} else if av, ok := tgt.([]interface{}); ok {
		for _, v := range av {
			if err = txDataValidation(v); err != nil {
				return err
			}
		}
	}

	return nil
}
