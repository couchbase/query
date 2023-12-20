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
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/eventing-ee/evaluator/defs"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server/http/router"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// we won't let a javascript function execute more than 2 minutes
const _MAX_TIMEOUT = 120000
const _DEF_RUNNERS = 32
const _MIN_THREADS_THRESHOLD = 4
const _DEFLATE_THRESHOLD = 48
const _MAX_THREAD_COUNT = 4096
const _MAX_LEVELS = 128

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

type evaluatorDesc struct {
	threads   int32
	available int32
	amending  int32
	name      string
	engine    defs.Engine
	evaluator defs.Evaluator
	libStore  defs.LibStore
}

var external evaluatorDesc
var internal evaluatorDesc
var tenants map[string]*evaluatorDesc
var tenantsLock sync.RWMutex

// - indexing issues: identify functions as deterministic and not running N1QL code

// TODO TENANT cleanup tenant runners

func Init(router router.Router, jsevaluatorPath string) {
	functions.FunctionsNewLanguage(functions.JAVASCRIPT, &javascript{})

	// TODO for serverless the global engine is there to service couchbase provided global libraries
	// we may decide to disable it completely

	// Create the engine for external JS functions
	external.engine = defs.SingleInstance
	external.name = "external jsevaluator"
	external.available = _DEF_RUNNERS

	globalCfg := defs.GlobalConfig{
		GlobalManagePermission: "cluster.n1ql.udf_external!manage",
		ScopeManagePermission:  "cluster.collection[%s].n1ql.udf_external!manage",

		// Restrictions on JavaScript code in the function body
		// should be enabled in all currently supported cluster deployment models
		JsRestrictionsEnabled:   true,
		ProcessIsolationEnabled: true, // tenant.IsServerless(),
	}

	// set the path to the jsevaluator binary
	if jsevaluatorPath != "" {
		globalCfg.EvaluatorExecPath = jsevaluatorPath
	}

	configErr := defs.ConfigureGlobalConfig(globalCfg)
	if (configErr != defs.Error{}) {
		logging.Infof("Global config error: %v", configErr)
		return
	}

	engConfig := defs.StaticEngineConfig{
		WorkerCount:    _DEF_RUNNERS,
		FeatureBitmask: uint32(8),
		IsIpV6:         util.IPv6,
		SysLogCallback: func(level, ts string, msg string, ctx interface{}) {
			logging.Infof("jsevaluator: %s", msg)
		},
	}

	dynConfig := defs.DynamicEngineConfig{
		LogLevel: 4,
	}

	external.threads = int32(_DEF_RUNNERS)

	err := external.engine.Configure(engConfig, dynConfig)
	if err.Err == nil {
		if router != nil {
			handle := external.engine.UIHandler()
			router.MapPrefix(handle.Path(), handle.Handler(), "GET", "POST", "DELETE")
		}
		if err.Err == nil {
			err = external.engine.Start()
		}
	}

	if err.Err != nil {
		logging.Infof("Unable to start javascript evaluator client, err : %v", err.Err)
	} else {
		external.evaluator = external.engine.Fetch()
		if external.evaluator == nil {
			logging.Infof("Unable to retrieve javascript evaluator")
		} else {
			logging.Infof("Started jsevaluator for %v with %v runners", external.name, _DEF_RUNNERS)
		}
	}
	if tenant.IsServerless() {
		tenants = make(map[string]*evaluatorDesc)
		tenant.RegisterResourceManager(manageTenant)
	} else {
		internal.name = "internal javascript"
		internal.engine, internal.libStore, internal.evaluator, _ = newEngine(internal.name, _DEF_RUNNERS)
		internal.threads = int32(_DEF_RUNNERS)
		internal.available = _DEF_RUNNERS
		if internal.libStore == nil {
			internal.evaluator = nil
		}
	}
}

