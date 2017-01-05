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
	"github.com/couchbase/query/value"
)

///////////////////////////////////////////////////
//
// Curl
//
///////////////////////////////////////////////////

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
			return value.NULL_VALUE, fmt.Errorf("Error initiallizing CURL")
		}
	}
	// End libcurl easy session
	defer func() {
		if this.myCurl != nil {
			this.myCurl.Cleanup()
			this.myCurl = nil
		}
	}()

	// Method - GET or POST
	method := args[0]

	// URL
	first := args[1]
	if method.Type() == value.MISSING || first.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if method.Type() != value.STRING || first.Type() != value.STRING {
		return value.NULL_VALUE, nil
	}

	curl_method := method.Actual().(string)

	// CURL URL
	curl_url := first.Actual().(string)

	// Empty options to pass into curl.
	options := map[string]interface{}{}

	// If we have options then process them.
	if len(args) == 3 {
		second := args[2]

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
	result, err := this.handleCurl(curl_method, curl_url, options)

	if err != nil {
		return value.NULL_VALUE, err
	}

	// For Silent mode where we dont want any output.
	if len(result) == 0 {
		return value.MISSING_VALUE, nil
	}

	return value.NewValue(result), nil
}

func (this *Curl) MinArgs() int { return 2 }

func (this *Curl) MaxArgs() int { return 3 }

/*
Factory method pattern.
*/
func (this *Curl) Constructor() FunctionConstructor {
	return NewCurl
}

func (this *Curl) handleCurl(curl_method string, url string, options map[string]interface{}) (map[string]interface{}, error) {
	// Handle different cases

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

	if curl_method != "GET" && curl_method != "POST" {
		if show_error == true {
			// This method is not supported. Hence return a null value and an error
			return nil, fmt.Errorf(" Method not supported ")
		} else {
			return nil, nil
		}
	}

	// When we dont have options, but only have the Method and URL.
	if len(options) == 0 {
		if curl_method == "GET" {
			this.simpleGet(url)
		} else {
			this.simplePost(url, "")
		}

	} else {
		// We have options that have been set.
		for k, val := range options {
			// Only support valid options.
			switch k {
			/*
				show_error: Do not output the errors with the CURL function
				in case this is set. This is handled in the beginning.
			*/
			case "show-error":
				break
			/*
				get: Send the -d data with a HTTP GET (H)
				Since we set the curl method as the first argument, it is
				important to note that providing this option does nothing.
			*/
			case "get":
				break
			/*
			   request: Specify request method to use. Since we set
			   the curl method as the first argument, it is important
			   to note that providing this option does nothing.
			*/
			case "request":
				break
			/*
				data: HTTP POST data (H). However in some cases in CURL
				this can be issued with a GET as well. In these cases, the
				data is appended to the URL followed by a ?.
			*/
			case "data":
				stringData := ""

				if value.NewValue(val).Type() == value.OBJECT {
					bytval, _ := value.NewValue(val).MarshalJSON()
					stringData = string(bytval)

				} else if value.NewValue(val).Type() == value.STRING {
					stringData = value.NewValue(val).Actual().(string)

				} else {
					stringData = value.NewValue(val).String()
				}

				if curl_method == "GET" {
					this.simpleGet(url + "?" + stringData)
				} else {
					this.simplePost(url, stringData)
				}
			/*
				header: Pass custom header to server (H). It has to be a string,
				otherwise we error out.
			*/
			case "header":
				if value.NewValue(val).Type() != value.STRING {
					if show_error == true {
						return nil, fmt.Errorf(" Incorrect type for header option in CURL ")
					} else {
						return nil, nil
					}

				}
				this.curlHeader(value.NewValue(val).Actual().(string))
			/*
				silent: Do not output anything. It has to be a boolean, otherwise
				we error out.
			*/
			case "silent":
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
			case "connect-timeout":
				if value.NewValue(val).Type() != value.NUMBER {
					return nil, fmt.Errorf(" Incorrect type for connect-timeout option in CURL ")
				}
				this.curlConnectTimeout(value.NewValue(val).Actual().(float64))
			/*
				max-time: Maximum time allowed for the transfer in seconds
			*/
			case "max-time":
				if value.NewValue(val).Type() != value.NUMBER {
					return nil, fmt.Errorf(" Incorrect type for max-time option in CURL ")
				}
				this.curlMaxTime(value.NewValue(val).Actual().(float64))
			/*
				user: Server user and password separated by a :. By default if a
				password is not specified, then use an empty password.
			*/
			case "user":
				this.curlAuth(value.NewValue(val).String())
			/*
				basic: Use HTTP Basic Authentication. It has to be a boolean, otherwise
				we error out.
			*/
			case "basic":
				if value.NewValue(val).Type() != value.BOOLEAN {
					if show_error == true {
						return nil, fmt.Errorf(" Incorrect type for basic option in CURL ")
					} else {
						return nil, nil
					}
				}
				if value.NewValue(val).Actual().(bool) == true {
					var CURLAUTH_BASIC = (1 << 0) /* Basic (default) */
					this.myCurl.Setopt(curl.OPT_HTTPAUTH, CURLAUTH_BASIC)
				}
			/*
				anyauth: curl to figure out authentication method by itself, and use the most secure one.
				It has to be a boolean, otherwise we error out.
			*/
			case "anyauth":
				if value.NewValue(val).Type() != value.BOOLEAN {
					if show_error == true {
						return nil, fmt.Errorf(" Incorrect type for anyauth option in CURL ")
					} else {
						return nil, nil
					}
				}
				if value.NewValue(val).Actual().(bool) == true {
					var CURLAUTH_ANY = ^(0) /* all types set */
					this.myCurl.Setopt(curl.OPT_HTTPAUTH, CURLAUTH_ANY)
				}
			/*
				insecure: Allow connections to SSL sites without certs (H). It has to be a boolean,
				otherwise we error out.
			*/
			case "insecure":
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
				keepalive-time: TODO.
			*/
			case "keepalive-time":
				if value.NewValue(val).Type() != value.NUMBER {
					return nil, fmt.Errorf(" Incorrect type for keepalive-time option in CURL ")
				}
				this.curlKeepAlive(value.NewValue(val).Actual().(float64))
			/*
				max-redirs: Maximum number of redirects allowed (H)
			*/
			case "max-redirs":
				if value.NewValue(val).Type() != value.NUMBER {
					return nil, fmt.Errorf(" Incorrect type for max-redirs option in CURL ")
				}
				this.curlMaxRedirs(value.NewValue(val).Actual().(float64))
			}

		}

	}

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

	if b.Len() != 0 {
		var dat map[string]interface{}

		if err := json.Unmarshal(b.Bytes(), &dat); err != nil {
			if show_error == true {
				return nil, err
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

func (this *Curl) simplePost(url string, data string) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_URL, url)
	myCurl.Setopt(curl.OPT_POST, true)
	myCurl.Setopt(curl.OPT_POSTFIELDS, data)
}

func (this *Curl) curlHeader(header string) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_HTTPHEADER, []string{header})
}

func (this *Curl) curlAuth(val string) {
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

func (this *Curl) curlConnectTimeout(val float64) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_CONNECTTIMEOUT, val)

}

func (this *Curl) curlMaxTime(val float64) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_TIMEOUT, val)
}

func (this *Curl) curlMaxRedirs(val float64) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_MAXREDIRS, val)
}

func (this *Curl) curlKeepAlive(val float64) {
	myCurl := this.myCurl
	myCurl.Setopt(curl.OPT_TCP_KEEPALIVE, 1.0)
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
