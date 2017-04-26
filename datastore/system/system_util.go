//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package system

import (
	"fmt"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type compiledSpan struct {
	low      string
	high     string
	evalLow  func(val, key string) bool
	evalHigh func(val, key string) bool

	// golang does not allow to compare functional pointers to functions...
	equality bool
}

func compileSpan(span *datastore.Span) (*compiledSpan, errors.Error) {
	var err errors.Error
	var isLowValued, isHighValued bool

	// currently system indexes are either primary or on a single field
	if len(span.Seek) > 1 || len(span.Range.Low) > 1 || len(span.Range.High) > 1 {
		return nil, errors.NewSystemDatastoreError(nil, "Invalid number of fields in span")
	}

	spanEvaluator := &compiledSpan{}
	if span.Seek != nil {
		val := span.Seek[0].Actual()
		switch t := val.(type) {
		case string:
		default:
			return nil, errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", t, val))
		}
		spanEvaluator.low = val.(string)
		spanEvaluator.high = spanEvaluator.low
		spanEvaluator.evalLow = equals
		spanEvaluator.evalHigh = noop
		spanEvaluator.equality = true
	} else {
		spanEvaluator.low, spanEvaluator.evalLow, isLowValued, err = compileRange(span.Range.Low, span.Range.Inclusion, datastore.LOW)
		if err != nil {
			return nil, err
		}
		spanEvaluator.high, spanEvaluator.evalHigh, isHighValued, err = compileRange(span.Range.High, span.Range.Inclusion, datastore.HIGH)
		if err != nil {
			return nil, err
		}
		if spanEvaluator.high == spanEvaluator.low && isLowValued && isHighValued {
			spanEvaluator.evalLow = equals
			spanEvaluator.evalHigh = noop
			spanEvaluator.equality = true
		}
	}
	return spanEvaluator, nil
}

func (this *compiledSpan) evaluate(key string) bool {
	return this.evalHigh(this.high, key) && this.evalLow(this.low, key)
}

func (this *compiledSpan) isEquals() bool {
	return this.equality
}

func (this *compiledSpan) key() string {
	return this.low
}

func compileRange(in value.Values, incl, side datastore.Inclusion) (string, func(string, string) bool, bool, errors.Error) {
	if len(in) == 0 {
		return "", noop, false, nil
	}
	val := in[0].Actual()
	t := in[0].Type()
	switch t {
	case value.STRING:
	case value.NULL:

		// > null is a noop, < null should never occur and it's an error
		if side == datastore.LOW {
			return "", noop, false, nil
		}
		fallthrough
	default:
		return "", nil, false, errors.NewSystemDatastoreError(nil, fmt.Sprintf("Invalid seek value %v of type %T.", val, t.String()))
	}
	retVal := val.(string)
	op := (incl & side) > 0
	if side == datastore.HIGH {
		if op {
			return retVal, lessOrEqual, true, nil
		} else {
			return retVal, less, true, nil
		}
	} else {
		if op {
			return retVal, greaterOrEqual, true, nil
		} else {
			return retVal, greater, true, nil
		}
	}
}

func equals(val, key string) bool {
	return key == val
}

func less(top, key string) bool {
	return key < top
}

func lessOrEqual(top, key string) bool {
	return key <= top
}

func greater(bottom, key string) bool {
	return key > bottom
}

func greaterOrEqual(bottom, key string) bool {
	return key >= bottom
}

func noop(val, key string) bool {
	return true
}

// Return the credentials presented in the context.
// The second parameter is the ns-server-auth-token value, from the original request,
// if one is present, else the empty string.
func credsFromContext(context datastore.QueryContext) (distributed.Creds, string) {
	credentials := context.Credentials()
	creds := make(distributed.Creds, len(credentials))
	for k, v := range credentials {
		creds[k] = v
	}
	authToken := ""
	req := context.OriginalHttpRequest()
	if req != nil && req.Header.Get("ns-server-ui") == "yes" {
		authToken = req.Header.Get("ns-server-auth-token")
	}
	return creds, authToken
}
