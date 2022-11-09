//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
	"github.com/couchbasedeps/go-curl"
)

///////////////////////////////////////////////////
//
// Curl
//
///////////////////////////////////////////////////

// To look at values for headers see https://sourceforge.net/p/curl/bugs/385/
// For a full list see :
// https://github.com/curl/curl/blob/6b7616690e5370c21e3a760321af6bf4edbabfb6/include/curl/curl.h

// Protocol constants
const (
	_CURLPROTO_HTTP  = 1 << 0 /* HTTP Protocol */
	_CURLPROTO_HTTPS = 1 << 1 /* HTTPS Protocol */

)

// Authentication constants
const (
	_CURLAUTH_BASIC = 1 << 0 /* Basic (default)*/
	_CURLAUTH_ANY   = ^(0)   /* all types set */
)

// N1QL User-Agent value
var (
	_N1QL_USER_AGENT = "couchbase/n1ql/" + util.VERSION
)

// Max request size from server (cant import because of cyclic dependency)
const (
	MIN_RESPONSE_SIZE = 20 * (1 << 20)
	MAX_RESPONSE_SIZE = 64 * (1 << 20)
)

// Path to certs
const (
	_PATH = "/../var/lib/couchbase/n1qlcerts/"
)

const (
	_WINCIPHERS = "CALG_AES_128,CALG_AES_256,CALG_SHA_256,CALG_SHA_384,CALG_RSA_SIGN,CALG_RSA_KEYX,CALG_ECDSA"
)

var hostname string

/*
This represents the curl function CURL(method, url, options).
It returns result of the curl operation on the url based on
the method and options.
*/
type Curl struct {
	FunctionBase
	myCurl *curl.CURL
}

