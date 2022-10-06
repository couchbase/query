//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build !enterprise
// +build !enterprise

package inline

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

// this body is used to fail function creation outside of EE
type inlineBody struct {
}

func Init() {
}

func NewInlineBody(expr expression.Expression) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported()
}

func (this *inlineBody) Lang() functions.Language {
	return functions.INLINE
}

// this will never be called, just a placeholder
func (this *inlineBody) Body(object map[string]interface{}) {
	object["functions_feature_disabled"] = true
}

//ditto
func (this *inlineBody) SetVars(vars []string) {
}

func (this *inlineBody) Indexable() value.Tristate {
	return value.FALSE
}

// ditto, for tests
func MakeInline(name functions.FunctionName, body []byte) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported()
}
