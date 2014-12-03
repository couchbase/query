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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/plan"
	"github.com/couchbaselabs/query/server"
	"github.com/couchbaselabs/query/value"
)

const MAX_REQUEST_BYTES = 1 << 20

type httpRequest struct {
	server.BaseRequest
	resp         http.ResponseWriter
	req          *http.Request
	writer       responseDataManager
	httpRespCode int
	resultCount  int
	resultSize   int
	errorCount   int
	warningCount int
}

func newHttpRequest(resp http.ResponseWriter, req *http.Request, bp BufferPool) *httpRequest {
	var httpArgs httpRequestArgs

	err := req.ParseForm()

	if req.Method != "GET" && req.Method != "POST" {
		err = fmt.Errorf("Unsupported http method: %s", req.Method)
	}

	if err == nil {
		httpArgs, err = getRequestParams(req)
	}

	var statement string
	if err == nil {
		statement, err = httpArgs.getStatement()
	}

	var prepared *plan.Prepared
	if err == nil {
		prepared, err = getPrepared(httpArgs)
	}

	if err == nil && statement == "" && prepared == nil {
		err = fmt.Errorf("Either statement or prepared must be provided.")
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
		timeout, err = httpArgs.getTimeDuration(TIMEOUT)
	}

	readonly := req.Method == "GET"
	if err == nil {
		readonly, err = getReadonly(httpArgs, readonly)
	}

	var metrics value.Tristate
	if err == nil {
		metrics, err = getMetrics(httpArgs)
	}

	var format Format
	if err == nil {
		format, err = getFormat(httpArgs)
	}

	if err == nil && format != JSON {
		err = fmt.Errorf("%s format not yet supported", format)
	}

	var signature bool
	if err == nil {
		signature, err = httpArgs.getBoolean(SIGNATURE, true)
	}

	var compression Compression
	if err == nil {
		compression, err = getCompression(httpArgs)
	}

	if err == nil && compression != NONE {
		err = fmt.Errorf("%s compression not yet supported", compression)
	}

	var encoding Encoding
	if err == nil {
		encoding, err = getEncoding(httpArgs)
	}

	if err == nil && encoding != UTF8 {
		err = fmt.Errorf("%s encoding not yet supported", encoding)
	}

	var pretty bool
	if err == nil {
		pretty, err = httpArgs.getBoolean(PRETTY, true)
	}

	if err == nil && !pretty {
		err = fmt.Errorf("%v pretty printing not yet supported", pretty)
	}

	var consistency *scanConfigImpl

	if err == nil {
		consistency, err = getScanConfiguration(httpArgs)

	}

	var creds datastore.Credentials
	if err == nil {
		creds, err = getCredentials(httpArgs, req.URL.User, req.Header["Authorization"])
	}

	client_id := ""
	if err == nil {
		client_id, err = httpArgs.getString(CLIENT_CONTEXT_ID, "")
	}

	base := server.NewBaseRequest(statement, prepared, namedArgs, positionalArgs,
		namespace, readonly, metrics, signature, consistency, client_id, creds)

	rv := &httpRequest{
		BaseRequest: *base,
		resp:        resp,
		req:         req,
	}

	rv.SetTimeout(rv, timeout)

	rv.writer = NewBufferedWriter(rv, bp)

	// Limit body size in case of denial-of-service attack
	req.Body = http.MaxBytesReader(resp, req.Body, MAX_REQUEST_BYTES)

	// Abort if client closes connection
	closeNotify := resp.(http.CloseNotifier).CloseNotify()
	go func() {
		<-closeNotify
		rv.Stop(server.TIMEOUT)
	}()

	if err != nil {
		rv.Fail(errors.NewError(err, ""))
	}

	return rv
}

