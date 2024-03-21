//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package functions

import (
	"fmt"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Language int

const (
	_MISSING Language = iota
	INLINE
	GOLANG
	JAVASCRIPT
	_SIZER
)

type Modifier int

const (
	NONE Modifier = 1 << iota
	READONLY
	INVARIANT
)

const _LIMIT = 16384

type FunctionName interface {
	Path() []string
	Name() string
	Key() string
	IsGlobal() bool
	QueryContext() string
	Signature(object map[string]interface{})
	Load() (FunctionBody, errors.Error)
	Save(body FunctionBody, replace bool) errors.Error
	Delete() errors.Error
	CheckStorage() bool
	ResetStorage()
}

type FunctionBody interface {
	Lang() Language
	SetVarNames(vars []string) errors.Error
	SetStorage(Context, []string) errors.Error
	Body(object map[string]interface{})
	Indexable() value.Tristate
	SwitchContext() value.Tristate
	IsExternal() bool
	Privileges() (*auth.Privileges, errors.Error)
	Test(name FunctionName) errors.Error
	Load(FunctionName) errors.Error
	Unload(FunctionName)
}

type FunctionEntry struct {
	FunctionName
	FunctionBody
	privs          *auth.Privileges
	tag            atomic.AlignedInt64
	LastUse        time.Time
	Uses           atomic.AlignedInt64
	ServiceTime    atomic.AlignedUint64
	MinServiceTime atomic.AlignedUint64
	MaxServiceTime atomic.AlignedUint64
}

type LanguageRunner interface {
	CheckAuthorize(name string, context Context) bool
	Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (
		value.Value, errors.Error)
	FunctionStatements(name FunctionName, body FunctionBody, context Context) (interface{}, errors.Error)
}

type functionCache struct {
	cache *util.GenCache
	tag   atomic.AlignedInt64
}

var Authorize func(privileges *auth.Privileges, credentials *auth.Credentials) errors.Error

var languages = [_SIZER]LanguageRunner{&missing{}, &empty{}}
var functions = &functionCache{}

// init functions cache
func FunctionsInit(limit int) {
	functions.cache = util.NewGenCache(_LIMIT)
	functions.cache.SetLimit(limit)
}

func FunctionsSetLimit(limit int) {
	functions.cache.SetLimit(limit)
}

func FunctionsLimit() int {
	return functions.cache.Limit()
}

func FunctionsNewLanguage(lang Language, runner LanguageRunner) {
	if runner != nil && lang != _MISSING {
		languages[lang] = runner
	}
}

// utilities for functions and system keyspaces
func CountFunctions() int {
	return functions.cache.Size()
}

func NameFunctions() []string {
	return functions.cache.Names()
}

func FunctionsForeach(nonBlocking func(string, *FunctionEntry) bool,
	blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*FunctionEntry))
	}
	functions.cache.ForEach(dummyF, blocking)
}

func FunctionDo(key string, f func(*FunctionEntry)) {
	var process func(interface{}) = nil

	if f != nil {
		process = func(entry interface{}) {
			ce := entry.(*FunctionEntry)
			f(ce)
		}
	}
	_ = functions.cache.Get(key, process)
}

func FunctionClear(key string, f func(*FunctionEntry)) bool {
	var process func(interface{}) = nil

	process = func(entry interface{}) {
		ce := entry.(*FunctionEntry)
		if f != nil {
			f(ce)
		}
		ce.Unload(ce.FunctionName)
	}
	return functions.cache.Delete(key, process)
}

func ClearScopeEntries(namespace, bucket, scope string) {
	var del bool
	var id string

	nonBlocking := func(k string, e interface{}) bool {
		ce := e.(*FunctionEntry)
		path := ce.Path()
		del = len(path) == 4 && namespace == path[0] && bucket == path[1] && scope == path[2]
		ce.Unload(ce.FunctionName)
		id = k
		return true
	}
	blocking := func() bool {
		if del {
			functions.cache.Delete(id, nil)
		}
		return true
	}
	functions.cache.ForEach(nonBlocking, blocking)
}

// name resolution
// mock system wide functions (for local testing)
type MockName struct {
	name      string
	namespace string
}

func MockFunction(namespace string, name string) FunctionName {
	return &MockName{name, namespace}
}

func (name *MockName) Path() []string {
	return []string{name.namespace, name.name}
}

func (name *MockName) Name() string {
	return name.name
}

func (name *MockName) Key() string {
	return name.namespace + ":" + name.name
}

func (name *MockName) IsGlobal() bool {
	return true
}

func (name *MockName) QueryContext() string {
	return name.namespace + ":"
}

func (name *MockName) Signature(object map[string]interface{}) {
	object["name"] = name.name
	object["namespace"] = name.namespace
	object["global"] = true
}

