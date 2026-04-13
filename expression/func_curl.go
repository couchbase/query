//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/accounting"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// /////////////////////////////////////////////////
//
// # Curl
//
// /////////////////////////////////////////////////

// Mapping of Cipher String to Cipher Id
var CipherMap map[string]uint16

func init() {
	// Map only the secure ciphers supported by crypto/tls
	list := tls.CipherSuites()
	CipherMap = make(map[string]uint16, len(list))

	for _, cipher := range list {
		CipherMap[strings.ToUpper(cipher.Name)] = cipher.ID
	}
}

// N1QL User-Agent value
var (
	_N1QL_USER_AGENT = "couchbase/n1ql/" + util.VERSION
)

// Max request size from server (cant import because of cyclic dependency)
const (
	_MIN_RESPONSE_SIZE          = 512 * util.KiB
	_DEFAULT_RESPONSE_SIZE      = 20 * util.MiB
	_MAX_NO_QUOTA_RESPONSE_SIZE = 128 * util.MiB
)

// Path to certs
const (
	_PATH = "/../var/lib/couchbase/n1qlcerts/"
)

const (
	// Unless changed in header - default Content-Type of POST requests
	_DEF_POST_CONTENT_TYPE = "application/x-www-form-urlencoded"
	// default libcurl connect-timeout is 300 seconds
	_DEF_CONNECT_TIMEOUT = 300 * time.Second
)

// Header constants
const (
	_DEF_HEADER_AUTHORIZATION = "Authorization"
	_DEF_HEADER_USER_AGENT    = "User-Agent"
	_DEF_HEADER_CONTENT_TYPE  = "Content-Type"
)

var hostname string

/*
This represents the curl function CURL(method, url, options).
It returns result of the curl operation on the url based on
the method and options.
*/
type Curl struct {
	FunctionBase
}

func NewCurl(operands ...Expression) Function {
	rv := &Curl{}
	rv.Init("curl", operands...)

	rv.setVolatile()
	rv.expr = rv
	return rv
}

/*
Visitor pattern.
*/
func (this *Curl) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitFunction(this)
}

func (this *Curl) Type() value.Type { return value.OBJECT }

/*
Uses a separate function for the body as external users make use of the computation portion.
*/
func (this *Curl) Evaluate(item value.Value, context Context) (value.Value, error) {
	var arg2 value.Value = nil
	arg1, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	if len(this.operands) > 1 {
		arg2, err = this.operands[1].Evaluate(item, context)
		if err != nil {
			return nil, err
		}
	}
	return this.DoEvaluate(context, arg1, arg2)
}

func (this *Curl) DoEvaluate(context Context, arg1, arg2 value.Value) (value.Value, error) {

	// In order to have restricted access, the administrator will have to create
	// curl_allowlist on the UI with the all_access field set to false.
	// In order to access all endpoints, the administrator will have to create
	// curl_allowlist.json with the all_access field set to true.

	// Before performing any checks curl_allowlist needs to be set on the UI.
	// 1. If it is empty, then return with error. (Disable access to the CURL function)
	// 2. For all other cases, CURL can execute depending on contents of the file, but we defer
	//    allowlist check to handle_curl()

	// URL
	first := arg1
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	// CURL URL
	curl_url := first.ToString()

	// Empty options to pass into curl.
	options := make(map[string]interface{})

	// If we have options then process them.
	if arg2 != nil {
		second := arg2

		if second.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if second.Type() == value.OBJECT {
			//Process the options
			for k, v := range second.Actual().(map[string]interface{}) {
				k = strings.TrimPrefix(strings.TrimPrefix(k, "--"), "-")
				options[k] = v
			}
		} else {
			return value.NULL_VALUE, nil
		}
	}

	accounting.UpdateCounter(accounting.CURL_CALLS)

	// Now you have the URL and the options with which to call curl.
	result, err := handleCurl(curl_url, options, context)

	if err != nil {
		accounting.UpdateCounter(accounting.CURL_CALL_ERRORS)
		return value.NULL_VALUE, errors.NewCurlExecutionError(err)
	}

	// For Silent mode where we dont want any output.
	switch results := result.(type) {
	case map[string]interface{}:
		if len(results) == 0 {
			return value.MISSING_VALUE, nil
		}
	case []interface{}:
		if len(results) == 0 {
			return value.MISSING_VALUE, nil
		}

	default:
		return value.NULL_VALUE, nil
	}

	return value.NewValue(result), nil
}

