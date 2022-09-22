//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise && go1.10

package javascript

import (
	goerrors "errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/couchbase/eventing-ee/evaluator/defs"
	"github.com/couchbase/eventing-ee/evaluator/n1ql_client"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
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
	prefix   string
	libName  string
	text     string // Internal JS functions have the function's JS code stored
}

var externalJSEnabled = false
var externalJSEvaluator defs.Evaluator
var threads int32
var threadCount int32
var libStore defs.LibStore
var internalJSEvaluator defs.Evaluator
var internalJSEnabled = false

// FIXME to be sorted
// - evaluator client does not yet take context
// - indexing issues: identify functions as deterministic and not running N1QL code

// TODO TENANT cleanup tenant runners

func Init(mux *mux.Router, t int) {
	functions.FunctionsNewLanguage(functions.JAVASCRIPT, &javascript{})

	if t < _MIN_SERVICERS {
		t = _MIN_SERVICERS
	} else if t > _MAX_SERVICERS {
		t = _MAX_SERVICERS
	}

	// Create the engine for external JS functions
	externalJSEngine := n1ql_client.SingleInstance
	config := make(map[defs.Config]interface{})
	config[defs.Threads] = t
	config[defs.FeatureBitmask] = uint32(8)
	config[defs.IsIpV6] = util.IPv6
	config[defs.GlobalManagePermission] = "cluster.n1ql.udf_external!manage"
	config[defs.ScopeManagePermission] = "cluster.collection[%s].n1ql.udf_external!manage"
	config[defs.SysLogLevel] = 4
	config[defs.SysLogCallback] = func(level, msg string, ctx interface{}) {
		logging.Infof("jsevaluator: %s", msg)
	}
	threads = int32(t)

	err := externalJSEngine.Configure(config)
	if err.Err == nil {
		if mux != nil {
			handle := externalJSEngine.UIHandler()
			mux.NewRoute().PathPrefix(handle.Path()).Methods("GET", "POST", "DELETE").HandlerFunc(handle.Handler())
		}
		if err.Err == nil {
			err = externalJSEngine.Start()
		}
	}

	if err.Err != nil {
		logging.Infof("Unable to start javascript evaluator client, err : %v", err.Err)
	} else {
		externalJSEvaluator = externalJSEngine.Fetch()
		if externalJSEvaluator == nil {
			logging.Infof("Unable to retrieve javascript evaluator")
		} else {
			externalJSEnabled = true
		}
	}

	// Create a new engine for Internal JS functions
	configInternalJS := make(map[defs.Config]interface{})
	configInternalJS[defs.Threads] = t
	configInternalJS[defs.FeatureBitmask] = uint32(8)
	configInternalJS[defs.IsIpV6] = util.IPv6
	configInternalJS[defs.GlobalManagePermission] = "cluster.n1ql.udf_external!manage"
	configInternalJS[defs.ScopeManagePermission] = "cluster.collection[%s].n1ql.udf_external!manage"
	configInternalJS[defs.SysLogLevel] = 4
	configInternalJS[defs.SysLogCallback] = func(level, msg string, ctx interface{}) {
		logging.Infof("Internal jsevaluator: %s", msg)
	}

	internalJSEngine := n1ql_client.NewEngine()
	err = internalJSEngine.Configure(configInternalJS)

	if err.Err == nil {
		err = internalJSEngine.Start()
	}

	if err.Err != nil {
		logging.Infof("Unable to start internal javascript evaluator client, err : %v", err.Err)
	} else {
		internalJSEvaluator = internalJSEngine.Fetch()
		if internalJSEvaluator == nil {
			logging.Infof("Unable to retrieve internal javascript evaluator")
		} else {
			internalJSEnabled = true
		}
	}

	libStore = internalJSEngine.GetLibStore()

	if libStore == nil {
		logging.Infof("libStore is nil")
		internalJSEnabled = false

	}
}

func (this *javascript) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args []interface{}

	funcName := name.Name()
	funcBody, ok := body.(*javascriptBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	if funcBody.text == "" {
		if !externalJSEnabled {
			return nil, errors.NewFunctionsDisabledError("External javascript")
		}
	} else {
		if !internalJSEnabled {
			return nil, errors.NewFunctionsDisabledError("Internal javascript")
		}
	}

	if funcBody.varNames != nil && len(values) != len(funcBody.varNames) {
		return nil, errors.NewArgumentsMismatchError(funcName)
	}
	for i, _ := range values {
		args = append(args, values[i].Actual())
	}

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

	var res interface{}
	var err defs.Error

	if funcBody.text == "" { // the function is an External JS function
		library := funcBody.libName

		// if nested paths are not specified, just use the unformalized library
		if library == "" {
			library = funcBody.library
		}
		res, err = externalJSEvaluator.Evaluate(library, funcBody.object, opts, args, functions.NewUdfContext(context, funcBody.prefix))

	} else { // the function is an Internal JS function

		// If the function body's text has not already been loaded, load it
		// For internal JS functions, the libName i.e the JS library will be the full path of the function
		path := name.Path()
		library := algebra.PathFromParts(path...)
		_, isLoaded := libStore.Read(library)

		if !isLoaded {
			err1 := LoadFunction(name, body, false)
			if err1 != nil {
				return nil, err1
			}
		}

		res, err = internalJSEvaluator.Evaluate(library, funcName, opts, args, functions.NewUdfContext(context, funcBody.prefix))
	}

	// TODO TENANT
	context.RecordJsCU(time.Millisecond, 10*1024)
	if err.Err != nil {
		return nil, funcBody.execError(err, funcName)
	} else {
		return value.NewValue(res), nil
	}
}