func (name *MockName) Load() (FunctionBody, errors.Error) {
	return nil, nil
}

func (name *MockName) Save(body FunctionBody, replace bool) errors.Error {
	return nil
}

func (name *MockName) Delete() errors.Error {
	return nil
}

func (name *MockName) CheckStorage() bool {
	return false
}

func (name *MockName) ResetStorage() {
}

// function primitives
func AddFunction(name FunctionName, body FunctionBody, replace bool) errors.Error {

	// add the function
	err := name.Save(body, replace)
	if err == nil {
		function := &FunctionEntry{FunctionName: name, FunctionBody: body}
		err = function.loadPrivileges()
		if err != nil {
			return err
		}
		key := name.Key()

		function.tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))

		// add it to the cache
		added := true
		functions.cache.Add(function, key, func(ce interface{}) util.Operation {
			// remove any cached missing entry
			_, ok := (ce.(*FunctionEntry)).FunctionBody.(*missing)
			if ok {
				return util.REPLACE
			} else if replace {
				e := ce.(*FunctionEntry)
				if e.tag < function.tag || (function.tag < 0 && e.tag > 0) {

					// empty any existing N1QL managed javascript function cache entry
					e.FunctionBody.Unload(e.FunctionName)

					return util.REPLACE
				}
			}

			// this should never be happening, but if somebody pushed it in the cache
			// in spite of us actually saving it, the cache can't be trusted!
			added = false
			return util.IGNORE
		})

		if !added {
			functions.cache.Delete(key, func(ce interface{}) {
				ce.(*FunctionEntry).Unload(ce.(*FunctionEntry).FunctionName)
			})
			logging.Debugf("Conflict in saving function to cache, key <ud>%v</ud>", key)
		}

		// remove any missing entry remotely
		distributed.RemoteAccess().DoRemoteOps([]string{}, "functions_cache", "DELETE", key, "",
			func(warn errors.Error) {
				if warn != nil {
					logging.Infof("failed to remote delete function <ud>%v</ud>: %v", name.Name(), warn)
				}
			}, distributed.NO_CREDS, "")
	}
	return err
}

func DeleteFunction(name FunctionName, context Context) errors.Error {
	f, err := checkDelete(name, context)
	if err != nil {
		return err
	} else if f == nil {
		return errors.NewMissingFunctionError(name.Name())
	}

	// do the delete
	err = name.Delete()
	if err == nil {
		key := name.Key()

		// if successful clear the cache locally
		functions.cache.Delete(key, func(ce interface{}) {
			ce.(*FunctionEntry).Unload(ce.(*FunctionEntry).FunctionName)
		})

		// and remotely
		distributed.RemoteAccess().DoRemoteOps([]string{}, "functions_cache", "DELETE", key, "",
			func(warn errors.Error) {
				if warn != nil {
					logging.Infof("failed to remote delete function <ud>%v</ud>: %v", name.Name(), warn)
				}
			}, distributed.NO_CREDS, "")
	}
	return err
}

func GetPrivilege(name FunctionName, body FunctionBody) auth.Privilege {
	var priv auth.Privilege

	if name.IsGlobal() {
		if body.IsExternal() {
			priv = auth.PRIV_QUERY_MANAGE_FUNCTIONS_EXTERNAL
		} else {
			priv = auth.PRIV_QUERY_MANAGE_FUNCTIONS
		}
	} else {
		if body.IsExternal() {
			priv = auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS_EXTERNAL
		} else {
			priv = auth.PRIV_QUERY_MANAGE_SCOPE_FUNCTIONS
		}
	}
	return priv
}

func preLoad(name FunctionName) *FunctionEntry {
	var err errors.Error

	//is the entry in the cache?
	key := name.Key()
	ce := functions.cache.Get(key, nil)
	if ce != nil {
		entry := ce.(*FunctionEntry)
		if entry.Lang() != _MISSING {
			return entry
		} else {
			return nil
		}
	}

	// nope, try to load it
	entry := &FunctionEntry{FunctionName: name}
	entry.FunctionBody, err = name.Load()

	// if all good, cache
	if entry.FunctionBody != nil && err == nil {
		if entry.loadPrivileges() == nil {
			entry.tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))
			entry = entry.add()
			return entry
		}
	}
	return nil
}

func checkDelete(name FunctionName, context Context) (*FunctionEntry, errors.Error) {
	f := preLoad(name)
	if f != nil {
		priv := GetPrivilege(name, f.FunctionBody)
		privs := auth.NewPrivileges()
		privs.Add(f.Key(), priv, auth.PRIV_PROPS_NONE)
		err := Authorize(privs, context.Credentials())
		if err != nil {
			return nil, err
		}
	}
	return f, nil
}

func PreLoad(name FunctionName) bool {
	f := preLoad(name)
	return (f != nil)
}