const ( // Request argument names
	READONLY          = "readonly"
	METRICS           = "metrics"
	NAMESPACE         = "namespace"
	TIMEOUT           = "timeout"
	ARGS              = "args"
	PREPARED          = "prepared"
	STATEMENT         = "statement"
	FORMAT            = "format"
	ENCODING          = "encoding"
	COMPRESSION       = "compression"
	SIGNATURE         = "signature"
	PRETTY            = "pretty"
	SCAN_CONSISTENCY  = "scan_consistency"
	SCAN_WAIT         = "scan_wait"
	SCAN_VECTOR       = "scan_vector"
	CREDS             = "creds"
	CLIENT_CONTEXT_ID = "client_context_id"
)

func getPrepared(a httpRequestArgs) (*plan.Prepared, error) {
	var prepared *plan.Prepared

	prepared_field, err := a.getString(PREPARED, "")
	if err == nil && prepared_field != "" {
		// XXX TODO unmarshal
		prepared = nil
	}

	return prepared, err
}

func getReadonly(a httpRequestArgs, isGet bool) (bool, error) {
	readonly, err := a.getBoolean(READONLY, isGet)

	if err != nil && !readonly && isGet {
		readonly = true
		err = fmt.Errorf("%s=false cannot be used with HTTP GET method.", READONLY)
	}

	return readonly, err
}

func getMetrics(a httpRequestArgs) (value.Tristate, error) {
	var metrics value.Tristate
	m, err := a.getBoolean(METRICS, true)

	if err != nil {
		metrics = value.ToTristate(m)
	}

	return metrics, err

}

func getCompression(a httpRequestArgs) (Compression, error) {
	var compression Compression

	compression_field, err := a.getString(COMPRESSION, "NONE")
	if err == nil && compression_field != "" {
		compression = newCompression(compression_field)
		if compression == UNDEFINED_COMPRESSION {
			err = fmt.Errorf("Unknown %s value: %s", COMPRESSION, compression)
		}
	}

	return compression, err
}

func getScanConfiguration(a httpRequestArgs) (*scanConfigImpl, error) {
	var sc scanConfigImpl

	scan_consistency_field, err := a.getString(SCAN_CONSISTENCY, "NOT_BOUNDED")
	if err == nil {
		sc.scan_level = newScanConsistency(scan_consistency_field)
		if sc.scan_level == server.UNDEFINED_CONSISTENCY {
			err = fmt.Errorf("Unknown %s value: %s", SCAN_CONSISTENCY, scan_consistency_field)
		}
	}

	if err == nil {
		sc.scan_wait, err = a.getTimeDuration(SCAN_WAIT)
	}

	if err == nil {
		sc.scan_vector_full, sc.scan_vector_sparse, err = a.getScanVector()
	}

	if sc.scan_vector_full != nil && len(sc.scan_vector_full) != SCAN_VECTOR_SIZE {
		err = fmt.Errorf("%s parameter has to contain %d sequence numbers", SCAN_VECTOR, SCAN_VECTOR_SIZE)
	}

	if sc.scan_vector_sparse != nil && len(sc.scan_vector_sparse) < 1 {
		err = fmt.Errorf("%s parameter has to contain at least one sequence number", SCAN_VECTOR)
	}

	if sc.scan_level == server.AT_PLUS && sc.scan_vector_sparse == nil && sc.scan_vector_full == nil {
		err = fmt.Errorf("%s parameter value of AT_PLUS requires %s", SCAN_CONSISTENCY, SCAN_VECTOR)
	}

	return &sc, err
}

func getEncoding(a httpRequestArgs) (Encoding, error) {
	var encoding Encoding

	encoding_field, err := a.getString(ENCODING, "UTF-8")
	if err == nil && encoding_field != "" {
		encoding = newEncoding(encoding_field)
		if encoding == UNDEFINED_ENCODING {
			err = fmt.Errorf("Unknown %s value: %s", ENCODING, encoding)
		}
	}

	return encoding, err
}

func getFormat(a httpRequestArgs) (Format, error) {
	var format Format

	format_field, err := a.getString(FORMAT, "JSON")
	if err == nil && format_field != "" {
		format = newFormat(format_field)
		if format == UNDEFINED_FORMAT {
			err = fmt.Errorf("Unknown %s value: %s", FORMAT, format)
		}
	}

	return format, err
}

