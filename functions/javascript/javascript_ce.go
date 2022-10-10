//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !enterprise !go1.10

package javascript

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
	"github.com/gorilla/mux"
)

// this body is used to fail function creation where not supported
type javascriptBody struct {
}

func Init(mix *mux.Router) {
}

func NewJavascriptBody(library, object string) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported()
}

func (this *javascriptBody) Lang() functions.Language {
	return functions.GOLANG
}

// this will never be called, just a placeholder
func (this *javascriptBody) Body(object map[string]interface{}) {
	object["functions_feature_disabled"] = true
}

//ditto
func (this *javascriptBody) SetVars(vars []string) {
}

func (this *javascriptBody) Indexable() value.Tristate {
	return value.FALSE
}

// ditto, for tests
func MakeJavascript(name functions.FunctionName, body []byte) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported()
}
