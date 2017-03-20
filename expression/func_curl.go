//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	curl "github.com/andelf/go-curl"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
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
const (
	_N1QL_USER_AGENT = "couchbase/n1ql/" + util.VERSION
)

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

	rv.volatile = true
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

func (this *Curl) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.Eval(this, item, context)
}

func (this *Curl) Apply(context Context, args ...value.Value) (value.Value, error) {

	if this.myCurl == nil {
		this.myCurl = curl.EasyInit()
		if this.myCurl == nil {
			return value.NULL_VALUE, fmt.Errorf("Error initializing libcurl")
		}
	}
	// End libcurl easy session
	defer func() {
		if this.myCurl != nil {
			this.myCurl.Cleanup()
			this.myCurl = nil
		}
	}()

	// URL
	first := args[0]
	if first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	// CURL URL
	curl_url := first.Actual().(string)

	// Empty options to pass into curl.
	options := map[string]interface{}{}

	// If we have options then process them.
	if len(args) == 2 {
		second := args[1]

		if second.Type() == value.MISSING {
			return value.MISSING_VALUE, nil
		} else if second.Type() == value.OBJECT {
			//Process the options
			options = second.Actual().(map[string]interface{})
		} else {
			return value.NULL_VALUE, nil
		}
	}

	// Now you have the URL and the options with which to call curl.
	result, err := this.handleCurl(curl_url, options)

	if err != nil {
		return value.NULL_VALUE, err
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

func (this *Curl) handleCurl(url string, options map[string]interface{}) (interface{}, error) {
	// Handle different cases

	// For data method
	getMethod := false
	dataOp := false
	stringData := ""

	// For silent mode
	silent := false

	// To show errors encountered when executing the CURL function.
	show_error := true

	showErrVal, ok := options["show_error"]
	if ok {
		if value.NewValue(showErrVal).Type() != value.BOOLEAN {
			return nil, fmt.Errorf(" Incorrect type for show_error option in CURL ")
		}
		show_error = value.NewValue(showErrVal).Actual().(bool)
	}

	// Set MAX_REDIRS to 0 as an added precaution to disable redirection.
	/*
		Libcurl code to set MAX_REDIRS
		curl_easy_setopt(hnd, CURLOPT_MAXREDIRS, 50L);
	*/
	this.myCurl.Setopt(curl.OPT_MAXREDIRS, 0)

	// Set what protocols are allowed.
	/*
		CURL.H  CURLPROTO_ defines are for the CURLOPT_*PROTOCOLS options
		#define CURLPROTO_HTTP   (1<<0)
		#define CURLPROTO_HTTPS  (1<<1)

		Libcurl code to set what protocols are allowed.
		curl_easy_setopt(curl, CURLOPT_PROTOCOLS,CURLPROTO_HTTP | CURLPROTO_HTTPS);
	*/
	this.myCurl.Setopt(curl.OPT_PROTOCOLS, _CURLPROTO_HTTP|_CURLPROTO_HTTPS)

	// Prepare a header []string - slist1 as per libcurl.
	header := []string{}

	// Set curl User-Agent by default.
	this.curlUserAgent(_N1QL_USER_AGENT)

	// When we dont have options, but only have the URL.
	/*
		Libcurl code to set the url
		curl_easy_setopt(hnd, CURLOPT_URL, "https://api.github.com/users/ikandaswamy/repos");
	*/
	this.myCurl.Setopt(curl.OPT_URL, url)

	for k, val := range options {
		// Only support valid options.
		switch k {
		/*
			show_error: Do not output the errors with the CURL function
			in case this is set. This is handled in the beginning.
		*/
		case "show-error", "--show-error", "S", "-S":
			break
		/*
			get: Send the -d data with a HTTP GET (H)
			Since we set the curl method as the first argument, it is
			important to note that providing this option does nothing.
		*/
		case "get", "--get", "G", "-G":
			if value.NewValue(val).Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for get option in CURL ")
				} else {
					return nil, nil
				}
			}
			get := value.NewValue(val).Actual().(bool)
			if get {
				getMethod = true
				this.simpleGet(url)
			}

		/*
		   request: Specify request method to use. Since we set
		   the curl method as the first argument, it is important
		   to note that providing this option does nothing.
		*/
		case "request", "--request", "X", "-X":
			request := value.NewValue(val)
			if request.Type() != value.STRING {
				return nil, fmt.Errorf(" Incorrect type for request option in CURL. It should be a string. ")
			}

			// Remove the quotations at the end.
			requestVal := request.String()
			requestVal = requestVal[1 : len(requestVal)-1]

			// Methods are case sensitive.
			if requestVal != "GET" && requestVal != "POST" {
				if show_error == true {
					return nil, fmt.Errorf(" CURL only supports GET and POST requests. ")
				} else {
					return nil, nil
				}
			}

			if requestVal == "GET" {
				getMethod = true
			}

			/*
				Libcurl code to handle requests is
				curl_easy_setopt(hnd, CURLOPT_CUSTOMREQUEST, "POST");
			*/
			this.myCurl.Setopt(curl.OPT_CUSTOMREQUEST, requestVal)

		/*
			data: HTTP POST data (H). However in some cases in CURL
			this can be issued with a GET as well. In these cases, the
			data is appended to the URL followed by a ?.
		*/
		case "data", "--data", "d", "-d":

			dataVal := value.NewValue(val).Actual()

			switch dataVal.(type) {
			case []interface{}:
			case string:
				dataVal = []interface{}{dataVal}
			default:
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for data option in CURL.It needs to be a string. ")
				} else {
					return nil, nil
				}
			}

			for _, data := range dataVal.([]interface{}) {
				newDval := value.NewValue(data)
				if newDval.Type() != value.STRING {
					if show_error == true {
						return nil, fmt.Errorf(" Incorrect type for data option. ")
					} else {
						return nil, nil
					}
				}

				dataT := newDval.Actual().(string)
				if stringData == "" {
					stringData = dataT
				} else {
					stringData = stringData + "&" + dataT
				}

			}
			dataOp = true

		/*
			header: Pass custom header to server (H). It has to be a string,
			otherwise we error out.
		*/
		case "headers", "header", "--header", "--headers", "H", "-H":
			/*
				Libcurl code to handle multiple headers using the --header and -H options.

				  struct curl_slist *slist1;
				  slist1 = NULL;
				  slist1 = curl_slist_append(slist1, "X-N1QL-User-Agent: couchbase/n1ql/1.7.0");
				  slist1 = curl_slist_append(slist1, "User-Agent: ikandaswamy");
			*/
			// Get the value
			headerVal := value.NewValue(val).Actual()

			switch headerVal.(type) {

			case []interface{}:
				//Do nothing
			case string:
				headerVal = []interface{}{headerVal}

			default:
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for header option " + value.NewValue(val).String() + " in CURL. Header option should be a string value or an array of strings.  ")
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
						return nil, fmt.Errorf(" Incorrect type for header option " + newHval.String() + " in CURL. Header option should be a string value or an array of strings.  ")
					} else {
						return nil, nil
					}

				}
				head := newHval.String()
				header = append(header, head[1:len(head)-1])
			}

		/*
			silent: Do not output anything. It has to be a boolean, otherwise
			we error out.
		*/
		case "silent", "--silent", "s", "-s":
			if value.NewValue(val).Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for silent option in CURL ")
				} else {
					return nil, nil
				}
			}
			silent = value.NewValue(val).Actual().(bool)
		/*
			connect-timeout: Maximum time allowed for connection in seconds
		*/
		case "connect-timeout", "--connect-timeout":
			/*
				Libcurl code to set connect-timeout is
				curl_easy_setopt(hnd, CURLOPT_CONNECTTIMEOUT_MS, 1000L);

				To save fractions of the decimal value, libcurl uses the _MS suffix to convert
				to milliseconds.
			*/
			if value.NewValue(val).Type() != value.NUMBER {
				return nil, fmt.Errorf(" Incorrect type for connect-timeout option in CURL ")
			}

			connTime := value.NewValue(val).Actual().(float64)

			this.curlConnectTimeout(int64(connTime))
		/*
			max-time: Maximum time allowed for the transfer in seconds
		*/
		case "max-time", "--max-time", "m", "-m":
			/*
				Libcurl code to set max-time is
				curl_easy_setopt(hnd, CURLOPT_TIMEOUT_MS, 1000L);

				To save fractions of the decimal value, libcurl uses the _MS suffix to convert
				to milliseconds.
			*/
			if value.NewValue(val).Type() != value.NUMBER {
				return nil, fmt.Errorf(" Incorrect type for max-time option in CURL ")
			}

			maxTime := value.NewValue(val).Actual().(float64)

			this.curlMaxTime(int64(maxTime))
		/*
			user: Server user and password separated by a :. By default if a
			password is not specified, then use an empty password.
		*/
		case "user", "--user", "-u", "u":
			/*
				Libcurl code to set user
				curl_easy_setopt(hnd, CURLOPT_USERPWD, "Administrator:password");
			*/
			if value.NewValue(val).Type() != value.STRING {
				return nil, fmt.Errorf(" Incorrect type for user option in CURL. It should be a string. ")
			}
			this.curlAuth(value.NewValue(val).String())
		/*
			basic: Use HTTP Basic Authentication. It has to be a boolean, otherwise
			we error out.
		*/
		case "basic", "--basic":
			/*
				Libcurl code to set --basic
				#define CURLAUTH_BASIC (1<<0) /* Basic (default)
				curl_easy_setopt(hnd, CURLOPT_HTTPAUTH, (long)CURLAUTH_BASIC);
			*/

			if value.NewValue(val).Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for basic option in CURL ")
				} else {
					return nil, nil
				}
			}
			if value.NewValue(val).Actual().(bool) == true {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_BASIC)
			}
		/*
			anyauth: curl to figure out authentication method by itself, and use the most secure one.
			It has to be a boolean, otherwise we error out.
		*/
		case "anyauth", "--anyauth":
			/*
				Libcurl code to set --anyauth
				#define CURLAUTH_ANY ~0
				curl_easy_setopt(hnd, CURLOPT_HTTPAUTH, (long)CURLAUTH_ANY);
			*/
			if value.NewValue(val).Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for anyauth option in CURL ")
				} else {
					return nil, nil
				}
			}
			if value.NewValue(val).Actual().(bool) == true {
				this.myCurl.Setopt(curl.OPT_HTTPAUTH, _CURLAUTH_ANY)
			}
		/*
			insecure: Allow connections to SSL sites without certs (H). It has to be a boolean,
			otherwise we error out.
		*/
		case "insecure", "--insecure", "k", "-k":
			/*
				Set the value to 1 for strict certificate check please
				curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 1L);

				If you want to connect to a site who isn't using a certificate that is
				signed by one of the certs in the CA bundle you have, you can skip the
				verification of the server's certificate. This makes the connection
				A LOT LESS SECURE.
			*/
			if value.NewValue(val).Type() != value.BOOLEAN {
				if show_error == true {
					return nil, fmt.Errorf(" Incorrect type for insecure option in CURL ")
				} else {
					return nil, nil
				}
			}
			insecure := value.NewValue(val).Actual().(bool)
			if insecure == true {
				this.myCurl.Setopt(curl.OPT_SSL_VERIFYPEER, insecure)
			}
		/*
			keepalive-time: Wait SECONDS between keepalive probes for low level TCP connectivity.
			(Does not affect HTTP level keep-alive)

		*/
		case "keepalive-time", "--keepalive-time":
			/*
				Libcurl code to set keepalive-time
				curl_easy_setopt(hnd, CURLOPT_TCP_KEEPALIVE, 1L);
				curl_easy_setopt(hnd, CURLOPT_TCP_KEEPIDLE, 1L);
				curl_easy_setopt(hnd, CURLOPT_TCP_KEEPINTVL, 1L);
			*/
			if value.NewValue(val).Type() != value.NUMBER {
				return nil, fmt.Errorf(" Incorrect type for keepalive-time option in CURL ")
			}

			kATime := value.NewValue(val).Actual().(float64)

			this.curlKeepAlive(int64(kATime))

		/*
			user-agent: Value for the User-Agent to send to the server.
		*/
		case "user-agent", "--user-agent", "A", "-A":
			/*
				Libcurl code to set user-agent
				curl_easy_setopt(hnd, CURLOPT_USERAGENT, "curl/7.43.0");
			*/
			if value.NewValue(val).Type() != value.STRING {
				return nil, fmt.Errorf(" Incorrect type for user-agent option in CURL. user-agent should be a string. ")
			}
			userAgent := value.NewValue(val).Actual().(string)
			this.curlUserAgent(userAgent)

		default:
			return nil, fmt.Errorf(" CURL option %v is not supported.", k)

		}

	}

	/*
		Check if we set the request method to GET either by passing in --get or
		by saying -XGET. This will be used to decide how data is passed for the
		-data option.
	*/
	if dataOp {
		if getMethod {
			this.simpleGet(url + "?" + stringData)
		} else {
			this.curlData(stringData)
		}
	}

	/*
		Libcurl code to write data to chunk of memory
		1. Send all data to this function
		 curl_easy_setopt(curl_handle, CURLOPT_WRITEFUNCTION, writeToBufferFunc);

		2. Pass the chunk to the callback function
		 curl_easy_setopt(curl_handle, CURLOPT_WRITEDATA, (void *)&b);

		3. Define callback function - getinmemory.c example (https://curl.haxx.se/libcurl/c/getinmemory.html)
				static size_t
				WriteMemoryCallback(void *contents, size_t size, size_t nmemb, void *userp)
				{
		  			size_t realsize = size * nmemb;
		  			struct MemoryStruct *mem = (struct MemoryStruct *)userp;

		  			mem->memory = realloc(mem->memory, mem->size + realsize + 1);
		  			if(mem->memory == NULL) {
		    			// out of memory!
		    			printf("not enough memory (realloc returned NULL)\n");
		    			return 0;
		  			}

		 			memcpy(&(mem->memory[mem->size]), contents, realsize);
		 	 		mem->size += realsize;
		  			mem->memory[mem->size] = 0;

		  			return realsize;
				}

		We use the bytes.Buffer package Write method. go-curl fixes the input and output format
		of the callback function to be func(buf []byte, userdata interface{}) bool {}
	*/

	// Set the header, so that the entire []string are passed in.
	this.curlHeader(header)

	var b bytes.Buffer

	// Callback function to save data instead of redirecting it into stdout.
	writeToBufferFunc := func(buf []byte, userdata interface{}) bool {
		if silent == false {
			b.Write([]byte(buf))
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
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_URL, url)
	myCurl.Setopt(curl.OPT_HTTPGET, 1)
}