func newEngine(desc string, t int) (defs.Engine, defs.LibStore, defs.Evaluator, defs.Error) {
	var evaluator defs.Evaluator

	engConfig := defs.StaticEngineConfig{
		WorkerCount:    _DEF_RUNNERS,
		FeatureBitmask: uint32(8),
		IsIpV6:         util.IPv6,
		SysLogCallback: func(level, ts string, msg string, ctx interface{}) {
			logging.Infof("jsevaluator for %v: %s", desc, msg)
		},
	}

	dynConfig := defs.DynamicEngineConfig{
		LogLevel: 4,
	}

	engine, err := defs.NewEngine()
	if (err != defs.Error{}) {
		logging.Infof("Unable to create new Engine for %v", desc)
	}

	err = engine.Configure(engConfig, dynConfig)
	if err.Err == nil {
		err = engine.Start()
	}

	if err.Err != nil {
		logging.Infof("Unable to start javascript evaluator client for %v, err : %v", desc, err.Err)
	} else {
		evaluator = engine.Fetch()
		if evaluator == nil {
			logging.Infof("Unable to retrieve javascript evaluator for %v", desc)
		}
	}

	libStore := engine.GetLibStore()

	if libStore == nil {
		logging.Infof("Failed to prime evaluator for %v: libStore is nil", desc)

	} else {
		logging.Infof("Started jsevaluator for %v with %v runners", desc, t)
	}
	return engine, libStore, evaluator, err
}

func manageTenant(bucket string) {
	tenantsLock.Lock()
	desc := tenants[bucket]
	delete(tenants, bucket)
	tenantsLock.Unlock()
	if desc != nil {
		desc.engine.Shutdown()
		logging.Infof("Unloading jsevaluator tenant %v", bucket)
	}
}

func getEvaluator(name functions.FunctionName) (*evaluatorDesc, errors.Error) {
	var tenant string
	var err defs.Error

	path := name.Path()

	// yawser! a global UDF in serverless! Administrator must be up to something!
	if len(path) == 2 {
		tenant = "#internal"
	} else {
		tenant = path[1]
	}
	tenantsLock.RLock()
	desc := tenants[tenant]
	tenantsLock.RUnlock()
	if desc != nil {
		return desc, nil
	}

	// not there
	tenantsLock.Lock()

	// somebody created in the interim?
	desc = tenants[tenant]
	if desc != nil {
		tenantsLock.Unlock()
		return desc, nil
	}
	desc = &evaluatorDesc{}
	desc.engine, desc.libStore, desc.evaluator, err = newEngine(tenant, _DEF_RUNNERS)
	if err.Err != nil {
		tenantsLock.Unlock()
		if err.Details != nil {
			return nil, errors.NewEvaluatorLoadingError(tenant, err.Details)
		} else {
			errorText := fmt.Errorf("%v %v", err.Message, err.Err)
			return nil, errors.NewEvaluatorLoadingError(tenant, errorText.Error())
		}
	}
	desc.threads = _DEF_RUNNERS
	desc.available = desc.threads
	desc.name = tenant
	tenants[tenant] = desc
	tenantsLock.Unlock()
	return desc, nil
}

func (this *javascript) CheckAuthorize(name string, context functions.Context) bool {
	return true
}

