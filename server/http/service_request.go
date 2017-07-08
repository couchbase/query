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
	"encoding/json"
	go_errors "errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/timestamp"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type httpRequest struct {
	server.BaseRequest
	resp            http.ResponseWriter
	req             *http.Request
	httpCloseNotify <-chan bool
	writer          responseDataManager
	httpRespCode    int
	resultCount     int
	resultSize      int
	errorCount      int
	warningCount    int
}

func (r *httpRequest) OriginalHttpRequest() *http.Request {
	return r.req
}

func newHttpRequest(resp http.ResponseWriter, req *http.Request, bp BufferPool, size int) *httpRequest {
	var httpArgs httpRequestArgs
	var err errors.Error

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
		httpArgs, err = getRequestParams(req)
	}

	var statement string
	if err == nil {
		statement, err = httpArgs.getStatement()
	}

	var prepared *plan.Prepared

	if err == nil {
		var decoded_plan *plan.Prepared
		var prepared_name string

		prepared_name, prepared, err = getPrepared(httpArgs)
		encoded_plan, plan_err := getEncodedPlan(httpArgs)

		// MB-18841 (encoded_plan processing affects latency)
		// MB-19509 (encoded_plan may corrupt cache)
		// MB-19659 (spurious 4080 on multi node reprepare)
		// If an encoded_plan has been supplied, only decode it
		// when the prepared statement can't be found or the plan
		// is different.
		// DecodePrepared() will make sure that the plan is only
		// updated if it matches the REST API encoded_plan
		// requirements.
		if encoded_plan != "" && plan_err == nil &&
			((err != nil && err.Code() == errors.NO_SUCH_PREPARED) ||
				(err == nil && prepared != nil &&
					prepared.MismatchingEncodedPlan(encoded_plan))) {

			// Monitoring API: we only need to track the prepared
			// statement if we couldn't do it in getPrepared()
			decoded_plan, plan_err = plan.DecodePrepared(prepared_name, encoded_plan, (prepared == nil))
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

	if err == nil && statement == "" && prepared == nil {
		err = errors.NewServiceErrorMissingValue("statement or prepared")
	}

	var namedArgs map[string]value.Value
	if err == nil {
		namedArgs, err = httpArgs.getNamedArgs()
	}

	var positionalArgs value.Values
	if err == nil {
		positionalArgs, err = httpArgs.getPositionalArgs()
	}

	var namespace string
	if err == nil {
		namespace, err = httpArgs.getString(NAMESPACE, "")
	}

	var timeout time.Duration
	if err == nil {
		timeout, err = httpArgs.getDuration(TIMEOUT)
	}

	var max_parallelism int
	if err == nil {
		var maxp string
		maxp, err = httpArgs.getString(MAX_PARALLELISM, "")
		if err == nil && maxp != "" {
			var e error
			max_parallelism, e = strconv.Atoi(maxp)
			if e != nil {
				err = errors.NewServiceErrorBadValue(go_errors.New("max_parallelism is invalid"), "max parallelism")
			}
		}
	}

	var scan_cap int64
	if err == nil {
		var scap string
		scap, err = httpArgs.getString(SCAN_CAP, "")
		if err == nil && scap != "" {
			var e error
			scan_cap, e = strconv.ParseInt(scap, 10, 64)
			if e != nil {
				err = errors.NewServiceErrorBadValue(go_errors.New("scan_cap is invalid"), "scan cap")
			}
		}
	}

	var pipeline_cap int64
	if err == nil {
		var pcap string
		pcap, err = httpArgs.getString(PIPELINE_CAP, "")
		if err == nil && pcap != "" {
			var e error
			pipeline_cap, e = strconv.ParseInt(pcap, 10, 64)
			if e != nil {
				err = errors.NewServiceErrorBadValue(go_errors.New("pipeline_cap is invalid"), "pipeline cap")
			}
		}
	}

	var pipeline_batch int
	if err == nil {
		var pbatch string
		pbatch, err = httpArgs.getString(PIPELINE_BATCH, "")
		if err == nil && pbatch != "" {
			var e error
			pipeline_batch, e = strconv.Atoi(pbatch)
			if e != nil {
				err = errors.NewServiceErrorBadValue(go_errors.New("pipeline_batch is invalid"), "pipeline batch")
			}
		}
	}

	var readonly value.Tristate
	if err == nil {
		readonly, err = getReadonly(httpArgs, req.Method == "GET")
	}

	var metrics value.Tristate
	if err == nil {
		metrics, err = httpArgs.getTristate(METRICS)
	}

	var format Format
	if err == nil {
		format, err = getFormat(httpArgs)
	}

	if err == nil && format != JSON {
		err = errors.NewServiceErrorNotImplemented("format", format.String())
	}

	var signature value.Tristate
	if err == nil {
		signature, err = httpArgs.getTristate(SIGNATURE)
	}

	var compression Compression
	if err == nil {
		compression, err = getCompression(httpArgs)
	}

	if err == nil && compression != NONE {
		err = errors.NewServiceErrorNotImplemented("compression", compression.String())
	}

	var encoding Encoding
	if err == nil {
		encoding, err = getEncoding(httpArgs)
	}

	if err == nil && encoding != UTF8 {
		err = errors.NewServiceErrorNotImplemented("encoding", encoding.String())
	}

	var pretty value.Tristate
	if err == nil {
		pretty, err = httpArgs.getTristate(PRETTY)
	}

	var consistency *scanConfigImpl

	if err == nil {
		consistency, err = getScanConfiguration(httpArgs)
	}

	var creds auth.Credentials
	if err == nil {
		creds, err = getCredentials(httpArgs, req.Header["Authorization"])
	}

	client_id := ""
	if err == nil {
		client_id, err = getClientID(httpArgs)
	}

	var controls value.Tristate
	if err == nil {
		controls, err = getControlsRequest(httpArgs)
	}

	userAgent := req.UserAgent()
	cbUserAgent := req.Header.Get("CB-User-Agent")
	if cbUserAgent != "" {
		userAgent = userAgent + " (" + cbUserAgent + ")"
	}
	base := server.NewBaseRequest(statement, prepared, namedArgs, positionalArgs,
		namespace, max_parallelism, scan_cap, pipeline_cap, pipeline_batch,
		readonly, metrics, signature, pretty, consistency, client_id, creds,
		req.RemoteAddr, userAgent)

	var prof server.Profile
	if err == nil {
		base.SetControls(controls)
		prof, err = getProfileRequest(httpArgs)
		if err == nil {
			base.SetProfile(prof)
		}
	}

	rv := &httpRequest{
		BaseRequest: *base,
		resp:        resp,
		req:         req,
	}

	rv.SetTimeout(timeout)

	rv.writer = NewBufferedWriter(rv, bp)

	// Abort if client closes connection; alternatively, return when request completes.
	rv.httpCloseNotify = resp.(http.CloseNotifier).CloseNotify()

	if err != nil {
		rv.Fail(err)
	}

	return rv
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
)

var _PARAMETERS = []string{
	STATEMENT,
	PREPARED,
	ENCODED_PLAN,
	CREDS,
	ARGS,
	TIMEOUT,
	SCAN_CONSISTENCY,
	SCAN_WAIT,
	SCAN_VECTOR,
	SCAN_VECTORS,
	MAX_PARALLELISM,
	SCAN_CAP,
	PIPELINE_CAP,
	PIPELINE_BATCH,
	READONLY,
	METRICS,
	NAMESPACE,
	FORMAT,
	ENCODING,
	COMPRESSION,
	SIGNATURE,
	PRETTY,
	CLIENT_CONTEXT_ID,
	PROFILE,
	CONTROLS,
}

func isValidParameter(a string) bool {
	a = strings.TrimSpace(a)
	// Ignore empty (whitespace) parameters. They are harmless.
	if a == "" {
		return true
	}
	if strings.IndexRune(a, '$') == 0 {
		return true
	}
	for _, p := range _PARAMETERS {
		if strings.EqualFold(p, a) {
			return true
		}
	}
	return false
}

func getPrepared(a httpRequestArgs) (string, *plan.Prepared, errors.Error) {
	prepared_field, err := a.getValue(PREPARED)
	if err != nil || prepared_field == nil {
		return "", nil, err
	}

	// Monitoring API: track prepared statement access
	prepared, err := plan.TrackPrepared(prepared_field)
	if err != nil || prepared == nil {
		return "", nil, err
	}

	prepared_name, ok := prepared_field.Actual().(string)
	if !ok {
		prepared_name = ""
	}
	return prepared_name, prepared, err
}

func getEncodedPlan(a httpRequestArgs) (string, errors.Error) {
	return a.getString(ENCODED_PLAN, "")
}

func getCompression(a httpRequestArgs) (Compression, errors.Error) {
	var compression Compression

	compression_field, err := a.getString(COMPRESSION, "NONE")
	if err == nil && compression_field != "" {
		compression = newCompression(compression_field)
		if compression == UNDEFINED_COMPRESSION {
			err = errors.NewServiceErrorUnrecognizedValue(COMPRESSION, compression_field)
		}
	}
	return compression, err
}

func getScanConfiguration(a httpRequestArgs) (*scanConfigImpl, errors.Error) {

	scan_consistency_field, err := a.getString(SCAN_CONSISTENCY, "NOT_BOUNDED")
	if err != nil {
		return nil, err
	}

	scan_level := newScanConsistency(scan_consistency_field)
	if scan_level == server.UNDEFINED_CONSISTENCY {
		return nil, errors.NewServiceErrorUnrecognizedValue(SCAN_CONSISTENCY, scan_consistency_field)
	}

	scan_wait, err := a.getDuration(SCAN_WAIT)
	if err != nil {
		return nil, err
	}

	scan_vector, err := a.getScanVector()
	if err != nil {
		return nil, err
	}

	scan_vectors, err := a.getScanVectors()
	if err != nil {
		return nil, err
	}

	var scan_vector_source timestamp.ScanVectorSource
	if scan_vector == nil {
		if scan_vectors == nil {
			if scan_level == server.AT_PLUS && scan_vector == nil && scan_vectors == nil {
				return nil, errors.NewServiceErrorMissingValue(SCAN_VECTOR)
			}
			scan_vector_source = &ZeroScanVectorSource{}
		} else {
			defaultNamespace, err := a.getString(NAMESPACE, "default")
			if err != nil {
				return nil, err
			}
			scan_vector_source = newMultipleScanVectorSource(defaultNamespace, scan_vectors)
		}
	} else {
		if scan_vectors == nil {
			scan_vector_source = &singleScanVectorSource{scan_vector: scan_vector}
		} else {
			// Not both scan_vector and scan_vectors.
			return nil, errors.NewServiceErrorMultipleValues("scan_vector and scan_vectors")
		}
	}

	return &scanConfigImpl{
		scan_level:         scan_level,
		scan_wait:          scan_wait,
		scan_vector_source: scan_vector_source,
	}, nil
}

func getEncoding(a httpRequestArgs) (Encoding, errors.Error) {
	var encoding Encoding

	encoding_field, err := a.getString(ENCODING, "UTF-8")
	if err == nil && encoding_field != "" {
		encoding = newEncoding(encoding_field)
		if encoding == UNDEFINED_ENCODING {
			err = errors.NewServiceErrorUnrecognizedValue(ENCODING, encoding_field)
		}
	}
	return encoding, err
}

func getFormat(a httpRequestArgs) (Format, errors.Error) {
	var format Format

	format_field, err := a.getString(FORMAT, "JSON")
	if err == nil && format_field != "" {
		format = newFormat(format_field)
		if format == UNDEFINED_FORMAT {
			err = errors.NewServiceErrorUnrecognizedValue(FORMAT, format_field)
		}
	}
	return format, err
}

func getReadonly(a httpRequestArgs, isGet bool) (value.Tristate, errors.Error) {
	readonly, err := a.getTristate(READONLY)
	if err == nil && isGet {
		switch readonly {
		case value.NONE:
			readonly = value.TRUE
		case value.FALSE:
			err = errors.NewServiceErrorReadonly(
				fmt.Sprintf("%s=false cannot be used with HTTP GET method.", READONLY))
		}
	}
	return readonly, err
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
func getClientID(a httpRequestArgs) (string, errors.Error) {
	client_id, err := a.getString(CLIENT_CONTEXT_ID, "")
	if err != nil {
		return client_id, err
	}
	if len(client_id) > MAX_CLIENTID {
		id_trunc := make([]byte, MAX_CLIENTID)
		copy(id_trunc[:], client_id)
		client_id = string(id_trunc)
	}
	l := len(client_id)
	for i := 0; i < l; i++ {
		switch client_id[i] {
		case '"':
		case '\\':
			return client_id, errors.NewServiceErrorClientID(client_id)
		default:
			continue
		}
	}
	return client_id, nil
}

func getProfile(a httpRequestArgs) (server.Profile, errors.Error) {
	profile, err := a.getString(PROFILE, "")
	if err == nil && profile != "" {
		prof, ok := server.ParseProfile(profile)
		if ok {
			return prof, nil
		} else {
			err = errors.NewServiceErrorUnrecognizedValue(PROFILE, profile)
		}

	}
	return server.ProfUnset, err
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
	getString(string, string) (string, errors.Error)
	getTristate(f string) (value.Tristate, errors.Error)
	getValue(field string) (value.Value, errors.Error)
	getDuration(string) (time.Duration, errors.Error)
	getNamedArgs() (map[string]value.Value, errors.Error)
	getPositionalArgs() (value.Values, errors.Error)
	getStatement() (string, errors.Error)
	getCredentials() ([]map[string]string, errors.Error)
	getScanVector() (timestamp.Vector, errors.Error)
	getScanVectors() (map[string]timestamp.Vector, errors.Error)
}

// getRequestParams creates a httpRequestArgs implementation,
// depending on the content type in the request
func getRequestParams(req *http.Request) (httpRequestArgs, errors.Error) {

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
		return newUrlArgs(req)
	}

	if strings.HasPrefix(content_type, JSON_CONTENT) {
		return newJsonArgs(req)
	}

	return newUrlArgs(req)
}

// urlArgs is an implementation of httpRequestArgs that reads
// request arguments from a url-encoded http request
type urlArgs struct {
	req *http.Request
}

func newUrlArgs(req *http.Request) (*urlArgs, errors.Error) {
	for arg, _ := range req.Form {
		if !isValidParameter(arg) {
			return nil, errors.NewServiceErrorUnrecognizedParameter(arg)
		}
	}
	return &urlArgs{req: req}, nil
}

func (this *urlArgs) getStatement() (string, errors.Error) {
	statement, err := this.formValue(STATEMENT)
	if err != nil {
		return "", err
	}

	if statement == "" && this.req.Method == "POST" {
		bytes, err := ioutil.ReadAll(this.req.Body)
		if err != nil {
			return "", errors.NewServiceErrorBadValue(go_errors.New("unable to read body of request"), STATEMENT)
		}

		statement = string(bytes)
	}

	return statement, nil
}

// A named argument is an argument of the form: $<identifier>=json_value
func (this *urlArgs) getNamedArgs() (map[string]value.Value, errors.Error) {
	var args map[string]value.Value

	for name, _ := range this.req.Form {
		if !strings.HasPrefix(name, "$") {
			continue
		}
		arg, err := this.formValue(name)
		if err != nil {
			return args, err
		}
		if len(arg) == 0 {
			//This is an error - there _has_ to be a value for a named argument
			return args, errors.NewServiceErrorMissingValue(fmt.Sprintf("named argument %s", name))
		}
		args = addNamedArg(args, name, value.NewValue([]byte(arg)))
	}
	return args, nil
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

	decoder, err := getJsonDecoder(strings.NewReader(args_field))
	if err != nil {
		return positionalArgs, err
	}
	e := decoder.Decode(&args)
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
	decoder, err := getJsonDecoder(strings.NewReader(scan_vector_data_field))
	if err != nil {
		return nil, err
	}
	e := decoder.Decode(&target)
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
	decoder, err := getJsonDecoder(strings.NewReader(scan_vectors_data_field))
	if err != nil {
		return nil, err
	}
	e := decoder.Decode(&target)
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

func (this *urlArgs) getString(f string, dflt string) (string, errors.Error) {
	value := dflt

	value_field, err := this.formValue(f)
	if err == nil && value_field != "" {
		value = value_field
	}
	return value, err
}

func (this *urlArgs) getTristate(f string) (value.Tristate, errors.Error) {
	tristate_value := value.NONE
	value_field, err := this.formValue(f)
	if err != nil {
		return tristate_value, err
	}
	if value_field == "" {
		return tristate_value, nil
	}
	bool_value, e := strconv.ParseBool(value_field)
	if e != nil {
		return tristate_value, errors.NewServiceErrorBadValue(go_errors.New("unable to parse tristate as a boolean"), f)
	}
	tristate_value = value.ToTristate(bool_value)
	return tristate_value, nil
}

func (this *urlArgs) getCredentials() ([]map[string]string, errors.Error) {
	var creds_data []map[string]string

	creds_field, err := this.formValue(CREDS)
	if err == nil && creds_field != "" {
		decoder, err := getJsonDecoder(strings.NewReader(creds_field))
		if err != nil {
			return creds_data, err
		}
		e := decoder.Decode(&creds_data)
		if e != nil {
			err = errors.NewServiceErrorBadValue(go_errors.New("unable to parse creds"), CREDS)
		}
	}
	return creds_data, err
}

func (this *urlArgs) getValue(field string) (value.Value, errors.Error) {
	var val value.Value
	value_field, err := this.getString(field, "")
	if err == nil && value_field != "" {
		val = value.NewValue([]byte(value_field))
	}
	return val, err
}

func (this *urlArgs) getField(field string) []string {
	for name, value := range this.req.Form {
		if strings.EqualFold(field, name) {
			return value
		}
	}
	return nil
}

func (this *urlArgs) formValue(field string) (string, errors.Error) {
	values := this.getField(field)

	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	default:
		return "", errors.NewServiceErrorMultipleValues(field)
	}
}

// jsonArgs is an implementation of httpRequestArgs that reads
// request arguments from a json-encoded http request
type jsonArgs struct {
	args map[string]interface{}
	req  *http.Request
}

// create a jsonArgs structure from the given http request.
func newJsonArgs(req *http.Request) (*jsonArgs, errors.Error) {
	var p jsonArgs
	decoder, e := getJsonDecoder(req.Body)
	if e != nil {
		return nil, e
	}
	err := decoder.Decode(&p.args)
	if err != nil {
		return nil, errors.NewServiceErrorBadValue(go_errors.New("unable to parse JSON"), "JSON request body")
	}
	for arg, _ := range p.args {
		if !isValidParameter(arg) {
			return nil, errors.NewServiceErrorUnrecognizedParameter(arg)
		}
	}
	p.req = req
	return &p, nil
}

func (this *jsonArgs) getField(field string) (interface{}, bool) {
	for name, value := range this.args {
		if strings.EqualFold(field, name) {
			return value, true
		}
	}
	return nil, false
}

func (this *jsonArgs) getStatement() (string, errors.Error) {
	return this.getString(STATEMENT, "")
}

func (this *jsonArgs) getNamedArgs() (map[string]value.Value, errors.Error) {
	var args map[string]value.Value
	for name, arg := range this.args {
		if !strings.HasPrefix(name, "$") {
			continue
		}
		args = addNamedArg(args, name, value.NewValue(arg))
	}
	return args, nil
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

func (this *jsonArgs) getTristate(f string) (value.Tristate, errors.Error) {
	value_tristate := value.NONE
	value_field, in_request := this.getField(f)
	if !in_request {
		return value_tristate, nil
	}

	b, type_ok := value_field.(bool)
	if !type_ok {
		return value_tristate, errors.NewServiceErrorTypeMismatch(f, "boolean")
	}

	value_tristate = value.ToTristate(b)
	return value_tristate, nil
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

func (this *jsonArgs) getValue(f string) (value.Value, errors.Error) {
	var val value.Value
	value_field, in_request := this.getField(f)
	if !in_request {
		return val, nil
	}

	val = value.NewValue(value_field)
	return val, nil
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
	sequenceNum, found := arr[0].(float64)
	if !found {
		return 0, "", errors.NewServiceErrorScanVectorBadSequenceNumber(arr[0])
	}
	if sequenceNum < 0.0 || sequenceNum > math.MaxUint64 || math.Floor(sequenceNum) != sequenceNum {
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
	// The '$' is trimmed from the argument name when added to args:
	args[strings.TrimPrefix(name, "$")] = arg
	return args
}

// helper function to create a time.Duration instance from a given string.
// There must be a unit - valid units are "ns", "us", "ms", "s", "m", "h"
func newDuration(s string) (duration time.Duration, err errors.Error) {
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
