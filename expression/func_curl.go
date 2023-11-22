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
	rv := &Curl{
		*NewFunctionBase("curl", operands...),
	}

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

	// Get allowlist from UI
	var allowlist map[string]interface{}

	_curlContext := context.(CurlContext)
	if _curlContext != nil {
		allowlist = _curlContext.GetAllowlist()
	}

	// Now you have the URL and the options with which to call curl.
	result, err := handleCurl(curl_url, options, allowlist, context)

	if err != nil {
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

func handleCurl(urlS string, options map[string]interface{}, allowlist map[string]interface{}, context Context) (
	interface{}, error) {

	// Convert URL string to net/url object that is valid to be used in CURL()
	urlObj, err := CurlURLStringToObject(urlS)
	if err != nil || urlObj == nil {
		return nil, err
	}

	// Check if the input URL contains elements that are restricted by Couchbase
	_, err = cbRestrictedURLCheck(urlObj)
	if err != nil {
		return nil, err
	}

	url := urlObj.String()

	// Handle different cases

	// initial check for curl_allowlist.json has been completed. The file exists.
	// Now we need to access the contents of the file and check for validity.
	err = allowlistCheck(allowlist, urlObj)
	if err != nil {
		return nil, err
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

	// Process the "header" option
	header, err := headerOptionProcessing(options)
	if err != nil {
		if show_error {
			return nil, err
		}
		return nil, nil
	}

	dialer := &net.Dialer{
		// connect-timeout
		Timeout: _DEF_CONNECT_TIMEOUT,
	}

	client := &http.Client{

		// Override the default CheckRedirect method
		// Now, if url redirection is attempted - call finishes after the first request
		// no error is returned ( since CheckRedirect returns the ErrUseLastResponse error )
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},

		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				// Default value of SSL Verification in libcurl is true
				InsecureSkipVerify: false,
			},
		},

		// Default value of max-time be the request timeout
		Timeout: context.GetTimeout(),
	}

	var body io.Reader
	body = nil
	username := ""
	password := ""
	authOp := false
	transport, _ := client.Transport.(*http.Transport)

	// By default, unless options processing decide otherwise )
	// the default request method is GET
	method := http.MethodGet

	// whether the request was explicitly set to "GET" via the "get" or "request" option
	getOptSet := false

	for k, val := range options {
		// Only support valid options.
		inputVal := value.NewValue(val)
		switch k {
		case "show_error", "show-error", "S":
			break
		case "get", "G":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for get option in CURL ")
				} else {
					return nil, nil
				}
			}
			if inputVal.Truth() {
				getOptSet = true
				method = http.MethodGet
			} else {
				getOptSet = false
				method = http.MethodPost
			}
		case "request", "X":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for request option in CURL. It should be a string. ")
			}
			requestVal := inputVal.ToString()
			if requestVal != "GET" && requestVal != "POST" {
				if show_error == true {
					return nil, fmt.Errorf("CURL only supports GET and POST requests. ")
				} else {
					return nil, nil
				}
			}
			if requestVal == "GET" {
				getOptSet = true
				method = http.MethodGet
			} else {
				getOptSet = false
				method = http.MethodPost
			}
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

			// The value of the "Authorization" specified in the "headers" option
			// takes precedence over "user" option
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
			// N1QL CURL() supports only Basic Authorization
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for basic option in CURL ")
				} else {
					return nil, nil
				}
			}
		case "anyauth":
			// N1QL CURL() supports only Basic Authorization
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

			if inputVal.Truth() {
				transport.TLSClientConfig.InsecureSkipVerify = true

			} else {
				transport.TLSClientConfig.InsecureSkipVerify = false

			}
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
				return nil, fmt.Errorf(subdir + " does not exist on node " + hostname)
			}

			// nsserver uses the inbox folder within var/lib/couchbase to read certificates from.
			certDir = certDir + subdir
			// dir. Paths are not allowed.
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for cacert option in CURL. It should be a string. ")
			}
			certName := inputVal.ToString()

			// Make sure this file is not a path.
			dir, file := path.Split(certName)
			if dir != "" || file == "" {
				return nil, fmt.Errorf("Cacert should only contain the certificate name. Paths are invalid. ")
			}

			// Also make sure the extension is .pem
			if path.Ext(file) != ".pem" {
				return nil, fmt.Errorf("Cacert should only contain the certificate name that refers to a valid pem file. ")
			}

			caCert, err := ioutil.ReadFile(certDir + file)

			if err != nil {
				return nil, fmt.Errorf("Error in reading cacert file - %v", err)
			}

			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			transport.TLSClientConfig.RootCAs = caCertPool

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

	// Akin to curl
	// If the data or data-urlencode options are set
	// The default method is POST
	// To make it a GET request, the method is to be explicitly set to GET via the "get" or "request" option
	if dataOp {
		finalStrData := stringData
		if stringDataUrlEnc != "" {
			if len(finalStrData) != 0 {
				finalStrData += "&"
			}
			finalStrData += stringDataUrlEnc
		}
		if getOptSet {
			url = url + "?" + finalStrData
		} else {
			method = http.MethodPost
			body = strings.NewReader(finalStrData)
		}
	}

	// Create the http request
	req, rerr := http.NewRequest(method, url, body)
	req.Header = header

	if rerr != nil {
		return nil, fmt.Errorf("Request could not be initialized.")
	}

	req.Header.Set("X-N1QL-User-Agent", _N1QL_USER_AGENT)

	// For POST requests, if "Content-Type" is not set in the "headers" option - set it to the default content type
	if method == http.MethodPost && req.Header.Get(_DEF_HEADER_CONTENT_TYPE) == "" {
		req.Header.Set(_DEF_HEADER_CONTENT_TYPE, _DEF_POST_CONTENT_TYPE)
	}

	if authOp {
		req.SetBasicAuth(username, password)
	}

	// if "ciphers" option is not set - use the default list of ciphers from cbauth
	if _, ok = options["ciphers"]; !ok {
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
				return nil, fmt.Errorf("Invalid JSON endpoint %v", url)
			} else {
				return nil, nil
			}
		}

		return dat, nil
	}
	return nil, nil
}

