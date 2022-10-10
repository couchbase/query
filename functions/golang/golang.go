//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build enterprise && go1.10 && !windows && !solaris

package golang

import (
	goerrors "errors"
	"fmt"
	"os"
	"plugin"

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

func (this *golang) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args, val value.Value

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

	udf, ok := obj.(func(value.Value, functions.Context) (value.Value, error))
	if !ok {
		return nil, funcBody.execError(fmt.Errorf("invalid object"), funcName)
	}

	if len(funcBody.varNames) != 0 {
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

	val, err = udf(args, context)
	if err != nil {
		return nil, funcBody.execError(err, funcName)
	} else {
		return val, nil
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

func (this *golangBody) Lang() functions.Language {
	return functions.GOLANG
}

func (this *golangBody) Body(object map[string]interface{}) {
	object["#language"] = "golang"
	object["library"] = this.library
	object["object"] = this.object
	if len(this.varNames) > 0 {
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
