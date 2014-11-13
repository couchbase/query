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
	"strconv"
	"strings"
	"time"

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
	httpRespCode int
	resultCount  int
	resultSize   int
	errorCount   int
	warningCount int
}

func newHttpRequest(resp http.ResponseWriter, req *http.Request) *httpRequest {
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

	base := server.NewBaseRequest(statement, prepared, namedArgs, positionalArgs,
		namespace, readonly, metrics, signature, consistency)

	rv := &httpRequest{
		BaseRequest: *base,
		resp:        resp,
		req:         req,
	}

	rv.SetTimeout(rv, timeout)

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

const (
	URL_CONTENT  = "application/x-www-form-urlencoded"
	JSON_CONTENT = "application/json"
)

const ( // Request argument names
	READONLY         = "readonly"
	METRICS          = "metrics"
	NAMESPACE        = "namespace"
	TIMEOUT          = "timeout"
	ARGS             = "args"
	PREPARED         = "prepared"
	STATEMENT        = "statement"
	FORMAT           = "format"
	ENCODING         = "encoding"
	COMPRESSION      = "compression"
	SIGNATURE        = "signature"
	PRETTY           = "pretty"
	SCAN_CONSISTENCY = "scan_consistency"
	SCAN_WAIT        = "scan_wait"
)

// httpRequestArgs is an interface for getting the arguments in a http request
type httpRequestArgs interface {
	getString(string, string) (string, error)
	getBoolean(string, bool) (bool, error)
	getTimeDuration(string) (time.Duration, error)
	getNamedArgs() (map[string]value.Value, error)
	getPositionalArgs() (value.Values, error)
	getStatement() (string, error)
}

// getRequestParams creates a httpRequestArgs implementation,
// depending on the content type in the request
func getRequestParams(req *http.Request) (httpRequestArgs, error) {
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
		namedArgs[namedArg] = argValue
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

type scanConfigImpl struct {
	scan_level server.ScanConsistency
	scan_wait  time.Duration
}

func (this *scanConfigImpl) ScanConsistency() server.ScanConsistency {
	return this.scan_level
}

func (this *scanConfigImpl) ScanWait() time.Duration {
	return this.scan_wait
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
