//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

//go:build enterprise && go1.10

package javascript

import (
	goerrors "errors"
	"fmt"

	"github.com/couchbase/eventing-ee/evaluator/defs"
	"github.com/couchbase/eventing-ee/evaluator/n1ql_client"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
	"github.com/gorilla/mux"
)

// we won't let a javascript function execute more than 2 minutes
const _MAX_TIMEOUT = 120000

type javascript struct {
}

type javascriptBody struct {
	varNames []string
	library  string
	object   string
}

var enabled = true
var evaluator defs.Evaluator

// FIXME to be sorted
// - evaluator client does not yet take credentials
// - indexing issues: identify functions as deterministic and not running N1QL code
// - deadly embrace between evaluator an n1ql services consuming each other's processes

func Init(mux *mux.Router) {
	functions.FunctionsNewLanguage(functions.JAVASCRIPT, &javascript{})

	engine := n1ql_client.SingleInstance
	config := make(map[defs.Config]interface{})
	config[defs.Threads] = 6

	err := engine.Configure(config)
	if err.Err == nil {
		if mux != nil {
			handle := engine.UIHandler()
			mux.NewRoute().PathPrefix(handle.Path()).Methods("GET", "POST", "DELETE").HandlerFunc(handle.Handler())
		}
		if err.Err == nil {
			err = engine.Start()
		}
	}

	if err.Err != nil {
		logging.Infof("Unable to start javascript evaluator client, err : %v", err.Err)
		enabled = false
	} else {
		evaluator = engine.Fetch()
		if evaluator == nil {
			logging.Infof("Unable to retrieve javascript evaluator")
			enabled = false
		}
	}
}

func (this *javascript) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args []interface{}

	if !enabled {
		return nil, errors.NewFunctionsDisabledError("javascript")
	}

	funcName := name.Name()
	funcBody, ok := body.(*javascriptBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	if funcBody.varNames != nil && len(values) != len(funcBody.varNames) {
		return nil, errors.NewArgumentsMismatchError(funcName)
	}
	for i, _ := range values {
		args = append(args, values[i].Actual())
	}

	// FIXME credentials
	// FIXME queryContext
	// the runners take timeouts in milliseconds
	timeout := int(context.GetTimeout().Milliseconds())
	if timeout == 0 || timeout > _MAX_TIMEOUT {
		timeout = _MAX_TIMEOUT
	}
	opts := map[defs.Option]interface{}{defs.SideEffects: (modifiers & functions.READONLY) == 0, defs.Timeout: timeout}
	res, err := evaluator.Evaluate(funcBody.library, funcBody.object, opts, args)
	if err.Err != nil {
		return nil, funcBody.execError(err.Err, err.Details, funcName)
	} else {
		return value.NewValue(res), nil
	}
}

func (this *javascriptBody) execError(err error, details fmt.Stringer, name string) errors.Error {
	return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
		name, fmt.Errorf("%v %v", err, details))
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
	if this.varNames != nil {
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

// queryContext and readonly are resolved outside of this request
func (this *javascriptBody) SwitchContext() value.Tristate {
	return value.FALSE
}

func (this *javascriptBody) IsExternal() bool {
	return true
}

func (this *javascriptBody) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}