func (this *Curl) curlData(data string) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_POST, true)
	myCurl.Setopt(curl.OPT_POSTFIELDS, data)
}

func (this *Curl) curlHeader(header []string) {

	/*
		Libcurl code to handle multiple headers using the --header and -H options.
		 slist1 = curl_slist_append(slist1, "X-N1QL-Header: n1ql-1.7.0");
		 curl_easy_setopt(hnd, CURLOPT_HTTPHEADER, slist1);
	*/

	// Set the Custom N1QL Header first.
	// This will allow localhost endpoints to recognize the query service.
	header = append(header, "X-N1QL-User-Agent: "+_N1QL_USER_AGENT)
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_HTTPHEADER, header)
}

func (this *Curl) curlUserAgent(userAgent string) {
	/*
		Libcurl code to set user-agent
		curl_easy_setopt(hnd, CURLOPT_USERAGENT, "curl/7.43.0");
	*/
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_USERAGENT, userAgent)
}

func (this *Curl) curlAuth(val string) {
	/*
		Libcurl code to set username password
		curl_easy_setopt(hnd, CURLOPT_USERPWD, "Administrator:password");
	*/
	myCurl := this.myCurl
	if val == "" {
		myCurl.Setopt(curl.OPT_USERPWD, "")
	} else {
		val = val[1 : len(val)-1]
		if !strings.Contains(val, ":") {
			// Append an empty password if there isnt one
			val = val + ":" + ""
		}

		myCurl.Setopt(curl.OPT_USERPWD, val)
	}
}