// Convert URL string to net/url object that is in a format supported by CURL()
func CurlURLStringToObject(urlS string) (*url.URL, error) {

	urlParsed, err := url.Parse(urlS)
	if err != nil {
		return nil, err
	}

	// Make preliminary checks of the parsed URL
	// since the path can be appropriately parsed only if the url is in a particular format
	// CURL() requires the protocol, host to be specified
	// CURL() only supports http and https protocols
	protocol := urlParsed.Scheme
	if protocol != "http" && protocol != "https" {
		return nil, fmt.Errorf("Unspecified or unsupported protocol scheme in request URL.")
	}

	if urlParsed.Host == "" {
		return nil, fmt.Errorf("No host in request URL.")
	}

	// Perform relative resolution on the input URL
	// Create a base URL object from an empty path
	baseURL, err := url.Parse("")
	if err != nil {
		return nil, err
	}

	// Resolve the input url object against the base URL
	// This will resolve references in the input URL
	resolvedURL := baseURL.ResolveReference(urlParsed)

	// Replace multiple adjacent forward slashes with a single forward slash
	urlObj := resolvedURL.JoinPath()

	return urlObj, nil

}

// Checks if the URL matches any URLs restricted by Couchbase
func cbRestrictedURLCheck(url *url.URL) (bool, error) {

	path := url.EscapedPath()

	// Restrict access to the cluster's /diag/eval endpoint
	// Restrict any URL where the path begins with /diag/eval
	matched := hasPathPrefix(path, "/diag/eval")

	if matched {
		return true, fmt.Errorf("Access restricted - %v.", url.String())
	}

	return false, nil
}

