//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
	Body(object map[string]interface{})
	Indexable() value.Tristate
	SwitchContext() value.Tristate
	IsExternal() bool
	Privileges() (*auth.Privileges, errors.Error)
}

type FunctionEntry struct {
	FunctionName
	FunctionBody
	privs          *auth.Privileges
	tag            atomic.AlignedInt64
	LastUse        time.Time
	Uses           int32
	ServiceTime    atomic.AlignedUint64
	MinServiceTime atomic.AlignedUint64
	MaxServiceTime atomic.AlignedUint64
}

type LanguageRunner interface {
	Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (value.Value, errors.Error)
}

type functionCache struct {
	cache *util.GenCache
	tag   atomic.AlignedInt64
}

var Constructor func(elem []string, namespace string, queryContext string) (FunctionName, errors.Error)
var Authorize func(privileges *auth.Privileges, credentials *auth.Credentials) errors.Error

var languages = [_SIZER]LanguageRunner{&missing{}, &empty{}}
var functions = &functionCache{}

// init functions cache
func init() {
	functions.cache = util.NewGenCache(_LIMIT)
}

func FunctionsNewLanguage(lang Language, runner LanguageRunner) {
	if runner != nil && lang != _MISSING {
		languages[lang] = runner
	}
}

// support for language runners

// golang UDFs can use this to execute N1QL
func Run(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, context Context) (value.Value, uint64, error) {
	return context.EvaluateStatement(statement, namedArgs, positionalArgs, false, context.Readonly())
}

// configure functions cache

func FunctionsLimit() int {
	return functions.cache.Limit()
}

func FunctionsSetLimit(limit int) {
	functions.cache.SetLimit(limit)
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

	if f != nil {
		process = func(entry interface{}) {
			ce := entry.(*FunctionEntry)
			f(ce)
		}
	}
	return functions.cache.Delete(key, process)
}

// name resolution
// mock system wide functions (for local testing)
type mockName struct {
	name      string
	namespace string
}

func mockFunction(namespace string, name string) FunctionName {
	return &mockName{name, namespace}
}

func (name *mockName) Name() string {
	return name.name
}

func (name *mockName) Key() string {
	return name.namespace + ":" + name.name
}

func (name *mockName) IsGlobal() bool {
	return true
}

func (name *mockName) QueryContext() string {
	return name.namespace + ":"
}

func (name *mockName) Signature(object map[string]interface{}) {
	object["name"] = name.name
	object["namespace"] = name.namespace
	object["global"] = true
}

func (name *mockName) Load() (FunctionBody, errors.Error) {
	return nil, nil
}

func (name *mockName) Save(body FunctionBody, replace bool) errors.Error {
	return nil
}

func (name *mockName) Delete() errors.Error {
	return nil
}

func (name *mockName) CheckStorage() bool {
	return false
}

func (name *mockName) ResetStorage() {
}

// function primitives
func AddFunction(name FunctionName, body FunctionBody, replace bool) errors.Error {

	// add the function
	err := name.Save(body, replace)
	if err == nil {
		function := &FunctionEntry{FunctionName: name, FunctionBody: body}
		key := name.Key()

		// add it to the cache
		added := true
		functions.cache.Add(function, key, func(ce interface{}) util.Operation {

			// remove any cached missing entry
			_, ok := (ce.(*FunctionEntry)).FunctionBody.(*missing)
			if ok {
				return util.REPLACE
			}

			// this should never be happening, but if somebody pushed it in the cache
			// in spite of us actually saving it, the cache can't be trusted!
			added = false
			return util.IGNORE
		})

		if !added {
			functions.cache.Delete(key, nil)
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
		functions.cache.Delete(key, nil)

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
		err := Authorize(f.privs, context.Credentials())
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
		entry = ce.(*FunctionEntry)
		body = entry.FunctionBody

		// if the storage change counter has moved, we may need to update the cache
		// note that since we reload the body outside of a cache lock (not to lock
		// out the whole cache bucket), there might be some temporary pile up on
		// storage
		if name.CheckStorage() {
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
						e.FunctionBody = body
						e.FunctionName.ResetStorage()
						resetPrivs = true
					}
				})
				if resetPrivs {
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

	if err != nil {
		return nil, err
	}

	// if neither worked, create and cache a missing entry
	if body == nil {

		// if somebody loaded in the interim, we're rescued!
		entry.FunctionBody = &missing{}
		entry = entry.add()
		body = entry.FunctionBody
	}

	// go and do the dirty deed
	err = Authorize(entry.privs, context.Credentials())
	if err != nil {
		return nil, err
	}

	newContext := context
	switchContext := body.SwitchContext()
	readonly := (modifiers & READONLY) != 0
	if switchContext == value.TRUE || (readonly && switchContext == value.NONE) {
		var ok bool

		newContext, ok = context.NewQueryContext(name.QueryContext(), readonly).(Context)
		if !ok {
			return nil, errors.NewInternalFunctionError(fmt.Errorf("Invalid function context received"), name.Name())
		}
	}
	start := time.Now()
	val, err := languages[entry.Lang()].Execute(name, body, modifiers, values, newContext)

	// update stats
	serviceTime := time.Since(start)
	atomic.AddInt32(&entry.Uses, 1)

	// this is strictly not correct, but we'd rather have an approximate time than lock
	entry.LastUse = start
	atomic.AddUint64(&entry.ServiceTime, uint64(serviceTime))
	util.TestAndSetUint64(&entry.MinServiceTime, uint64(serviceTime),
		func(old, new uint64) bool { return old > new }, 0)
	util.TestAndSetUint64(&entry.MaxServiceTime, uint64(serviceTime),
		func(old, new uint64) bool { return old < new }, 0)
	return val, err
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
	privs, err := entry.FunctionBody.Privileges()
	if err != nil {
		return err
	}
	if privs == nil {
		privs = auth.NewPrivileges()
	}
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
	entry.privs = privs
	return nil
}

// dummy runner throwing errors, for initialization purposes
type empty struct {
}

func (this *empty) Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (value.Value, errors.Error) {
	return nil, errors.NewFunctionsNotSupported("")
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

func (this *missing) Execute(name FunctionName, body FunctionBody, modifiers Modifier, values []value.Value, context Context) (value.Value, errors.Error) {
	return nil, errors.NewMissingFunctionError(name.Name())
}