func (this *Curl) curlConnectTimeout(val int64) {
	/*
		Libcurl code to set connect-timeout is
		curl_easy_setopt(hnd, CURLOPT_CONNECTTIMEOUT_MS, 1000L);

		To save fractions of the decimal value, libcurl uses the _MS suffix to convert
		to milliseconds.
	*/
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_CONNECTTIMEOUT, val)

}

func (this *Curl) curlMaxTime(val int64) {
	/*
		Libcurl code to set max-time is
		curl_easy_setopt(hnd, CURLOPT_TTIMEOUT_MS, 1000L);

		To save fractions of the decimal value, libcurl uses the _MS suffix to convert
		to milliseconds.
	*/
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_TIMEOUT, val)
}

func (this *Curl) curlKeepAlive(val int64) {
	/*
		Libcurl code to set keepalive-time
		curl_easy_setopt(hnd, CURLOPT_TCP_KEEPALIVE, 1L);
		curl_easy_setopt(hnd, CURLOPT_TCP_KEEPIDLE, 1L);
		curl_easy_setopt(hnd, CURLOPT_TCP_KEEPINTVL, 1L);
	*/
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_TCP_KEEPALIVE, 1)
	myCurl.Setopt(curl.OPT_TCP_KEEPIDLE, val)
	myCurl.Setopt(curl.OPT_TCP_KEEPINTVL, val)
}

/* Other auth values
var (
			CURLAUTH_NONE    = 0        /* nothing
			CURLAUTH_BASIC   = (1 << 0) /* Basic (default)
			CURLAUTH_DIGEST  = (1 << 1) /* Digest
			CURLAUTH_ANYSAFE = (^CURLAUTH_BASIC)
		)
*/
