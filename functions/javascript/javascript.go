//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//  build enterprise,go1.10

// +build ignore

package javascript

import (
	goerrors "errors"
	"fmt"

	"github.com/couchbase/eventing-ee/js-evaluator/evaluator-client/client"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

type javascript struct {
}

type javascriptBody struct {
	varNames []string
	library  string
	object   string
}

var enabled = true
var evaluatorClient client.EvaluatorClient

// FIXME to be sorted
// - can't have the evaluator created by the client - what about projector and query on the same node?
// - evaluator client does not yet take arguments
// - evaluator client does not yet take credentials
// - indexing issues: identify functions as deterministic and not running N1QL code
// - deadly embrace between evaluator an n1ql services consuming each other's processes

func Init() {
	var err error

	functions.FunctionsNewLanguage(functions.JAVASCRIPT, &javascript{})
	evaluatorClient, err := client.NewEvaluatorClient(&adapter.Configuration{
		WorkersPerNode:   2,
		ThreadsPerWorker: 3,
		HttpPort:         port.Port(9090),
		NsServerUrl:      "http://locahost:9000",
	})
	if err != nil {
		logging.Infof("Unable to start javascript evaluator client, err : %v", err)
		enabled = false
	}
}

func (this *javascript) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args, val value.Value

	if !enabled {
		return nil, errors.NewFunctionsDisabledError("javascript")
	}

	funcName := name.Name()
	funcBody, ok := body.(*javascriptBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	/* FIXME currently not supported by the evaluator API
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
	*/
	// FIXME context, credentials

	val, err = evaluatorClient.Evaluate(funcBody.library, funcBody.object)
	if err != nil {
		return nil, funcBody.execError(err, funcName)
	} else {
		return val, nil
	}
}

func (this *javascriptBody) execError(err error, name string) errors.Error {
	return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
		name, err)
}

func NewJavascriptBody(library, object string) (functions.FunctionBody, errors.Error) {
	if !enabled {
		return nil, errors.NewFunctionsDisabledError("javascript")
	}
	return &javascriptBody{library: library, object: object}, nil
}

func (this *javascriptBody) SetVarNames(vars []string) errors.Error {
	this.varNames = vars
	return nil
}

func (this *javascriptBody) Lang() functions.Language {
	return functions.JAVASCRIPT
}

func (this *javascriptBody) Body(object map[string]interface{}) {
	object["#language"] = "javascript"
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

func (this *javascriptBody) Indexable() value.Tristate {

	// for now
	return value.FALSE
}
