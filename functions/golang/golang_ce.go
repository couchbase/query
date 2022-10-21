//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise || !go1.10 || windows || solaris

package golang

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/value"
)

// this body is used to fail function creation where not supported
type golangBody struct {
}

func Init() {
}

func NewGolangBody(library, object string) (functions.FunctionBody, errors.Error) {
	return nil, errors.NewFunctionsNotSupported("golang")
}

func (this *golangBody) Lang() functions.Language {
	return functions.GOLANG
}

// this will never be called, just a placeholder
func (this *golangBody) Body(object map[string]interface{}) {
	object["functions_feature_disabled"] = true
}

//ditto
func (this *golangBody) SetVarNames(vars []string) errors.Error {
	return nil
}

func (this *golangBody) SetStorage(context functions.Context, path []string) errors.Error {
	return nil
}

func (this *golangBody) Indexable() value.Tristate {
	return value.FALSE
}

func (this *golangBody) Test(name functions.FunctionName) errors.Error {
	return nil
}

func (this *golangBody) Load(name functions.FunctionName) errors.Error {
	return nil
}

func (this *golangBody) Unload(name functions.FunctionName) {
}