func getCredentials(a httpRequestArgs, hdrCreds *url.Userinfo, auths []string) ([]*url.Userinfo, error) {
	var cred_data []map[string]string
	var creds []*url.Userinfo
	var err error

	if hdrCreds != nil {
		// Credentials are in the request URL:
		creds = make([]*url.Userinfo, 1)
		creds[0] = hdrCreds
		return creds, err
	}

	if len(auths) > 0 {
		// Credentials are in the request header:
		// TODO: implement non-Basic auth (digest, ntlm)
		creds = make([]*url.Userinfo, 1)
		auth := auths[0]
		if strings.HasPrefix(auth, "Basic ") {
			var decoded_creds []byte
			encoded_creds := strings.Split(auth, " ")[1]
			decoded_creds, err = base64.StdEncoding.DecodeString(encoded_creds)
			if err == nil {
				// Authorization header is in format "user:pass"
				// per http://tools.ietf.org/html/rfc1945#section-10.2
				u_details := strings.Split(string(decoded_creds), ":")
				if len(u_details) == 2 {
					creds[0] = url.UserPassword(u_details[0], u_details[1])
				}
				if len(u_details) == 3 {
					// Support usernames like "local:xxx" or "admin:xxx"
					creds[0] = url.UserPassword(strings.Join(u_details[:2], ":"), u_details[2])
				}
			}
		}
		return creds, err
	}

	// Credentials may be in request arguments:
	cred_data, err = a.getCredentials()
	if err == nil && len(cred_data) > 0 {
		creds = make([]*url.Userinfo, len(cred_data))
		for i, cred := range cred_data {
			user, user_ok := cred["user"]
			pass, pass_ok := cred["pass"]
			if user_ok && pass_ok {
				creds[i] = url.UserPassword(user, pass)
			} else {
				err = fmt.Errorf("creds requires both user and pass")
				break
			}
		}
	}

	return creds, err
}

// httpRequestArgs is an interface for getting the arguments in a http request
type httpRequestArgs interface {
	getString(string, string) (string, error)
	getBoolean(string, bool) (bool, error)
	getTimeDuration(string) (time.Duration, error)
	getNamedArgs() (map[string]value.Value, error)
	getPositionalArgs() (value.Values, error)
	getStatement() (string, error)
	getCredentials() ([]map[string]string, error)
	getScanVector() ([]int, map[string]int, error)
}

// getRequestParams creates a httpRequestArgs implementation,
// depending on the content type in the request
func getRequestParams(req *http.Request) (httpRequestArgs, error) {

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
		return &urlArgs{req: req}, nil
	}

	if strings.HasPrefix(content_type, JSON_CONTENT) {
		return newJsonArgs(req)
	}

	return &urlArgs{req: req}, nil
}

// urlArgs is an implementation of httpRequestArgs that reads
// request arguments from a url-encoded http request
type urlArgs struct {
	req *http.Request
}

func (this *urlArgs) getStatement() (string, error) {
	statement, err := this.formValue(STATEMENT)
	if err != nil {
		return "", err
	}

	if statement == "" && this.req.Method == "POST" {
		bytes, err := ioutil.ReadAll(this.req.Body)
		if err != nil {
			return "", err
		}

		statement = string(bytes)
	}

	return statement, nil
}

// A named argument is an argument of the form: $<identifier>=json_value
func (this *urlArgs) getNamedArgs() (map[string]value.Value, error) {
	var namedArgs map[string]value.Value

	for namedArg, _ := range this.req.Form {
		if !strings.HasPrefix(namedArg, "$") {
			continue
		}
		argString, err := this.formValue(namedArg)
		if err != nil {
			return namedArgs, err
		}
		if len(argString) == 0 {
			//This is an error - there _has_ to be a value for a named argument
			return namedArgs, fmt.Errorf("Named argument %s must have a value", namedArg)
		}
		argValue := value.NewValue([]byte(argString))
		if namedArgs == nil {
			namedArgs = make(map[string]value.Value)
		}
		// NB the '$' is trimmed from the argument name when put in the Value map:
		namedArgs[strings.TrimPrefix(namedArg, "$")] = argValue
	}

	return namedArgs, nil
}

