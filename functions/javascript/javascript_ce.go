//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise || !go1.10

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

func NewJavascriptBody(library, object, text string) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported("javascript")
}

func NewJavascriptBodyWithDetails(library, object, prefix, name, text string) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported("javascript")
}

func (this *javascriptBody) Lang() functions.Language {
	return functions.GOLANG
}

// this will never be called, just a placeholder
func (this *javascriptBody) Body(object map[string]interface{}) {
	object["functions_feature_disabled"] = true
}

// ditto
func (this *javascriptBody) SetVarNames(vars []string) errors.Error {
	return nil
}

func (this *javascriptBody) SetStorage(context functions.Context, path []string) errors.Error {
	return nil
}

func (this *javascriptBody) Indexable() value.Tristate {
	return value.FALSE
}

func (this *javascriptBody) Test(name functions.FunctionName) errors.Error {
	return nil
}

func (this *javascriptBody) Load(name functions.FunctionName) errors.Error {
	return nil
}

func (this *javascriptBody) Unload(name functions.FunctionName) {
}