func (this *Curl) Privileges() *auth.Privileges {
	unionPrivileges := auth.NewPrivileges()
	unionPrivileges.Add("", auth.PRIV_QUERY_EXTERNAL_ACCESS, auth.PRIV_PROPS_NONE)

	children := this.Children()
	for _, child := range children {
		unionPrivileges.AddAll(child.Privileges())
	}

	return unionPrivileges
}

func (this *Curl) Indexable() bool {
	return false
}

func (this *Curl) MinArgs() int { return 1 }

func (this *Curl) MaxArgs() int { return 2 }

/*
Factory method pattern.
*/
func (this *Curl) Constructor() FunctionConstructor {
	return NewCurl
}

func handleCurl(urlS string, options map[string]any, context Context) (
	any, error) {

	// Parse and validate the request URL: requires http/https scheme, non-empty host,
	// and normalises the path. Errors here identify the CURL() target URL as malformed.
	urlObj, err := util.ParseAndValidateURL(urlS)
	if err != nil || urlObj == nil {
		return nil, fmt.Errorf("CURL(): invalid request URL %q: %v", urlS, err)
	}

	availableQuota := uint64(_MAX_NO_QUOTA_RESPONSE_SIZE)
	responseSize := uint64(_DEFAULT_RESPONSE_SIZE)

	ctx, ok := context.(QuotaContext)
	if !ok || !ctx.UseRequestQuota() {
		ctx = nil
	} else if ctx.MemoryQuota() != 0 {
		availableQuota = uint64(float64(ctx.MemoryQuota()) * (1.0 - ctx.CurrentQuotaUsage()))
		if availableQuota < _MIN_RESPONSE_SIZE {
			availableQuota = _MIN_RESPONSE_SIZE
		}
		if responseSize > availableQuota {
			responseSize = availableQuota
		}
	}

	dataOp := false
	stringData := ""
	stringDataUrlEnc := ""
	silent := false
	url := urlObj.String()

	// To show errors encountered when executing the CURL function.
	show_error := true
	showErrVal, ok := options["show_error"]
	if !ok {
		showErrVal, ok = options["show-error"]
	}
	if !ok {
		showErrVal, ok = options["S"]
	}
	if ok {
		if value.NewValue(showErrVal).Type() != value.BOOLEAN {
			return nil, fmt.Errorf("Incorrect type for show_error option in CURL ")
		}
		show_error = value.NewValue(showErrVal).Actual().(bool)
	}

	err = IsUrlAllowedInCluster(urlObj, context)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	header := http.Header{}

	cred_id, ok := options["cred_id"]
	if ok {
		if value.NewValue(cred_id).Type() != value.STRING {
			return nil, fmt.Errorf("Incorrect type for cred_id option in CURL ")
		}
		credId := value.NewValue(cred_id).ToString()

		// cred_id centralizes auth and TLS material in credstore; explicit options
		// are still parsed later, but this branch seeds the client/header defaults
		// based on the credential type stored in the credstore for the given cred_id.
		client, header, err = HandleCred(urlObj, credId, context)
		if err != nil {
			return nil, err
		}
	} else {
		client, err = GetDefaultHttpClient(context)
		if err != nil {
			return nil, err
		}
	}

	transport := UnwrapTransport(client.Transport)
	if transport == nil {
		return nil, fmt.Errorf("Unable to find *http.Transport in client transport chain")
	}

	// Process the "header" option
	explicitHeader, err := headerOptionProcessing(options)
	if err != nil {
		if show_error {
			return nil, err
		}
		return nil, nil
	}

	// User-supplied headers are applied after the credential-derived headers established by HandleCred().
	// New header keys are appended to the existing set, while keys that conflict with credential-derived headers are intentionally overridden. This allows callers to reuse transport-level credential settings (e.g. certificates, private keys) from a stored credential while supplying a different authentication
	// token or other request-specific headers at call time.
	for k, v := range explicitHeader {
		header[http.CanonicalHeaderKey(k)] = v
	}

	// Process the "get" and "request" options
	method, err := getAndRequestOptionProcessing(options)
	if err != nil {
		if show_error == true {
			return nil, err
		}
		return nil, nil
	}

	// whether the request was explicitly set to "GET" via the "get" or "request" option
	getOptSet := false

	if method == "" {
		// request method not explicitly set
		// Akin to curl - by default the request method is GET
		// unless processing of data options decide otherwise
		method = http.MethodGet
	} else if method == http.MethodGet {
		getOptSet = true
	}

	connectTimeout := context.GetTimeout()
	if connectTimeout <= 0 {
		connectTimeout = _DEF_CONNECT_TIMEOUT
	}

	// Create the net/http client
	dialer := &net.Dialer{
		// connect-timeout
		Timeout: connectTimeout,
	}

	var body io.Reader
	var certFile, keyFile string
	var passPhrase []byte
	body = nil
	username := ""
	password := ""
	authOp := false

	for k, val := range options {
		// Only support valid options.
		inputVal := value.NewValue(val)
		switch k {
		case "show_error", "show-error", "S":
			break
		case "get", "G":
			break
		case "request", "X":
			break
		case "cred_id":
			break
		case "data-urlencode":
			stringDataUrlEnc, err = handleData(true, val, show_error)
			if stringDataUrlEnc == "" {
				return nil, err
			}
			dataOp = true
		case "data", "d":
			stringData, err = handleData(false, val, show_error)
			if stringData == "" {
				return nil, err
			}
			dataOp = true
		case "headers", "header", "H":
			// CURL() also supports JWT authentication. User need to provide the JWT access token in this format:
			// Authorization: Bearer <JWT_ACCESS_TOKEN>
			break
		case "silent", "s":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for silent option in CURL ")
				} else {
					return nil, nil
				}
			}
			silent = inputVal.Truth()
		case "connect-timeout":
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for connect-timeout option in CURL ")
			}

			// convert input in seconds to nanoseconds
			dialer.Timeout = time.Duration(value.AsNumberValue(inputVal).Int64() * int64(time.Second))
		case "max-time", "m":
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for max-time option in CURL ")
			}

			// convert input in seconds to nanoseconds
			client.Timeout = time.Duration(value.AsNumberValue(inputVal).Int64() * int64(time.Second))
		case "user", "u":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for user option in CURL. It should be a string. ")
			}

			// The Authorization header set by cred_id or the "headers" option takes
			// precedence over the "user" option.
			if header.Get(_DEF_HEADER_AUTHORIZATION) == "" {
				username, password = splitUser(inputVal.ToString())
				authOp = true
			}

		case "ciphers":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for ciphers option in CURL. It should be a string. ")
			}
			cipherIds, err := cipherStringToIds(inputVal.ToString(), false)
			if err != nil {
				return nil, err
			} else {
				transport.TLSClientConfig.CipherSuites = cipherIds
			}

		case "basic":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for basic option in CURL ")
				} else {
					return nil, nil
				}
			}
		case "anyauth":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for anyauth option in CURL ")
				} else {
					return nil, nil
				}
			}
		case "insecure", "k":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for insecure option in CURL ")
				} else {
					return nil, nil
				}
			}

			transport.TLSClientConfig.InsecureSkipVerify = inputVal.Truth()
		case "keepalive-time":
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for keepalive-time option in CURL ")
			}

			// dialer.KeepAlive sets both the TCP_KEEPIDLE and TCP_KEEPINTVL value
			dialer.KeepAlive = time.Duration(value.AsNumberValue(inputVal).Int64() * int64(time.Second))
		case "user-agent", "A":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for user-agent option in CURL. user-agent should be a string. ")
			}

			// The value of the "User-Agent" specified in the "headers" option
			// takes precedence over "user-agent" option
			if header.Get(_DEF_HEADER_USER_AGENT) == "" {
				header.Set(_DEF_HEADER_USER_AGENT, inputVal.ToString())
			}

		case "cacert":
			caFile, err := getFileName(inputVal)
			if err != nil {
				return nil, err
			}
			caCert, err := ioutil.ReadFile(caFile)
			if err != nil {
				return nil, fmt.Errorf("Error in reading %s file - %v", caFile, err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			transport.TLSClientConfig.RootCAs = caCertPool

		case "cert":
			var err1 error
			certFile, err1 = getFileName(inputVal)
			if err1 != nil {
				return nil, err1
			}

		case "key":
			var err1 error
			keyFile, err1 = getFileName(inputVal)
			if err1 != nil {
				return nil, err1
			}

		case "passphrase":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Passphrase must be a string.")
			}
			passPhrase = []byte(inputVal.ToString())

		case "result-cap":
			// Restricted to 0.5 - 256 MiB
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for result-cap option in CURL ")
			}
			// negatives set to minimum
			rs := value.AsNumberValue(inputVal).Int64()
			if rs < _MIN_RESPONSE_SIZE {
				logging.Debugf("CURL (%v) result-cap %v set to %v", url, rs, _MIN_RESPONSE_SIZE)
				rs = _MIN_RESPONSE_SIZE
			}
			responseSize = uint64(rs)
			// if there is a quota the remaining available memory enforces the upper limit
			if responseSize > availableQuota {
				logging.Debugf("CURL (%v) result-cap %v limited to %v", url, responseSize, availableQuota)
				responseSize = availableQuota
			}
		default:
			return nil, fmt.Errorf("CURL option %v is not supported.", k)

		}

	}

	if certFile != "" && keyFile != "" {
		if _curlContext, ok := context.(CurlContext); ok {
			tlsCert1, err := _curlContext.LoadX509KeyPair(certFile, keyFile, passPhrase)
			if err != nil {
				return nil, err
			}
			tlsCert, _ := tlsCert1.(tls.Certificate)
			transport.TLSClientConfig.Certificates = []tls.Certificate{tlsCert}
		} else {
			return nil, fmt.Errorf("Curl context is nil")
		}
	} else if certFile != "" || keyFile != "" {
		//error need to pass both certfile and keyfile
		return nil, fmt.Errorf("Requires both cert and key options")
	}

	// Akin to curl
	// If the data or data-urlencode options are set
	// The default method is POST
	// Unless the request is explicitly made a GET request via the "get" or "request" options
	if dataOp {
		finalStrData := stringData
		if stringDataUrlEnc != "" {
			if len(finalStrData) != 0 {
				finalStrData += "&"
			}
			finalStrData += stringDataUrlEnc
		}
		if getOptSet {
			// GET requests have the data appended to the URL
			url = url + "?" + finalStrData
		} else {
			// POST requests have the data sent in the request body
			method = http.MethodPost
			body = strings.NewReader(finalStrData)
		}
	}

	// Create the http request
	req, rerr := http.NewRequest(method, url, body)
	if rerr != nil {
		return nil, fmt.Errorf("Request could not be initialized.")
	}
	req.Header = header

	req.Header.Set("X-N1QL-User-Agent", _N1QL_USER_AGENT)

	// For POST requests, if "Content-Type" is not set in the "headers" option - set it to the default content type
	if method == http.MethodPost && req.Header.Get(_DEF_HEADER_CONTENT_TYPE) == "" {
		req.Header.Set(_DEF_HEADER_CONTENT_TYPE, _DEF_POST_CONTENT_TYPE)
	}

	if authOp {
		req.SetBasicAuth(username, password)
	}

	// if "ciphers" option is not set - use the default list of ciphers from cbauth
	_, credSet := options["cred_id"]
	if _, ok = options["ciphers"]; !ok && !credSet {
		cipherIds, err := cipherStringToIds("", true)
		if err != nil {
			return nil, err
		} else {
			transport.TLSClientConfig.CipherSuites = cipherIds
		}

	}

	if ctx != nil {
		err := ctx.TrackValueSize(responseSize)
		if err != nil {
			return nil, err
		}
		defer ctx.ReleaseValueSize(responseSize)
	}

	// Send request
	transport.DialContext = dialer.DialContext
	resp, rerr := client.Do(req)

	if rerr != nil {
		if show_error {
			// Detect if it was a timeout error
			if isTimeoutError(rerr) {
				return nil, fmt.Errorf("curl: Timeout was reached")
			}

			return nil, fmt.Errorf("Error during request - %v", rerr)
		} else {
			return nil, nil
		}
	}

	// Read the response body
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	var b bytes.Buffer
	var buf [4096]byte

	for {
		n, err := reader.Read(buf[0:])

		if err != nil && n <= 0 {
			if err == io.EOF {
				err = nil
			} else {
				if show_error {
					// Detect if it was a timeout error
					if isTimeoutError(err) {
						return nil, fmt.Errorf("curl: Timeout was reached")
					}

					return nil, fmt.Errorf("Error while reading response body - %v", err)
				} else {
					return nil, nil
				}
			}
			break
		}

		// if maximum allowed response size has been reached - throw an error
		if uint64(b.Len()) > responseSize {
			if show_error {
				return nil, fmt.Errorf("Response size limit of %v has been reached.", responseSize)
			} else {
				return nil, nil
			}
		}

		// only write if "silent" is false
		if silent == false {
			b.Write([]byte(buf[0:n]))
		}
	}

	// The return type can either be and ARRAY or an OBJECT
	if b.Len() != 0 {
		var dat interface{}

		if err := json.Unmarshal(b.Bytes(), &dat); err != nil {
			if show_error == true {
				// Include the HTTP status and raw body so the caller can see what
				// the server actually returned (e.g. "415 Unsupported Media Type").
				return nil, fmt.Errorf("HTTP %d: response from %v is not valid JSON: %s",
					resp.StatusCode, url, strings.TrimSpace(b.String()))
			} else {
				return nil, nil
			}
		}

		return dat, nil
	}

	// Empty response body: surface non-2xx status codes so callers are not left
	// with a silent NULL (e.g. 415 Unsupported Media Type with no body).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if show_error {
			return nil, fmt.Errorf("HTTP %d from %v", resp.StatusCode, url)
		}
		return nil, nil
	}
	return nil, nil
}