// Positional args are of the form: args=json_list
func (this *urlArgs) getPositionalArgs() (value.Values, error) {
	var positionalArgs value.Values

	args_field, err := this.formValue(ARGS)
	if err != nil || args_field == "" {
		return positionalArgs, err
	}

	var args []interface{}

	decoder := json.NewDecoder(strings.NewReader(args_field))
	err = decoder.Decode(&args)
	if err != nil {
		return positionalArgs, err
	}

	positionalArgs = make([]value.Value, len(args))
	// Put each element of args into positionalArgs
	for i, arg := range args {
		positionalArgs[i] = value.NewValue(arg)
	}

	return positionalArgs, nil
}

func (this *urlArgs) getScanVector() ([]int, map[string]int, error) {
	var full_vect []int
	var sparse_vect map[string]int

	scan_vect_field, err := this.formValue(SCAN_VECTOR)

	if err != nil || scan_vect_field == "" {
		return full_vect, sparse_vect, err
	}

	decoder := json.NewDecoder(strings.NewReader(scan_vect_field))
	err = decoder.Decode(&full_vect)
	if err == nil {
		return full_vect, sparse_vect, err
	}

	decoder = json.NewDecoder(strings.NewReader(scan_vect_field))
	err = decoder.Decode(&sparse_vect)

	return full_vect, sparse_vect, err
}

func (this *urlArgs) getTimeDuration(f string) (time.Duration, error) {
	var timeout time.Duration

	timeout_field, err := this.formValue(f)
	if err == nil && timeout_field != "" {
		timeout, err = time.ParseDuration(timeout_field)
	}

	return timeout, err
}

func (this *urlArgs) getString(f string, dflt string) (string, error) {
	value := dflt

	value_field, err := this.formValue(f)
	if err == nil && value_field != "" {
		value = value_field
	}

	return value, err
}

func (this *urlArgs) getBoolean(f string, dflt bool) (bool, error) {
	value := dflt

	value_field, err := this.formValue(f)
	if err == nil && value_field != "" {
		value, err = strconv.ParseBool(value_field)
	}

	return value, err
}

func (this *urlArgs) getCredentials() ([]map[string]string, error) {
	var creds_data []map[string]string

	creds_field, err := this.formValue(CREDS)
	if err == nil && creds_field != "" {
		decoder := json.NewDecoder(strings.NewReader(creds_field))
		err = decoder.Decode(&creds_data)
	}
	return creds_data, err
}

func (this *urlArgs) formValue(field string) (string, error) {
	values := this.req.Form[field]

	switch len(values) {
	case 0:
		return "", nil
	case 1:
		return values[0], nil
	default:
		return "", fmt.Errorf("Multiple values for field %s.", field)
	}
}

// jsonArgs is an implementation of httpRequestArgs that reads
// request arguments from a json-encoded http request
type jsonArgs struct {
	args map[string]interface{}
	req  *http.Request
}

// create a jsonArgs structure from the given http request.
func newJsonArgs(req *http.Request) (*jsonArgs, error) {
	var p jsonArgs
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&p.args)
	if err != nil {
		return nil, err
	}
	p.req = req
	return &p, nil
}

func (this *jsonArgs) getStatement() (string, error) {
	return this.getString(STATEMENT, "")
}

func (this *jsonArgs) getNamedArgs() (map[string]value.Value, error) {
	var namedArgs map[string]value.Value

	for namedArg, arg := range this.args {
		if strings.HasPrefix(namedArg, "$") {
			// Found a named argument - parse it into a value.Value
			argValue := value.NewValue(arg)
			if namedArgs == nil {
				namedArgs = make(map[string]value.Value)
			}
			namedArgs[namedArg] = argValue
		}
	}

	return namedArgs, nil
}