// Check if URLs in allowlist and disallowedList contain the input URL
func allowlistCheck(list map[string]interface{}, urlObj *url.URL) error {
	// Structure is as follows
	// {
	//  "all_access":true/false,
	//  "allowed_urls":[ list of allowed URL strings ]
	//  "allowed_transformed_urls":[ list of allowed net/url objects that are valid to be processed in CURL() ],
	//  "disallowed_urls":[ list of disallowed URL strings ]
	//  "disallowed_transformed_urls":[ list of disallowed net/url objects that are valid to be processed in CURL() ],
	// }

	// allowlist passed through ns server is empty then no access
	if len(list) == 0 {
		return fmt.Errorf("Allowed list for cluster is empty.")
	}

	// allowlist passed through ns server doesnt contain all access field then no access
	allaccess, ok := list["all_access"]
	if !ok {
		return fmt.Errorf("all_access does not exist in allowedlist.")
	}

	_, isOk := allaccess.(bool)

	if !isOk {
		// Type check error
		return fmt.Errorf("all_access should be boolean value in the CURL allowedlist.")
	}

	if allaccess.(bool) {
		return nil
	}

	// ALLOWED AND DISALLOWED URLS

	// If all_access false - Use only those entries that are valid.
	// Restricted access based on fields allowed_transformed_urls and disallowed_transformed_urls

	if disallowedUrls, ok_dall := list["disallowed_transformed_urls"]; ok_dall {
		dURL, ok := disallowedUrls.([]*url.URL)
		if !ok {
			return fmt.Errorf("Restrict access with disallowed urls")
		}
		if len(dURL) > 0 {
			disallow := matchUrl(urlObj, dURL)
			if disallow {
				return fmt.Errorf("The endpoint %s is not permitted", urlObj.String())
			}
		}
	}

	if allowedUrls, ok_all := list["allowed_transformed_urls"]; ok_all {
		alURL, ok := allowedUrls.([]*url.URL)
		if !ok {
			return fmt.Errorf("allowed_urls should be list of urls present in the allowedlists.")
		}
		if len(alURL) > 0 {
			allow := matchUrl(urlObj, alURL)
			if allow {
				return nil
			}
		}
	}

	// URL is not present in disallowed url and is not in allowed_urls.
	// If it reaches here, then the url isnt in the allowed_urls or the prefix_urls, and is also
	// not in the disallowed urls.
	return fmt.Errorf("The end point %s is not permitted.  List allowed end points in the configuration.", urlObj.String())

}

// Check if URL is allowed/disallowed as per the allowlist/ disallow list
// Checks each element of the input URL with the URLs in the list
func matchUrl(u *url.URL, list []*url.URL) bool {

	inputUserInfo := u.User.String()

	for _, l := range list {
		if u.Scheme != l.Scheme {
			continue
		}

		if u.Host != l.Host {
			continue
		}

		// Only in when the URL in the list has User Information - compare it with Input URL's user information
		lUserInfo := l.User.String()
		if lUserInfo != "" {
			if inputUserInfo != lUserInfo {
				continue
			}
		}

		matched := hasPathPrefix(u.EscapedPath(), l.EscapedPath())

		if matched {
			return true
		}
	}

	return false
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
			return nil, fmt.Errorf("Incorrect type for header option " + value.NewValue(val).String() +
				" in CURL. Header option should be a string value or an array of strings.  ")
		}

		// We have an array of interfaces that represent different fields in the Header.
		// Add all the headers to a []string to pass to OPT_HTTPHEADER
		for _, hval := range headerVal.([]interface{}) {
			newHval := value.NewValue(hval)
			if newHval.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for header option " + newHval.String() +
					" in CURL. Header option should be a string value or an array of strings.  ")
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

// Helper function to determine if 'path' starts with a particular path 'prefix'
func hasPathPrefix(path string, prefix string) bool {

	// Initial prefix matching
	if !strings.HasPrefix(path, prefix) {
		return false
	}

	// If the 'prefix' path does not end with "/" then the initial prefix matching might not be sufficient
	// The following check is to prevent the mistaken matching of cases like:
	// path = "/testt" and prefix = "/test"
	if !strings.HasSuffix(prefix, "/") {
		n := len(prefix)

		// It is an exact match
		if len(path) == n {
			return true
		}

		// if the next character after the matched prefix is a path separator / - it is a match
		if path[n] == '/' {
			return true
		}

		return false
	}

	return true

}