// encodeData: if val is to be url encoded
func handleData(encodedData bool, val interface{}, show_error bool) (string, error) {

	// used to create the data string when "data" option is set
	stringData := ""

	dataVal := value.NewValue(val).Actual()

	switch dataVal.(type) {
	case []interface{}:
	case string:
		dataVal = []interface{}{dataVal}
	default:
		if show_error == true {
			return "", fmt.Errorf("Incorrect type for data option in CURL.It needs to be a string. ")
		} else {
			return "", nil
		}
	}

	dv := dataVal.([]interface{})

	var params url.Values
	if encodedData {
		params = make(url.Values, len(dv))
	}

	for _, data := range dv {
		newDval := value.NewValue(data)
		if newDval.Type() != value.STRING {
			if show_error == true {
				return "", fmt.Errorf("Incorrect type for data option. ")
			} else {
				return "", nil
			}
		}

		dataT := newDval.ToString()

		// If data is to be url-encoded
		// URL parameters are of the form: key=value
		if encodedData {
			if idx := strings.IndexByte(dataT, '='); idx >= 0 {
				// Split string into key and value
				key := dataT[:idx]
				value := dataT[idx+1:]
				params.Add(key, value)
			} else {
				params.Add(dataT, "")
			}

			continue
		} else {
			if stringData == "" {
				stringData = dataT
			} else {
				stringData = stringData + "&" + dataT
			}
		}

	}

	if encodedData {
		// Perform url encoding
		return params.Encode(), nil
	}

	return stringData, nil
}

