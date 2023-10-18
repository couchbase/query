//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise && go1.10 && !windows && !solaris
// +build enterprise,go1.10,!windows,!solaris

package golang

import (
	goerrors "errors"
	"fmt"
	"os"
	"plugin"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type golang struct {
}

type golangBody struct {
	varNames []string
	library  string
	object   string
}

var _PATH string
var enabled = true

func Init() {
	functions.FunctionsNewLanguage(functions.GOLANG, &golang{})

	// only enable golang udfs if can determine our own absolute path
	p, _ := os.Getwd()
	if p != "" {
		_PATH = p + "/udf/"
	} else {
		enabled = false
	}
}

func (this *golang) CheckAuthorize(name string, context functions.Context) bool {
	return true
}

func (this *golang) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args value.Value
	var val interface{}

	funcName := name.Name()
	funcBody, ok := body.(*golangBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	if !enabled || !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_GOLANG_UDF) {
		return nil, errors.NewFunctionsDisabledError("golang")
	}

	path := _PATH + funcBody.library
	handle, err := plugin.Open(path)
	if err != nil {
		return nil, funcBody.execError(err, funcName)
	}
	obj, err := handle.Lookup(funcBody.object)
	if err != nil {
		return nil, funcBody.execError(err, funcName)
	}

	udf, ok := obj.(func(interface{}, interface{}) (interface{}, error))
	if !ok {
		return nil, funcBody.execError(fmt.Errorf("invalid object"), funcName)
	}

	if funcBody.varNames != nil {
		if len(values) != len(funcBody.varNames) {
			return nil, errors.NewArgumentsMismatchError(funcName)
		}
		argsObj := make(map[string]interface{}, len(values))
		for i, _ := range values {
			argsObj[funcBody.varNames[i]] = values[i]
		}
		args = value.NewValue(argsObj)
	} else {
		args = value.NewValue(values)
	}

	val, err = udf(args, functions.NewUdfContext(context, ""))
	if err != nil {
		return nil, funcBody.execError(err, funcName)
	} else {
		return value.NewValue(val), nil
	}
}

func (this *golangBody) execError(err error, name string) errors.Error {
	return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
		name, err)
}

func NewGolangBody(library, object string) (functions.FunctionBody, errors.Error) {
	if !enabled || !util.IsFeatureEnabled(util.GetN1qlFeatureControl(), util.N1QL_GOLANG_UDF) {
		return nil, errors.NewFunctionsDisabledError("golang")
	}
	return &golangBody{library: library, object: object}, nil
}

func (this *golangBody) SetVarNames(vars []string) errors.Error {
	this.varNames = vars
	return nil
}

func (this *golangBody) SetStorage(context functions.Context, path []string) errors.Error {
	return nil
}

func (this *golangBody) Lang() functions.Language {
	return functions.GOLANG
}

func (this *golangBody) Body(object map[string]interface{}) {
	object["#language"] = "golang"
	object["library"] = this.library
	object["object"] = this.object
	if this.varNames != nil {
		vars := make([]value.Value, len(this.varNames))
		for v, _ := range this.varNames {
			vars[v] = value.NewValue(this.varNames[v])
		}
		object["parameters"] = vars
	}
}

func (this *golangBody) Indexable() value.Tristate {

	// for now
	return value.FALSE
}

func (this *golangBody) SwitchContext() value.Tristate {
	return value.NONE

}

func (this *golangBody) IsExternal() bool {
	return true
}

func (this *golangBody) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}