// Returns a map of:
// Embedded: slice of strings. The Embedded query strings inside the JS UDF
// Dynamic:  slice of uint. The line numbers of Dynamic N1QL queries inside the JS UDF
func (this *javascript) FunctionStatements(name functions.FunctionName, body functions.FunctionBody,
	context functions.Context) (interface{}, errors.Error) {
	var evaluator *evaluatorDesc

	funcName := name.Name()
	funcBody, ok := body.(*javascriptBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	// Get the evaluator
	if funcBody.text == "" {
		evaluator = &external
	} else if tenant.IsServerless() {
		var err errors.Error
		evaluator, err = getEvaluator(name)
		if evaluator == nil {
			return nil, err
		}
	} else {
		evaluator = &internal
	}
	if evaluator.evaluator == nil {
		return nil, errors.NewFunctionsDisabledError(evaluator.name)
	}

	var library string

	if funcBody.text == "" {
		// External JS UDF
		library = funcBody.libName
		funcName = funcBody.object

		// if nested paths are not specified, just use the unformalized library
		if library == "" {
			library = funcBody.library
		}
	} else {
		// N1QL managed JS UDF
		// If the function body's text has not already been loaded, load it
		library = nameToLibrary(name)
		_, isLoaded, _ := evaluator.libStore.Read(library)

		if !isLoaded {
			err1 := body.Load(name)
			if err1 != nil {
				return nil, err1
			}
		}

	}

	// Note: Since AllStatements() does not invoke a js runner - runner management is not necessary

	queries, err := evaluator.evaluator.AllStatements(library, funcName, functions.NewUdfBasicContext(context, funcBody.prefix))

	if err.Err != nil {
		if err.Details != nil {
			return nil, errors.NewFunctionStatementsError(fmt.Sprintf("(%v:%v)", library, funcName), funcName, err.Details)
		}

		return nil, errors.NewFunctionStatementsError(fmt.Sprintf("(%v:%v)", library, funcName), funcName,
			fmt.Sprintf("%v:%v", err.Err, err.Message))

	}

	rv := make(map[string]interface{}, 2)

	// slice of strings-  embedded query strings inside the UDF
	rv["embedded"] = queries.Embedded

	// slice of uint- the line numbers of Dynamic N1QL queries inside the UDF
	rv["dynamic"] = queries.Dynamic
	return rv, nil
}

func (this *javascript) Execute(name functions.FunctionName, body functions.FunctionBody, modifiers functions.Modifier, values []value.Value, context functions.Context) (value.Value, errors.Error) {
	var args []interface{}
	var evaluator *evaluatorDesc

	funcName := name.Name()
	funcBody, ok := body.(*javascriptBody)

	if !ok {
		return nil, errors.NewInternalFunctionError(goerrors.New("Wrong language being executed!"), funcName)
	}

	if funcBody.text == "" {
		evaluator = &external
	} else if tenant.IsServerless() {
		var err errors.Error
		evaluator, err = getEvaluator(name)
		if evaluator == nil {
			return nil, err
		}
	} else {
		evaluator = &internal
	}
	if evaluator.evaluator == nil {
		return nil, errors.NewFunctionsDisabledError(evaluator.name)
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
	opts := defs.Options{
		SideEffects: (modifiers & functions.READONLY) == 0,
		Timeout:     uint32(timeout),
	}
	available := atomic.AddInt32(&(*evaluator).available, -1)
	defer atomic.AddInt32(&(*evaluator).available, 1)
	levels := context.IncRecursionCount(1)
	defer context.IncRecursionCount(-1)
	if levels > _MAX_LEVELS {
		return nil, errors.NewFunctionExecutionNestedError(levels, funcName)
	}

	// inflate the pools if required
	// beyond the maximum number of runners the function will have to wait for a runner to be free
	// we keep track of the remaining runners, rather than the active to be sure that only one request
	// at a time inflates the runner pool
	var errorText interface{}
	if available == _MIN_THREADS_THRESHOLD && evaluator.threads < _MAX_THREAD_COUNT {
		if atomic.AddInt32(&(*evaluator).amending, 1) == 1 {

			newThreads, inflateErr := evaluator.engine.InflatePoolBy(_DEF_RUNNERS)
			if newThreads > 0 {
				totThreads := atomic.AddInt32(&(*evaluator).threads, int32(newThreads))
				atomic.AddInt32(&(*evaluator).available, int32(newThreads))
				logging.Infof("Adding %v runners to evaluator %v: actual increment %v to %v", _DEF_RUNNERS, evaluator.name, newThreads, totThreads)
			} else {

				switch {
				case inflateErr.Details != nil:
					errorText = inflateErr.Details
				case inflateErr.Err != nil:
					errorText = fmt.Sprintf("%v %v", inflateErr.Message, inflateErr.Err)
				default:
					errorText = "could not allocate threads, but no error received"
				}
				logging.Infof("Adding %v runners to evaluator %v: error %v", _DEF_RUNNERS, evaluator.name, errorText)
			}
		}
		atomic.AddInt32(&(*evaluator).amending, -1)
	}

	// avoid possible self stalls if the requested has a majority of the executing functions
	if errorText != nil && levels > int((evaluator.threads-available)/2) {
		return nil, errors.NewEvaluatorInflatingError(evaluator.name, fmt.Errorf("%v", errorText))
	}

	var res interface{}
	var err defs.Error
	var library string

	if funcBody.text == "" {
		library = funcBody.libName
		funcName = funcBody.object

		// if nested paths are not specified, just use the unformalized library
		if library == "" {
			library = funcBody.library
		}
	} else {

		// If the function body's text has not already been loaded, load it
		library = nameToLibrary(name)
		_, isLoaded, _ := evaluator.libStore.Read(library)

		if !isLoaded {
			err1 := body.Load(name)
			if err1 != nil {
				return nil, err1
			}
		}

	}

	context.Park(nil, true)
	res, err = evaluator.evaluator.Evaluate(library, funcName, opts, args, functions.NewUdfContext(context, funcBody.prefix, name.Key()))
	context.Resume(true)

	// deflate the pool if required
	if evaluator.threads > _DEF_RUNNERS && evaluator.available >= _DEFLATE_THRESHOLD {
		if atomic.AddInt32(&(*evaluator).amending, 1) == 1 {
			newThreads, _ := evaluator.engine.DeflatePoolBy(_DEF_RUNNERS)
			totThreads := atomic.AddInt32(&(*evaluator).threads, -int32(newThreads))
			atomic.AddInt32(&(*evaluator).available, -int32(newThreads))
			logging.Infof("Dropping %v runners from evaluator %v: actual decrement %v to %v", _DEF_RUNNERS, evaluator.name, newThreads, totThreads)
		}
		atomic.AddInt32(&(*evaluator).amending, -1)
	}

	// TODO TENANT
	context.RecordJsCU(time.Millisecond, 10*1024)
	switch {
	case err.Err == nil:
	case err.IsNestedErr:
		return nil, errors.NewInnerFunctionExecutionError(fmt.Sprintf("(%v:%v)", library, funcName),
			funcName, fmt.Errorf("%v", err.Message))
	case err.Details != nil:
		return nil, errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", library, funcName),
			funcName, err.Details)
	default:
		return nil, errors.NewFunctionExecutionError(fmt.Sprintf("(%v:%v)", library, funcName),
			funcName, fmt.Errorf("%v %v", err.Err, err.Message))
	}
	return value.NewValue(res), nil
}

func NewJavascriptBody(library, object, text string) (functions.FunctionBody, errors.Error) {
	return NewJavascriptBodyWithDetails(library, object, "", "", text)
}

func NewJavascriptBodyWithDetails(library, object, prefix, libName, text string) (functions.FunctionBody, errors.Error) {
	var evalName string
	enabled := false

	if text == "" {
		enabled = external.evaluator != nil
		evalName = external.name
	} else if tenant.IsServerless() {
		enabled = tenants != nil
		evalName = "tenant evaluator"
	} else {
		enabled = internal.evaluator != nil
		evalName = internal.name
	}
	if !enabled {
		return nil, errors.NewFunctionsDisabledError(evalName)
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

func (this *javascriptBody) Test(name functions.FunctionName) errors.Error {

	// avoid overwriting any existing entry
	id := "test_" + nameToLibrary(name)
	err := this.load(name, id)
	this.unload(name, id)
	return err
}

func (this *javascriptBody) Load(name functions.FunctionName) errors.Error {
	return this.load(name, nameToLibrary(name))
}

func (this *javascriptBody) Unload(name functions.FunctionName) {
	this.unload(name, nameToLibrary(name))
}

func (this *javascriptBody) load(name functions.FunctionName, id string) errors.Error {
	var evaluator *evaluatorDesc

	if this.text == "" {
		return nil
	}

	if tenant.IsServerless() {
		var err errors.Error

		evaluator, err = getEvaluator(name)
		if err != nil {
			return err
		}
	} else {
		evaluator = &internal
	}

	err := evaluator.libStore.Load(id, this.text)

	switch {
	case err.Err == nil:
		return nil
	case err.Details != nil:
		return errors.NewFunctionLoadingError(name.Name(), err.Details)
	default:
		errorText := fmt.Errorf("%v %v", err.Message, err.Err)
		return errors.NewFunctionLoadingError(name.Name(), errorText.Error())
	}
}

func (this *javascriptBody) unload(name functions.FunctionName, id string) {
	var evaluator *evaluatorDesc

	if this.text == "" {
		return
	}
	if tenant.IsServerless() {
		var err errors.Error

		evaluator, err = getEvaluator(name)
		if err != nil {
			return
		}
	} else {
		evaluator = &internal
	}
	evaluator.libStore.Unload(id)
}

func nameToLibrary(name functions.FunctionName) string {
	return algebra.PathFromParts(name.Path()...)
}