func CheckDelete(name FunctionName, context Context) errors.Error {
	_, e := checkDelete(name, context)
	return e
}

func Indexable(name FunctionName) value.Tristate {
	f := preLoad(name)
	if f == nil {
		return value.FALSE
	}
	return f.Indexable()
}

func ExecuteFunction(name FunctionName, modifiers Modifier, values []value.Value, context Context) (value.Value, errors.Error) {

	// Get the function's entry
	body, entry, err := getBodyAndEntry(name)

	if err != nil {
		return nil, err
	}

	lang := entry.Lang()
	// go and do the dirty deed
	if languages[lang].CheckAuthorize(name.Key(), context) {
		err = Authorize(entry.privs, context.Credentials())
		if err != nil {
			return nil, err
		}
	}

	newContext := context
	switchContext := body.SwitchContext()
	readonly := (modifiers & READONLY) != 0

	if switchContext == value.TRUE || (switchContext == value.NONE && (readonly != context.Readonly() ||
		name.QueryContext() != context.QueryContext())) || context.PreserveProjectionOrder() {
		var ok bool
		newContext, ok = context.NewQueryContext(name.QueryContext(), readonly).(Context)
		if !ok {
			return nil, errors.NewInternalFunctionError(fmt.Errorf("Invalid function context received"), name.Name())
		}
	}
	start := util.Now()
	val, err := languages[lang].Execute(name, body, modifiers, values, newContext)

	// update stats
	serviceTime := util.Now().Sub(start)
	atomic.AddInt64(&entry.Uses, 1)

	// this is strictly not correct, but we'd rather have an approximate time than lock
	entry.LastUse = start.ToTime()
	atomic.AddUint64(&entry.ServiceTime, uint64(serviceTime))
	util.TestAndSetUint64(&entry.MinServiceTime, uint64(serviceTime),
		func(old, new uint64) bool { return old > new }, 0)
	util.TestAndSetUint64(&entry.MaxServiceTime, uint64(serviceTime),
		func(old, new uint64) bool { return old < new }, 0)

	// propagate transaction context if necessary
	if context != newContext && context.GetTxContext() == nil {
		newTxContext := newContext.GetTxContext()
		if newTxContext != nil {
			context.SetTxContext(newTxContext)
		}
	}

	return val, err
}

// Returns all N1QL query statements inside a function
func FunctionStatements(name FunctionName, creds *auth.Credentials, context Context) (Language, interface{}, errors.Error) {

	// Get the function's entry and body
	body, entry, err := getBodyAndEntry(name)

	if err != nil {
		return _MISSING, nil, err
	}

	lang := entry.Lang()

	// Verify privileges for the entry
	// Authorizes privileges required to execute the function
	// Note: for Inline functions, this performs Authorization for the queries inside the function as well
	err = Authorize(entry.privs, creds)
	if err != nil {
		return lang, nil, err
	}

	rv, err := languages[lang].FunctionStatements(name, body, context)

	return lang, rv, err
}

// Returns function entry and function body
// Returns function body as well - so as to allow cache entry changes to the function body pointer even after.
func getBodyAndEntry(name FunctionName) (FunctionBody, *FunctionEntry, errors.Error) {
	var err errors.Error
	var entry *FunctionEntry

	// we copy the body pointer to allow cache entry changes after we load
	var body FunctionBody

	// get the body from the cache
	key := name.Key()
	ce := functions.cache.Get(key, nil)

	// if not cached, get it from storage
	if ce == nil {
		entry = &FunctionEntry{FunctionName: name}
		entry.FunctionBody, err = name.Load()

		// if all good, cache
		if entry.FunctionBody != nil && err == nil {
			err = entry.loadPrivileges()
			if err == nil {
				entry.tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))
				entry = entry.add()
				body = entry.FunctionBody
			}
		}
	} else {
		var added bool
		entry = ce.(*FunctionEntry)
		body = entry.FunctionBody

		// if the storage change counter has moved, we may need to update the cache
		// note that since we reload the body outside of a cache lock (not to lock
		// out the whole cache bucket), there might be some temporary pile up on
		// storage
		if entry.FunctionName.CheckStorage() {
			var tag atomic.AlignedInt64

			// reserve a change counter and load new body
			tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))
			body, err = name.Load()

			if body != nil {
				resetPrivs := false

				// now that we have a body, we have to amend the cache,
				// but only if noone has done it in the interim
				ce = functions.cache.Get(key, func(ce interface{}) {
					e := ce.(*FunctionEntry)

					// our body is newer
					if e.tag < tag || (tag < 0 && e.tag > 0) {
						e.tag = tag

						// unload N1QL managed javascript body, as it might be stale
						body.Unload(e.FunctionName)
						e.FunctionBody = body
						e.FunctionName.ResetStorage()
						resetPrivs = true
					}
				})
				if ce == nil {
					// Some one deleted but we have new body from disk create entry
					entry = &FunctionEntry{FunctionName: name, FunctionBody: body}
					err = entry.loadPrivileges()
					if err == nil {
						entry.tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))
						entry = entry.add()
					}
					added = true
				} else if resetPrivs {
					e := ce.(*FunctionEntry)
					err = e.loadPrivileges()

					// if we couldn't work out the privileges, this entry is unusable
					if err != nil {
						ce = nil
						functions.cache.Delete(key, nil)
					}
				}
			} else {
				ce = nil
			}

			if !added {
				if ce == nil {
					entry = &FunctionEntry{FunctionName: name}
					entry.tag = atomic.AlignedInt64(atomic.AddInt64(&functions.tag, 1))
					body = nil
				} else {
					entry = ce.(*FunctionEntry)
					body = entry.FunctionBody
				}
			}
		}
	}

	if err != nil {
		return body, nil, err
	}

	// if neither worked, create and cache a missing entry
	if body == nil {

		// if somebody loaded in the interim, we're rescued!
		entry.FunctionBody = &missing{}
		entry = entry.add()
		body = entry.FunctionBody
	}

	return body, entry, err
}

