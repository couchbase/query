//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// +build enterprise,go1.10

package javascript

import (
	goerrors "errors"
	"fmt"
	"sync/atomic"

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
const _MIN_SERVICERS = 6
const _MAX_SERVICERS = 96

type javascript struct {
}

type javascriptBody struct {
	varNames []string
	library  string
	object   string
}

var enabled = true
var evaluator defs.Evaluator
var threads int32
var threadCount int32

// FIXME to be sorted
// - evaluator client does not yet take context
// - indexing issues: identify functions as deterministic and not running N1QL code

func Init(mux *mux.Router, t int) {
	functions.FunctionsNewLanguage(functions.JAVASCRIPT, &javascript{})

	if t < _MIN_SERVICERS {
		t = _MIN_SERVICERS
	} else if t > _MAX_SERVICERS {
		t = _MAX_SERVICERS
	}
	engine := n1ql_client.SingleInstance
	config := make(map[defs.Config]interface{})
	config[defs.Threads] = t
	config[defs.FeatureBitmask] = uint32(8)
	config[defs.IsIpV6] = false
	threads = int32(t)

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

	// FIXME context
	// the runners take timeouts in milliseconds
	timeout := int(context.GetTimeout().Milliseconds())
	if timeout == 0 || timeout > _MAX_TIMEOUT {
		timeout = _MAX_TIMEOUT
	}
	opts := map[defs.Option]interface{}{defs.SideEffects: (modifiers & functions.READONLY) == 0, defs.Timeout: timeout}
	runners := atomic.AddInt32(&threadCount, 1)
	defer atomic.AddInt32(&threadCount, -1)
	levels := context.IncRecursionCount(1)
	defer context.IncRecursionCount(-1)

	// make sure that requests don't flood the runners by calling nested functions too deeply
	// the higher the load, the lower the threshold
	if levels > 1 && levels > int(threads-runners) {
		return nil, errors.NewFunctionExecutionNestedError(levels, funcName)
	}
	res, err := evaluator.Evaluate(funcBody.library, funcBody.object, opts, args, functions.NewUdfContext(context))
	if err.Err != nil {
		return nil, funcBody.execError(err, funcName)
	} else {
		return value.NewValue(res), nil
	}
}

func (this *javascriptBody) execError(err defs.Error, name string) errors.Error {
	if err.IsNestedErr {
		return errors.NewInnerFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
			name, fmt.Errorf("%v", err.Details))
	}
	return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
		name, fmt.Errorf("%v %v", err.Err, err.Details))
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

func (this *javascriptBody) SwitchContext() value.Tristate {
	return value.NONE
}

func (this *javascriptBody) IsExternal() bool {
	return true
}

func (this *javascriptBody) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}
