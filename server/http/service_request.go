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
	"net/http"
	"strconv"
	"strings"
	"time"

	json "github.com/couchbase/go_json"
	adt "github.com/couchbase/goutils/go-cbaudit"
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

	prefix string
	indent string

	elapsedTime   time.Duration
	executionTime time.Duration

	stmtCnt int
	consCnt int
}

var zeroScanVectorSource = &ZeroScanVectorSource{}

func (r *httpRequest) OriginalHttpRequest() *http.Request {
	return r.req
}

func newHttpRequest(rv *httpRequest, resp http.ResponseWriter, req *http.Request, bp BufferPool, size int) *httpRequest {
	var httpArgs httpRequestArgs
	var err errors.Error
	var urlArgs urlArgs
	var jsonArgs jsonArgs

	// This is literally when we become aware of the request
	reqTime := time.Now()

	// Limit body size in case of denial-of-service attack
	req.Body = http.MaxBytesReader(resp, req.Body, int64(size))

	e := req.ParseForm()
	if e != nil {
		err = errors.NewServiceErrorBadValue(go_errors.New("unable to parse form"), "request form")
	}

	if err != nil && req.Method != "GET" && req.Method != "POST" {
		err = errors.NewServiceErrorHTTPMethod(req.Method)
	}

	err = contentNegotiation(resp, req)

	if err == nil {
		httpArgs, err = getRequestParams(req, &urlArgs, &jsonArgs)
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
		var creds auth.Credentials

		rv.SetNamedArgs(httpArgs.getNamedArgs())
		creds, err = getCredentials(httpArgs, req.Header["Authorization"])
		if err == nil {
			rv.SetCredentials(creds)

			if rv.consCnt > 0 {
				err = getScanConfiguration(&rv.consistency, httpArgs)
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

	if err != nil {
		rv.Fail(err)
	}

	return rv
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

// stub for those parameters that have to be handled specially
func handleDummy(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	return nil
}

func handleStatement(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	statement, err := httpArgs.getStringVal(parm, val)
	if err == nil {
		rv.SetStatement(statement)
		rv.stmtCnt++
	}
	return err
}

func handlePrepared(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	var phaseTime time.Duration

	prepared_name, prepared, err := getPrepared(httpArgs, parm, val, &phaseTime)

	// MB-18841 (encoded_plan processing affects latency)
	// MB-19509 (encoded_plan may corrupt cache)
	// MB-19659 (spurious 4080 on multi node reprepare)
	// MB-27355 / MB-27778 (distrubute plans / deprecate encoded_plan)
	// If an encoded_plan has been supplied, only decode it
	// when the prepared statement can't be found, for backwards
	// compatibility with older SDKs
	if util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_ENCODED_PLAN) &&
		err != nil && err.Code() == errors.NO_SUCH_PREPARED {
		encoded_plan, plan_err := httpArgs.getString(ENCODED_PLAN, "")
		if plan_err == nil && encoded_plan != "" && encoded_plan != prepareds.EmptyPlan {
			var decoded_plan *plan.Prepared

			// Monitoring API: we only need to track the prepared
			// statement if we couldn't do it in getPrepared()
			// Distributed plans: we don't propagate on encoded_plan parameter
			// because the client will be using it across all nodes anyway, and
			// we want to avoid a plan distribution stampede
			decoded_plan, plan_err = prepareds.DecodePrepared(prepared_name, encoded_plan, (prepared == nil), false, &phaseTime)
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
	positionalArgs, err := httpArgs.getPositionalArgs()
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

func handleConsistency(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	rv.consCnt++
	return nil
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

func handleUseFts(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error {
	useFts, err := httpArgs.getTristateVal(parm, val)
	if err == nil {
		rv.SetUseFts(useFts == value.TRUE)
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

const ( // Request argument names
	MAX_PARALLELISM   = "max_parallelism"
	SCAN_CAP          = "scan_cap"
	PIPELINE_CAP      = "pipeline_cap"
	PIPELINE_BATCH    = "pipeline_batch"
	READONLY          = "readonly"
	METRICS           = "metrics"
	NAMESPACE         = "namespace"
	TIMEOUT           = "timeout"
	ARGS              = "args"
	PREPARED          = "prepared"
	ENCODED_PLAN      = "encoded_plan"
	STATEMENT         = "statement"
	FORMAT            = "format"
	ENCODING          = "encoding"
	COMPRESSION       = "compression"
	SIGNATURE         = "signature"
	PRETTY            = "pretty"
	SCAN_CONSISTENCY  = "scan_consistency"
	SCAN_WAIT         = "scan_wait"
	SCAN_VECTOR       = "scan_vector"
	SCAN_VECTORS      = "scan_vectors"
	CREDS             = "creds"
	CLIENT_CONTEXT_ID = "client_context_id"
	PROFILE           = "profile"
	CONTROLS          = "controls"
	N1QL_FEAT_CTRL    = "n1ql_feat_ctrl"
	MAX_INDEX_API     = "max_index_api"
	AUTO_PREPARE      = "auto_prepare"
	AUTO_EXECUTE      = "auto_execute"
	USE_FTS           = "use_fts"
)

var _PARAMETERS = map[string]func(rv *httpRequest, httpArgs httpRequestArgs, parm string, val interface{}) errors.Error{
	STATEMENT:         handleStatement,
	PREPARED:          handlePrepared,
	ENCODED_PLAN:      handleDummy,
	CREDS:             handleDummy,
	ARGS:              handlePositionalArgs,
	TIMEOUT:           handleTimeout,
	SCAN_CONSISTENCY:  handleConsistency,
	SCAN_WAIT:         handleConsistency,
	SCAN_VECTOR:       handleConsistency,
	SCAN_VECTORS:      handleConsistency,
	MAX_PARALLELISM:   handleMaxParallelism,
	SCAN_CAP:          handleScanCap,
	PIPELINE_CAP:      handlePipelineCap,
	PIPELINE_BATCH:    handlePipelineBatch,
	READONLY:          handleReadonly,
	METRICS:           handleMetrics,
	NAMESPACE:         handleNamespace,
	FORMAT:            handleFormat,
	ENCODING:          handleEncoding,
	COMPRESSION:       handleCompression,
	SIGNATURE:         handleSignature,
	PRETTY:            handlePretty,
	CLIENT_CONTEXT_ID: handleClientContextID,
	PROFILE:           handleProfile,
	CONTROLS:          handleControls,
	N1QL_FEAT_CTRL:    handleN1QLFeatCtrl,
	MAX_INDEX_API:     handleMaxIndexAPI,
	AUTO_PREPARE:      handleAutoPrepare,
	AUTO_EXECUTE:      handleAutoExecute,
	USE_FTS:           handleUseFts,
}

// take note while handling: initial parameters will not be found in fields or form values!
var _INITIAL_PARAMETERS = map[string]bool{
	N1QL_FEAT_CTRL: true,
	MAX_INDEX_API:  true,
}

func isValidParameter(a string) bool {
	a = util.TrimSpace(a)
	// Ignore empty (whitespace) parameters. They are harmless.
	if a == "" {
		return true
	}
	if strings.IndexRune(a, '$') == 0 {
		return true
	}

	_, ok := _PARAMETERS[a]
	return ok
}

func getPrepared(a httpRequestArgs, parm string, val interface{}, phaseTime *time.Duration) (string, *plan.Prepared, errors.Error) {
	prepared_field, err := a.getPreparedVal(parm, val)
	if err != nil || prepared_field == nil {
		return "", nil, err
	}

	prepared_name, ok := prepared_field.Actual().(string)
	if !ok {
		prepared_name = ""
	}

	// Monitoring API: track prepared statement access
	prepared, err := prepareds.GetPrepared(prepared_field, prepareds.OPT_TRACK|prepareds.OPT_REMOTE|prepareds.OPT_VERIFY, phaseTime)
	if err != nil || prepared == nil {
		return prepared_name, nil, err
	}

	return prepared_name, prepared, err
}

func getEmptyScanConfiguration(rv *scanConfigImpl) {
	rv.scan_level = newScanConsistency("NOT_BOUNDED")
	rv.scan_vector_source = zeroScanVectorSource
}

func getScanConfiguration(rv *scanConfigImpl, a httpRequestArgs) errors.Error {

	scan_consistency_field, err := a.getString(SCAN_CONSISTENCY, "NOT_BOUNDED")
	if err != nil {
		return err
	}

	scan_level := newScanConsistency(scan_consistency_field)
	if scan_level == server.UNDEFINED_CONSISTENCY {
		return errors.NewServiceErrorUnrecognizedValue(SCAN_CONSISTENCY, scan_consistency_field)
	}

	scan_wait, err := a.getDuration(SCAN_WAIT)
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
			defaultNamespace, err := a.getString(NAMESPACE, "default")
			if err != nil {
				return err
			}
			scan_vector_source = newMultipleScanVectorSource(defaultNamespace, scan_vectors)
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

func getCredentials(a httpRequestArgs, auths []string) (auth.Credentials, errors.Error) {
	// Cred_data retrieves credentials from either the URL parameters or from the body of the JSON request.
	cred_data, err := a.getCredentials()
	if err != nil {
		return nil, err
	}

	// Credentials can come from the cred_data, from the Basic authorization field
	// in  the request, both, or neither. If from both, the credentials are combined.
	// If neither, this function should return nil, nil.
	var creds auth.Credentials = nil

	if len(cred_data) > 0 {
		// Credentials are in request parameters:
		creds = auth.Credentials{}
		for _, cred := range cred_data {
			user, user_ok := cred["user"]
			pass, pass_ok := cred["pass"]
			if user_ok && pass_ok {
				creds[user] = pass
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
			if creds == nil {
				creds = auth.Credentials{}
			}
			switch len(u_details) {
			case 0, 1:
				// Authorization header format is incorrect
				return nil, errors.NewServiceErrorBadValue(nil, CREDS)
			case 2:
				creds[u_details[0]] = u_details[1]
			default:
				// Support passwords like "local:xxx" or "admin:xxx"
				creds[u_details[0]] = strings.Join(u_details[1:], ":")
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
	getString(string, string) (string, errors.Error)
	getStringVal(field string, v interface{}) (string, errors.Error)
	getTristateVal(field string, v interface{}) (value.Tristate, errors.Error)
	getPreparedVal(field string, v interface{}) (value.Value, errors.Error)
	getDuration(string) (time.Duration, errors.Error)
	getNamedArgs() map[string]value.Value
	getPositionalArgs() (value.Values, errors.Error)
	getCredentials() ([]map[string]string, errors.Error)
	getScanVector() (timestamp.Vector, errors.Error)
	getScanVectors() (map[string]timestamp.Vector, errors.Error)
}

// getRequestParams creates a httpRequestArgs implementation,
// depending on the content type in the request
func getRequestParams(req *http.Request, urlArgs *urlArgs, jsonArgs *jsonArgs) (httpRequestArgs, errors.Error) {

	const (
		URL_CONTENT  = "application/x-www-form-urlencoded"
		JSON_CONTENT = "application/json"
	)
	content_types := req.Header["Content-Type"]
	content_type := URL_CONTENT

	if len(content_types) > 0 {
		content_type = content_types[0]
	}

	if strings.HasPrefix(content_type, URL_CONTENT) {
		return newUrlArgs(req, urlArgs)
	}

	if strings.HasPrefix(content_type, JSON_CONTENT) {
		return newJsonArgs(req, jsonArgs)
	}

	return newUrlArgs(req, urlArgs)
}

// urlArgs is an implementation of httpRequestArgs that reads
// request arguments from a url-encoded http request
type urlArgs struct {
	req   *http.Request
	named map[string]value.Value
}

func newUrlArgs(req *http.Request, urlArgs *urlArgs) (*urlArgs, errors.Error) {
	var named map[string]value.Value

	for arg, val := range req.Form {
		newArg := util.TrimSpace(strings.ToLower(arg))
		if !isValidParameter(newArg) {
			return nil, errors.NewServiceErrorUnrecognizedParameter(newArg)
		}
		if newArg[0] == '$' {
			delete(req.Form, arg)
			switch len(val) {
			case 0:
				//This is an error - there _has_ to be a value for a named argument
				return nil, errors.NewServiceErrorMissingValue(fmt.Sprintf("named argument %s", arg))
			case 1:
				named = addNamedArg(named, util.TrimSpace(arg),
					value.NewValue([]byte(util.TrimSpace(val[0]))))
			default:
				return nil, errors.NewServiceErrorMultipleValues(arg)
			}
		} else if arg != newArg {
			delete(req.Form, arg)
			req.Form[newArg] = val
		}
	}
	if req.Form[STATEMENT] == nil && req.Method == "POST" {
		bytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), STATEMENT)
		}
		if len(bytes) > 0 {
			req.Form[STATEMENT] = []string{string(bytes)}
		}
	}

	urlArgs.req = req
	urlArgs.named = named
	return urlArgs, nil
}

func (this *urlArgs) processParameters(rv *httpRequest) errors.Error {
	var err errors.Error

	for parm, val := range this.req.Form {
		err = _PARAMETERS[parm](rv, this, parm, val)
		if err != nil {
			break
		}
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
func (this *urlArgs) getPositionalArgs() (value.Values, errors.Error) {
	var positionalArgs value.Values

	args_field, err := this.formValue(ARGS)
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
	scan_vector_data_field, err := this.formValue(SCAN_VECTOR)

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
	scan_vectors_data_field, err := this.formValue(SCAN_VECTORS)

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

func (this *urlArgs) getDuration(f string) (time.Duration, errors.Error) {
	var timeout time.Duration

	timeout_field, err := this.formValue(f)
	if err == nil && timeout_field != "" {
		timeout, err = newDuration(timeout_field)
	}
	return timeout, err
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

func (this *urlArgs) getString(f string, dflt string) (string, errors.Error) {
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

	creds_field, err := this.formValue(CREDS)
	if err == nil && creds_field != "" {
		e := json.Unmarshal([]byte(creds_field), &creds_data)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("unable to parse creds"), CREDS)
		}
	}
	return creds_data, err
}

func (this *urlArgs) getPreparedVal(field string, v interface{}) (value.Value, errors.Error) {
	var val value.Value

	value_field, err := this.getStringVal(field, v)
	if value_field == "" || err != nil {
		return val, err
	}

	// MB-25351: accept non quoted prepared names (much like the actual PREPARE statement)
	if value_field[0] != '"' {
		val = value.NewValue(value_field)
	} else {
		val = value.NewValue(strings.Trim(value_field, "\""))
	}

	return val, nil
}

func (this *urlArgs) getField(field string) []string {
	return this.req.Form[field]
}

func (this *urlArgs) formValue(field string) (string, errors.Error) {
	values := this.getField(field)

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
	args  map[string]interface{}
	named map[string]value.Value
	req   *http.Request
}

// create a jsonArgs structure from the given http request.
func newJsonArgs(req *http.Request, p *jsonArgs) (*jsonArgs, errors.Error) {
	decoder, e := getJsonDecoder(req.Body)
	if e != nil {
		return nil, e
	}
	err := decoder.Decode(&p.args)
	if err != nil {
		return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to parse JSON"), "JSON request body")
	}
	for arg, val := range p.args {
		newArg := util.TrimSpace(strings.ToLower(arg))
		if !isValidParameter(newArg) {
			return nil, errors.NewServiceErrorUnrecognizedParameter(newArg)
		}
		if newArg[0] == '$' {
			delete(p.args, arg)
			p.named = addNamedArg(p.named, arg, value.NewValue(val))
		} else if arg != newArg {
			delete(p.args, arg)
			p.args[newArg] = val
		}
	}
	p.req = req
	return p, nil
}

func (this *jsonArgs) processParameters(rv *httpRequest) errors.Error {
	var err errors.Error

	for parm, val := range this.args {
		err = _PARAMETERS[parm](rv, this, parm, val)
		if err != nil {
			break
		}
	}
	return err
}

func (this *jsonArgs) getField(field string) (interface{}, bool) {
	value, ok := this.args[field]
	return value, ok
}

func (this *jsonArgs) getNamedArgs() map[string]value.Value {
	return this.named
}

func (this *jsonArgs) getPositionalArgs() (value.Values, errors.Error) {
	var positionalArgs value.Values

	args_field, in_request := this.getField(ARGS)
	if !in_request {
		return positionalArgs, nil
	}

	args, type_ok := args_field.([]interface{})
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

func (this *jsonArgs) getCredentials() ([]map[string]string, errors.Error) {
	creds_field, in_request := this.getField(CREDS)
	if !in_request {
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
	scan_vectors_data, in_request := this.getField(SCAN_VECTORS)
	if !in_request {
		return nil, nil
	}
	return getScanVectorsFromJSON(scan_vectors_data)
}

func (this *jsonArgs) getScanVector() (timestamp.Vector, errors.Error) {
	scan_vector_data, in_request := this.getField(SCAN_VECTOR)
	if !in_request {
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

func (this *jsonArgs) getDuration(f string) (time.Duration, errors.Error) {
	var timeout time.Duration
	t, err := this.getString(f, "0s")
	if err != nil {
		return timeout, err
	}
	return newDuration(t)
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
func (this *jsonArgs) getString(f string, dflt string) (string, errors.Error) {
	value_field, in_request := this.getField(f)
	if !in_request {
		return dflt, nil
	}

	value, type_ok := value_field.(string)
	if !type_ok {
		return value, errors.NewServiceErrorTypeMismatch(f, "string")
	}
	return value, nil
}

func (this *jsonArgs) getPreparedVal(field string, v interface{}) (value.Value, errors.Error) {
	return value.NewValue(v), nil
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
		return datastore.UNBOUNDED
	}
	switch this.scan_level {
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

func (this *scanConfigImpl) ScanWait() time.Duration {
	return this.scan_wait
}

func (this *scanConfigImpl) ScanVectorSource() timestamp.ScanVectorSource {
	return this.scan_vector_source
}

func newScanConsistency(s string) server.ScanConsistency {
	switch strings.ToUpper(s) {
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