// execution cache work horse
func (entry *FunctionEntry) add() *FunctionEntry {
	functions.cache.Add(entry, entry.Key(), func(ce interface{}) util.Operation {
		oldEntry := ce.(*FunctionEntry)

		// we win
		if oldEntry.tag < entry.tag || (entry.tag < 0 && oldEntry.tag > 0) {
			return util.REPLACE
		}

		// they win
		// note that if their body is *missing, the function was dropped after we loaded!
		entry = oldEntry
		return util.IGNORE
	})
	return entry
}

func (entry *FunctionEntry) loadPrivileges() errors.Error {
	bodyPrivs, err := entry.FunctionBody.Privileges()
	if err != nil {
		return err
	}

	privs := auth.NewPrivileges()

	// add the privilege required to execute the function first so that its Authorization check occurs first too
	if entry.IsGlobal() {
		if entry.IsExternal() {
			privs.Add("", auth.PRIV_QUERY_EXECUTE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
		} else {
			privs.Add("", auth.PRIV_QUERY_EXECUTE_FUNCTIONS, auth.PRIV_PROPS_NONE)
		}
	} else {
		if entry.IsExternal() {
			privs.Add(entry.Key(), auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS_EXTERNAL, auth.PRIV_PROPS_NONE)
		} else {
			privs.Add(entry.Key(), auth.PRIV_QUERY_EXECUTE_SCOPE_FUNCTIONS, auth.PRIV_PROPS_NONE)
		}
	}

	// then add the body privileges
	if bodyPrivs != nil {
		privs.AddAll(bodyPrivs)
	}

	entry.privs = privs
	return nil
}

// dummy runner throwing errors, for initialization purposes
type empty struct {
}

func (this *empty) Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (
	value.Value, errors.Error) {

	return nil, errors.NewFunctionsNotSupported("")
}

func (this *empty) FunctionStatements(name FunctionName, body FunctionBody, context Context) (interface{}, errors.Error) {
	return nil, errors.NewFunctionUnsupportedActionError("", "EXPLAIN FUNCTION")
}

func (this *empty) CheckAuthorize(name string, context Context) bool {
	return true
}

// dummy language throwing errors, for caching missing entries
type missing struct {
}

func (this *missing) Lang() Language {
	return _MISSING
}

func (this *missing) Body(object map[string]interface{}) {
	object["undefined_function"] = true
}

func (this *missing) Indexable() value.Tristate {
	return value.FALSE
}

func (this *missing) SwitchContext() value.Tristate {
	return value.FALSE
}

func (this *missing) IsExternal() bool {
	return false
}

func (this *missing) Privileges() (*auth.Privileges, errors.Error) {
	return nil, nil
}

func (this *missing) SetVarNames(vars []string) errors.Error {
	return nil
}

func (this *missing) SetStorage(context Context, path []string) errors.Error {
	return nil
}

func (this *missing) Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (
	value.Value, errors.Error) {

	return nil, errors.NewMissingFunctionError(name.Name())
}

func (this *missing) Test(name FunctionName) errors.Error {
	return nil
}

func (this *missing) Load(name FunctionName) errors.Error {
	return nil
}

func (this *missing) Unload(name FunctionName) {
}

func (this *missing) FunctionStatements(name FunctionName, body FunctionBody, context Context) (interface{}, errors.Error) {
	return nil, errors.NewMissingFunctionError(name.Name())
}

func (this *missing) CheckAuthorize(name string, context Context) bool {
	return true
}