// Given a string of the format: [username:password]
// Returns username, password
func splitUser(val string) (username string, password string) {
	if val != "" {
		if idx := strings.IndexByte(val, ':'); idx >= 0 {
			username = val[:idx]
			password = val[idx+1:]
		} else {
			username = val
			// no specified password
			// Password is an empty password if there isnt one
			password = ""
		}
	}

	return username, password

}

// Converts string of comma separated cipher names
// to a list of their corresponding cipher IDs
// if cbDefault = true - return default cipherId list from cbauth
func cipherStringToIds(val string, cbDefault bool) ([]uint16, error) {
	if !cbDefault {
		cipherStrings := strings.Split(val, ",")
		cipherIds := make([]uint16, len(cipherStrings))

		for _, c := range cipherStrings {
			cipherName := strings.TrimSpace(c)

			// find the id of the cipher given the name

			if cipherId, ok := CipherMap[strings.ToUpper(cipherName)]; ok {
				cipherIds = append(cipherIds, cipherId)
			} else {
				return nil, fmt.Errorf("The specified cipher is not supported")
			}
		}

		return cipherIds, nil

	} else {
		// Get the Ciphers supported by couchbase based on the level set
		tlsCfg, err := cbauth.GetTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("Failed to get cbauth tls config: %v", err.Error())
		}

		return tlsCfg.CipherSuites, nil
	}
}