func (this *javascriptBody) execError(err defs.Error, name string) errors.Error {
	if err.IsNestedErr {
		return errors.NewInnerFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
			name, fmt.Errorf("%v", err.Message))
	} else if err.Details != nil {
		return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
			name, err.Details)
	} else {
		return errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", this.library, this.object),
			name, fmt.Errorf("%v %v", err.Err, err.Message))
	}
}

func NewJavascriptBody(library, object, text string) (functions.FunctionBody, errors.Error) {
	return NewJavascriptBodyWithDetails(library, object, "", "", text)
}

func NewJavascriptBodyWithDetails(library, object, prefix, libName, text string) (functions.FunctionBody, errors.Error) {
	if text == "" {
		if !externalJSEnabled {
			return nil, errors.NewFunctionsDisabledError("External javascript")
		}
	} else {
		if !internalJSEnabled {
			return nil, errors.NewFunctionsDisabledError("Internal javascript")
		}
	}
	return &javascriptBody{library: library, object: object, prefix: prefix, libName: libName, text: text}, nil
}

func (this *javascriptBody) SetVarNames(vars []string) errors.Error {
	this.varNames = vars
	return nil
}

func (this *javascriptBody) SetStorage(context functions.Context, path []string) errors.Error {
	// If it is an Internal JS function
	if this.text != "" {
		return nil
	}

	var storageContext string

	if len(path) == 4 {
		storageContext = path[1] + "/" + path[2]
	}
	this.prefix = ""
	this.libName = this.library
	if context.IsTracked() && len(path) == 4 {
		this.prefix = storageContext
	}

	// check if nested library path is used
	firstSlash := strings.IndexByte(this.library, '/')
	switch firstSlash {
	case -1:

		// nothing, all good
		return nil
	case 0:

		// absolute paths are forbidden
	default:

		// relative path, adjust and allow
		if strings.HasPrefix(this.library, "./") && strings.IndexByte(this.library[2:], '/') < 0 {
			this.libName = this.library[2:]
			this.prefix = storageContext
			return nil
		} else if !context.IsTracked() && strings.HasPrefix(this.library, storageContext+"/") && strings.IndexByte(this.library[len(storageContext)+1:], '/') < 0 {

			// for tenant scenarios, no nested path is allowed
			// for scope functions, only the function scope is allowed
			return nil
		}
	}
	return errors.NewFunctionLibraryPathError(this.library)
}

func (this *javascriptBody) Lang() functions.Language {
	return functions.JAVASCRIPT
}

func (this *javascriptBody) Body(object map[string]interface{}) {
	object["#language"] = "javascript"

	if this.varNames != nil {
		vars := make([]value.Value, len(this.varNames))
		for v, _ := range this.varNames {
			vars[v] = value.NewValue(this.varNames[v])
		}
		object["parameters"] = vars
	}

	if this.text != "" { // If is an Internal JS function
		object["text"] = this.text
	} else {
		object["library"] = this.library
		object["object"] = this.object

		if this.prefix != "" {
			object["prefix"] = this.prefix
		}
		if this.libName != "" {
			object["libName"] = this.libName
		}
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

// For internal JS functions load the JS code in the function body text to LibStore
func LoadFunction(name functions.FunctionName, body functions.FunctionBody, unloadAfter bool) errors.Error {
	funcBody, ok := body.(*javascriptBody)

	if !ok || (ok && funcBody.text == "") {
		return nil
	}

	path := name.Path()
	library := algebra.PathFromParts(path...)

	// For internal JS functions, the libName i.e the JS library will be the full path of the function
	err := libStore.Load(library, funcBody.text)

	if err.Err != nil {
		return loadError(err, library)
	}

	if unloadAfter {
		libStore.Unload(library)
	}

	return nil
}

func loadError(err defs.Error, name string) errors.Error {
	if err.Details != nil {
		return errors.NewFunctionLoadingError(name, err.Details)
	} else {
		errorText := fmt.Errorf("%v %v", err.Message, err.Err)
		return errors.NewFunctionLoadingError(name, errorText.Error())
	}
}