func NewCurl(operands ...Expression) Function {
	rv := &Curl{
		*NewFunctionBase("curl", operands...),
		nil,
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

	this.myCurl = curl.EasyInit()
	if this.myCurl == nil {
		return value.NULL_VALUE, fmt.Errorf("Error initializing libcurl")
	}
	// Now you have the URL and the options with which to call curl.
	result, err := this.handleCurl(curl_url, options, allowlist)

	this.myCurl.Cleanup()
	this.myCurl = nil

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

func (this *Curl) handleCurl(url string, options map[string]interface{}, allowlist map[string]interface{}) (interface{}, error) {
	// Handle different cases

	// initial check for curl_allowlist.json has been completed. The file exists.
	// Now we need to access the contents of the file and check for validity.
	err := allowlistCheck(allowlist, url)
	if err != nil {
		return nil, err
	}

	responseSize := setResponseSize(MIN_RESPONSE_SIZE)
	sizeError := false
	getMethod := false
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

	this.myCurl.Setopt(curl.OPT_MAXREDIRS, 0)
	this.myCurl.Setopt(curl.OPT_PROTOCOLS, _CURLPROTO_HTTP|_CURLPROTO_HTTPS)
	this.myCurl.Setopt(curl.OPT_USERAGENT, _N1QL_USER_AGENT)
	this.myCurl.Setopt(curl.OPT_URL, url)

	header := []string{}

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
				getMethod = true
				this.simpleGet(url)
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
				getMethod = true
			}
			this.myCurl.Setopt(curl.OPT_CUSTOMREQUEST, requestVal)
		case "data-urlencode":
			stringDataUrlEnc, err = this.handleData(true, val, show_error)
			if stringDataUrlEnc == "" {
				return nil, err
			}
			dataOp = true
		case "data", "d":
			stringData, err = this.handleData(false, val, show_error)
			if stringData == "" {
				return nil, err
			}
			dataOp = true
		case "headers", "header", "H":
			// Get the value
			headerVal := inputVal.Actual()
			switch headerVal.(type) {
			case []interface{}:
				//Do nothing
			case string:
				headerVal = []interface{}{headerVal}
			default:
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for header option " + value.NewValue(val).String() + " in CURL. Header option should be a string value or an array of strings.  ")
				} else {
					return nil, nil
				}
			}
			// We have an array of interfaces that represent different fields in the Header.
			// Add all the headers to a []string to pass to OPT_HTTPHEADER
			for _, hval := range headerVal.([]interface{}) {
				newHval := value.NewValue(hval)
				if newHval.Type() != value.STRING {
					if show_error == true {
						return nil, fmt.Errorf("Incorrect type for header option " + newHval.String() + " in CURL. Header option should be a string value or an array of strings.  ")
					} else {
						return nil, nil
					}

				}
				header = append(header, newHval.ToString())
			}
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
			this.myCurl.Setopt(curl.OPT_CONNECTTIMEOUT, value.AsNumberValue(inputVal).Int64())
		case "max-time", "m":
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for max-time option in CURL ")
			}
			this.myCurl.Setopt(curl.OPT_TIMEOUT, value.AsNumberValue(inputVal).Int64())
		case "user", "u":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for user option in CURL. It should be a string. ")
			}
			this.curlAuth(inputVal.ToString())
		case "ciphers":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for ciphers option in CURL. It should be a string. ")
			}
			this.curlCiphers(inputVal.ToString())
		case "basic":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for basic option in CURL ")
				} else {
					return nil, nil
				}
			}
			if inputVal.Truth() {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_BASIC)
			} else {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_ANY)
			}
		case "anyauth":
			if inputVal.Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf("Incorrect type for anyauth option in CURL ")
				} else {
					return nil, nil
				}
			}
			if inputVal.Truth() {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_ANY)
			} else {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_BASIC)
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
				this.myCurl.Setopt(curl.OPT_SSL_VERIFYPEER, 0)
			} else {
				this.myCurl.Setopt(curl.OPT_SSL_VERIFYPEER, 1)
			}
		case "keepalive-time":
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for keepalive-time option in CURL ")
			}
			val := value.AsNumberValue(inputVal).Int64()
			this.myCurl.Setopt(curl.OPT_TCP_KEEPALIVE, 1)
			this.myCurl.Setopt(curl.OPT_TCP_KEEPIDLE, val)
			this.myCurl.Setopt(curl.OPT_TCP_KEEPINTVL, val)
		case "user-agent", "A":
			if inputVal.Type() != value.STRING {
				return nil, fmt.Errorf("Incorrect type for user-agent option in CURL. user-agent should be a string. ")
			}
			this.myCurl.Setopt(curl.OPT_USERAGENT, inputVal.ToString())
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

			this.myCurl.Setopt(curl.OPT_SSLCERTTYPE, "PEM")
			this.myCurl.Setopt(curl.OPT_CAINFO, certDir+file)
		case "result-cap":
			// In order to restrict size of response use curlopt-range.
			// Min allowed = 20MB  20971520
			// Max allowed = request-size-cap default 67 108 864
			if inputVal.Type() != value.NUMBER {
				return nil, fmt.Errorf("Incorrect type for result-cap option in CURL ")
			}
			responseSize = setResponseSize(value.AsNumberValue(inputVal).Int64())
		default:
			return nil, fmt.Errorf("CURL option %v is not supported.", k)

		}

	}

	if dataOp {
		finalStrData := stringData
		if stringDataUrlEnc != "" {
			if len(finalStrData) != 0 {
				finalStrData += "&"
			}
			finalStrData += stringDataUrlEnc
		}
		if getMethod {
			this.simpleGet(url + "?" + finalStrData)
		} else {
			this.myCurl.Setopt(curl.OPT_POST, true)
			this.myCurl.Setopt(curl.OPT_POSTFIELDS, finalStrData)
		}
	}

	// Set the header, so that the entire []string are passed in.
	header = append(header, "X-N1QL-User-Agent: "+_N1QL_USER_AGENT)
	this.myCurl.Setopt(curl.OPT_HTTPHEADER, header)

	if _, ok = options["ciphers"]; !ok {
		if err := this.curlCiphers(""); err != nil {
			return nil, err
		}
	}

	var b bytes.Buffer

	// Callback function to save data instead of redirecting it into stdout.
	writeToBufferFunc := func(buf []byte, userdata interface{}) bool {
		if silent == false {
			// Check length of buffer b. If it is greater than
			if int64(b.Len()) > responseSize {
				// No more writing we are all done
				// If this interrupts the stream of data then we throw not a JSON endpoint error.
				sizeError = true
				return true
			} else {
				b.Write([]byte(buf))
			}
		}
		return true
	}

	this.myCurl.Setopt(curl.OPT_WRITEFUNCTION, writeToBufferFunc)
	this.myCurl.Setopt(curl.OPT_WRITEDATA, b)

	if err := this.myCurl.Perform(); err != nil {
		if show_error == true {
			return nil, err
		} else {
			return nil, nil
		}
	}

	if sizeError {
		return nil, fmt.Errorf("Response Size has been exceeded. The max response capacity is %v", responseSize)
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

func (this *Curl) simpleGet(url string) {
	this.myCurl.Setopt(curl.OPT_URL, url)
	this.myCurl.Setopt(curl.OPT_HTTPGET, 1)
}

func (this *Curl) curlAuth(val string) {
	if val == "" {
		this.myCurl.Setopt(curl.OPT_USERPWD, "")
	} else {
		val = val[1 : len(val)-1]
		if !strings.Contains(val, ":") {
			// Append an empty password if there isnt one
			val = val + ":" + ""
		}
		this.myCurl.Setopt(curl.OPT_USERPWD, val)
	}
}

func (this *Curl) curlCiphers(val string) error {
	nVal := strings.TrimSpace(val)
	if nVal != "" {
		this.myCurl.Setopt(curl.OPT_SSL_CIPHER_LIST, nVal)
	} else {
		cbCiphers := ""
		if runtime.GOOS != "windows" {
			// Get the Ciphers supported by couchbase based on the level set
			tlsCfg, err := cbauth.GetTLSConfig()
			if err != nil {
				return fmt.Errorf("Failed to get cbauth tls config: %v", err.Error())
			}
			cbCiphers = strings.Join(tlsCfg.CipherSuiteOpenSSLNames, ",")
		} else {
			cbCiphers = _WINCIPHERS
		}
		this.myCurl.Setopt(curl.OPT_SSL_CIPHER_LIST, cbCiphers)
	}
	return nil
}

func setResponseSize(maxSize int64) int64 {
	/*
			 get the first 200 bytes
			 curl_easy_setopt(curl, CURLOPT_RANGE, "0-199")

			 The unfortunate part is that for HTTP, CURLOPT_RANGE is not always enforced.
			 In this case we want to be able to still restrict the amount of data written
			 to the buffer.

			 For now we shall not use this. In the future, if the option becomes enforced
			 for HTTP then it can be used.

			 finalRange := "0-" + fmt.Sprintf("%s", MIN_REQUEST_SIZE)
		     finalRange = "0-" + fmt.Sprintf("%s", MAX_REQUEST_SIZE)
		     finalRange = "0-" + fmt.Sprintf("%s", maxSize)

		     this.myCurl.Setopt(curl.OPT_RANGE, finalRange)
	*/
	// Max Value = 64MB
	// Min Value = 20MB
	if maxSize > MAX_RESPONSE_SIZE {
		return MAX_RESPONSE_SIZE
	} else if maxSize < MIN_RESPONSE_SIZE {
		return MIN_RESPONSE_SIZE
	}
	return maxSize

}

func allowlistCheck(list map[string]interface{}, urlP string) error {
	// Structure is as follows
	// {
	//  "all_access":true/false,
	//  "allowed_urls":[ list of urls ],
	//  "disallowed_urls":[ list of urls ],
	// }

	urlParsed, err := url.Parse(urlP)
	if err != nil {
		return err
	}

	if strings.Contains(urlParsed.Path, "/diag/eval") {
		return fmt.Errorf("Access restricted - %v.", urlP)

	}

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
	// Restricted access based on fields allowed_urls and disallowed_urls

	if disallowedUrls, ok_dall := list["disallowed_urls"]; ok_dall {
		dURL, ok := disallowedUrls.([]interface{})
		if !ok {
			return fmt.Errorf("Restrict access with disallowed urls")
		}
		if len(dURL) > 0 {
			disallow, err := sliceContains(dURL, urlP)
			if err == nil && disallow {
				return fmt.Errorf("The endpoint " + urlP + " is not permitted")
			} else {
				if err != nil {
					return err
				}
			}
		}
	}

	if allowedUrls, ok_all := list["allowed_urls"]; ok_all {
		alURL, ok := allowedUrls.([]interface{})
		if !ok {
			return fmt.Errorf("allowed_urls should be list of urls present in the allowedlists.")
		}
		if len(alURL) > 0 {
			allow, err := sliceContains(alURL, urlP)
			if err == nil && allow {
				return nil
			} else {
				if err != nil {
					return err
				}
			}
		}
	}

	// URL is not present in disallowed url and is not in allowed_urls.
	// If it reaches here, then the url isnt in the allowed_urls or the prefix_urls, and is also
	// not in the disallowed urls.
	return fmt.Errorf("The end point " + urlP + " is not permitted.  List allowed end points in the configuration.")

}

// Check if urls fields in allowlist contain the input url
func sliceContains(field []interface{}, url string) (bool, error) {
	for _, val := range field {
		nVal, ok := val.(string)
		if !ok {
			return false, fmt.Errorf("Both allowed urls and disallowed urls should be list of url strings.")
		}
		// Check if list of values is a prefix of input url
		if strings.HasPrefix(url, nVal) {
			return true, nil
		}
	}
	return false, nil
}

func (this *Curl) handleData(encodedData bool, val interface{}, show_error bool) (string, error) {
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

	for _, data := range dataVal.([]interface{}) {
		newDval := value.NewValue(data)
		if newDval.Type() != value.STRING {
			if show_error == true {
				return "", fmt.Errorf("Incorrect type for data option. ")
			} else {
				return "", nil
			}
		}

		dataT := newDval.ToString()

		// If the option is data-urlencode then encode the data first.
		if encodedData {
			// When we encode strings, = should not be encoded.
			// The curl.Escape() method for go behaves different to the libcurl method.
			// q=select 1 should be q=select%201 and not q%3Dselect%201
			// Hence split the string, encode and then rejoin.
			stringComponent := strings.Split(dataT, "=")
			for i, _ := range stringComponent {
				stringComponent[i] = this.myCurl.Escape(stringComponent[i])
			}

			dataT = strings.Join(stringComponent, "=")
		}

		if stringData == "" {
			stringData = dataT
		} else {
			stringData = stringData + "&" + dataT
		}

	}
	return stringData, nil
}