// Process the "get" and "request" options
// returns "GET" if explicitly set to a GET request
// returns "POST" if explicitly set to a POST request
// returns an empty string "" if a request type is not explicitly set
func getAndRequestOptionProcessing(options map[string]interface{}) (string, error) {

	// Process 'get' option
	getTrue := false

	gVal, okG := options["get"]
	if !okG {
		gVal, okG = options["G"]
	}

	if okG {
		val := value.NewValue(gVal)
		if val.Type() != value.BOOLEAN {
			return "", fmt.Errorf("Incorrect type for get option in CURL ")
		}
		if val.Truth() {
			getTrue = true
		}
	}

	if getTrue {
		return http.MethodGet, nil
	}

	// Only if the 'get' option was not set or was set to False evaluate the 'request' option
	rVal, okR := options["request"]
	if !okR {
		rVal, okR = options["X"]
	}

	method := ""

	if okR {
		val := value.NewValue(rVal)
		if val.Type() != value.STRING {
			return "", fmt.Errorf("Incorrect type for request option in CURL. It should be a string. ")
		}

		requestVal := val.ToString()
		if requestVal == "GET" {
			method = http.MethodGet
		} else if requestVal == "POST" {
			method = http.MethodPost
		} else {
			return "", fmt.Errorf("CURL only supports GET and POST requests. ")
		}
	}

	return method, nil

}