func (this *jsonArgs) getPositionalArgs() (value.Values, error) {
	var positionalArgs value.Values

	args_field, in_request := this.args[ARGS]

	if !in_request {
		return positionalArgs, nil
	}

	args, type_ok := args_field.([]interface{})

	if !type_ok {
		return positionalArgs, fmt.Errorf("%s parameter has to be an %s", ARGS, "array")
	}

	positionalArgs = make([]value.Value, len(args))
	// Put each element of args into positionalArgs
	for i, arg := range args {
		positionalArgs[i] = value.NewValue(arg)
	}

	return positionalArgs, nil
}

func (this *jsonArgs) getCredentials() ([]map[string]string, error) {
	var creds_data []map[string]string

	creds_field, in_request := this.args[CREDS]

	if !in_request {
		return creds_data, nil
	}

	creds_data, type_ok := creds_field.([]map[string]string)

	if !type_ok {
		return creds_data, fmt.Errorf("%s parameter has to be an %s", CREDS, "array of { user, pass }")
	}

	return creds_data, nil
}

func (this *jsonArgs) getScanVector() (full_vect []int, sparse_vect map[string]int, err error) {
	var type_ok bool

	scan_vect_field, in_request := this.args[SCAN_VECTOR]

	if !in_request {
		return
	}

	full_vect, type_ok = scan_vect_field.([]int)

	if type_ok {
		return
	}

	sparse_vect_i, type_ok := scan_vect_field.(map[string]interface{})

	if !type_ok {
		err = fmt.Errorf("%s parameter - format not recognised", SCAN_VECTOR)
	}

	// the json library marshals numbers to float64 - need to convert these to ints:
	sparse_vect = make(map[string]int, len(sparse_vect_i))

	for vbuck_no, seq_no := range sparse_vect_i {
		switch seq_no := seq_no.(type) {
		case float64:
			sparse_vect[vbuck_no] = int(seq_no)
		default:
			err = fmt.Errorf("%s parameter - invalid format for sequence number: %v", SCAN_VECTOR, seq_no)
			break
		}
	}
	return
}

func (this *jsonArgs) getTimeDuration(f string) (time.Duration, error) {
	var timeout time.Duration

	t, err := this.getString(f, "0")

	if err != nil {
		timeout, err = time.ParseDuration(t)
	}

	return timeout, err
}

// helper function to get a boolean typed argument
func (this *jsonArgs) getBoolean(f string, dflt bool) (bool, error) {
	value := dflt

	value_field, in_request := this.args[f]

	if !in_request {
		return value, nil
	}

	b, type_ok := value_field.(bool)

	if !type_ok {
		return value, fmt.Errorf("%s parameter has to be a %s", f, "boolean")
	}

	value = b

	return value, nil
}

// helper function to get a string type argument
func (this *jsonArgs) getString(f string, dflt string) (string, error) {
	value := dflt

	value_field, in_request := this.args[f]

	if !in_request {
		return value, nil
	}

	s, type_ok := value_field.(string)

	if !type_ok {
		return value, fmt.Errorf("%s has to be a %s", f, "string")
	}

	value = s

	return s, nil
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

const SCAN_VECTOR_SIZE = 1024

type scanConfigImpl struct {
	scan_level         server.ScanConsistency
	scan_wait          time.Duration
	scan_vector_full   []int
	scan_vector_sparse map[string]int
}

func (this *scanConfigImpl) ScanConsistency() server.ScanConsistency {
	return this.scan_level
}

func (this *scanConfigImpl) ScanWait() time.Duration {
	return this.scan_wait
}

func (this *scanConfigImpl) ScanVectorFull() []int {
	return this.scan_vector_full
}

func (this *scanConfigImpl) ScanVectorSparse() map[string]int {
	return this.scan_vector_sparse
}

func (this *scanConfigImpl) String() string {
	return fmt.Sprintf("scan_level: %d, scan_wait: %s, scan_vector_full: %v, scan_vector_sparse: %v",
		this.scan_level, this.scan_wait, this.scan_vector_full, this.scan_vector_sparse)
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