// Process the "header" option
func headerOptionProcessing(options map[string]interface{}) (http.Header, error) {
	header := http.Header{}
	hVal, ok := options["headers"]

	if !ok {
		hVal, ok = options["header"]
	}

	if !ok {
		hVal, ok = options["H"]
	}

	if ok {
		val := value.NewValue(hVal)

		// Get the value
		headerVal := val.Actual()
		switch headerVal.(type) {
		case []interface{}:
			//Do nothing
		case string:
			headerVal = []interface{}{headerVal}
		default:
			return nil, fmt.Errorf("Incorrect type for header option %s"+
				" in CURL. Header option should be a string value or an array of strings.  ",
				value.NewValue(val).String())
		}

		// We have an array of interfaces that represent different fields in the Header.
		// Add all the headers to a []string to pass to OPT_HTTPHEADER
		for _, hval := range headerVal.([]interface{}) {
			newHval := value.NewValue(hval)
			if newHval.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for header option %s"+
					" in CURL. Header option should be a string value or an array of strings.  ",
					newHval.String())
			}

			h := newHval.ToString()

			// Individual header string processing
			// A header is of the form:-  Key: Value
			if idx := strings.IndexByte(h, ':'); idx >= 0 {
				key := strings.TrimSpace(h[:idx])
				val := h[idx+1:]

				// Multiple values associated with the header key
				// will be separated by commas
				split := strings.Split(val, ",")
				for _, v := range split {
					header.Add(key, strings.TrimSpace(v))
				}
			} else {
				header.Add(h, "")
			}
		}
	}

	return header, nil
}

// if the error is a timeout error returned by net/http
func isTimeoutError(err error) bool {
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return true
	}

	return false
}

func getFileName(val value.Value) (string, error) {
	// All the certificates are stored withing the ..var/lib/couchbase/n1qlcerts
	// Find the os
	subdir := filepath.FromSlash(_PATH)

	// Get directory of currently running file.
	certDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		// Get ip addresses to display in error
		name, _ := os.Hostname()
		addrs, err := net.LookupHost(name)
		if err != nil {
			logging.Infof("Error looking up hostname: %v", err)
		}

		hostname = strings.Join(addrs, ",")
		return "", fmt.Errorf("%s does not exist on node %s", subdir, hostname)
	}

	// nsserver uses the inbox folder within var/lib/couchbase to read certificates from.
	certDir = certDir + subdir
	// dir. Paths are not allowed.
	if val.Type() != value.STRING {
		return "", fmt.Errorf("cacert/cert/key must be a string.")
	}
	name := val.ToString()
	// Make sure this file is not a path.
	dir, file := path.Split(name)
	if dir != "" || file == "" {
		return "", fmt.Errorf("cacert/cert/key should only contain the name. Paths are invalid.")
	}

	return certDir + file, nil
}
