//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package metaStorage

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/couchbase/cbauth/metakv"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/resolver"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const _FUNC_PATH = "/query/functions/"
const _CHANGE_COUNTER_PATH = "/query/functions_cache/"
const _CHANGE_COUNTER = _CHANGE_COUNTER_PATH + "counter"

var changeCounter int32

func Init() {

	// setup the change counter if not there
	err := metakv.Add(_CHANGE_COUNTER, fmtChangeCounter())

	// if we got some non 200 status from ns_server
	// remote cache invalidation won't work!
	if err != metakv.ErrRevMismatch {
		logging.Infof("Unable to initialize functions cache monitor %v", errors.NewMetaKVChangeCounterError(err))
	}

	// fire callback runner. It won't ever return
	go metakv.RunObserveChildren(_CHANGE_COUNTER_PATH, callback, make(chan struct{}))
}

// change callback
func callback(kve metakv.KVEntry) error {

	// should never happen
	if kve.Path != _CHANGE_COUNTER {
		return nil
	}
	node, _ := distributed.RemoteAccess().SplitKey(string(kve.Value))

	// unclustered nodes can't check against themselves as there may be many of
	// them, and all present themselves with an empty name
	if node == "" || node != distributed.RemoteAccess().WhoAmI() {
		atomic.AddInt32(&changeCounter, 1)
	}
	return nil
}

func setChange() {
	atomic.AddInt32(&changeCounter, 1)
	err := metakv.Set(_CHANGE_COUNTER, fmtChangeCounter(), nil)

	// should not happen, but the change counter has gone awol
	// let's try to reinstate it
	if isNotFoundError(err) {
		err = metakv.Add(_CHANGE_COUNTER, fmtChangeCounter())
	}
	if err != nil {
		logging.Infof("Unable to update functions cache monitor %v", errors.NewMetaKVChangeCounterError(err))
	}
}

func NewChangeCounter() int32 {
	setChange()
	return changeCounter
}

func ChangeCounter() int32 {
	return changeCounter
}

// dodgy, but the not found error is not exported in metakv
func isNotFoundError(err error) bool {
	return err != nil && err.Error() == "Not found"
}

// we propagate the node name so that if our own change counter gets back to us
// we don't act on it, and append the change counter so that repeated store
// changes by the same node get propagated, and not lumped as one
func fmtChangeCounter() []byte {
	return []byte(distributed.RemoteAccess().MakeKey(distributed.RemoteAccess().WhoAmI(), strconv.Itoa(int(changeCounter))))
}

// datastore and function store actions
// TODO this is very inefficient - we'll amend when we write the new storage
func DropScope(namespace, bucket, scope string) {
	scopePath := _FUNC_PATH + namespace + ":" + bucket + "." + scope + "."
	metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		if strings.HasPrefix(kve.Path, scopePath) {

			// technically, we don't need to clear the cache because clearing the
			// storage will invalidate the entry, but for completeness
			functions.FunctionClear(kve.Path[len(scopePath):], nil)
			metakv.Delete(kve.Path, nil)
		}
		return nil
	})
}

func Foreach(b string, f func(path string, v value.Value) error) error {
	if b == "" {
		return metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
			path := kve.Path[len(_FUNC_PATH):]
			if algebra.PartsFromPath(path) == 2 {
				return f(path, value.NewValue(kve.Value))
			}
			return nil
		})
	}
	return metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		path := kve.Path[len(_FUNC_PATH):]
		parts := algebra.ParsePath(path)
		if len(parts) == 4 && (parts[1] == b || b == "*") {
			return f(path, value.NewValue(kve.Value))
		}
		return nil
	})
}

func ForeachBodyEntry(f func(parts []string, entry map[string]interface{}) errors.Error) errors.Error {
	var _unmarshalled struct {
		Identity   json.RawMessage `json:"identity"`
		Definition json.RawMessage `json:"definition"`
	}

	err1 := metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		path := kve.Path[len(_FUNC_PATH):]
		parts := algebra.ParsePath(path)
		if len(parts) == 4 {

			// unmarshal signature and body
			err := json.Unmarshal(kve.Value, &_unmarshalled)
			if err != nil {
				logging.Infof("processing %v error %v unmarshalling entry", parts, err)
				name := parts[0] + ":" + parts[1] + "." + parts[2] + "." + parts[3]
				return errors.NewFunctionEncodingError("decode", name, err)
			}

			entry, err := resolver.MakeBodyEntry(path, _unmarshalled.Definition)
			if err != nil {
				logging.Infof("processing %v error %v constructing function body entry", parts, err)
				return err
			}
			err = f(parts, entry)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err1 != nil {
		return errors.NewMetaKVError("Error during scanning of function definitions", err1)
	}
	return nil
}

func ForeachBody(f func(parts []string, b functions.FunctionBody) errors.Error) errors.Error {
	var _unmarshalled struct {
		Identity   json.RawMessage `json:"identity"`
		Definition json.RawMessage `json:"definition"`
	}

	err1 := metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		path := kve.Path[len(_FUNC_PATH):]
		parts := algebra.ParsePath(path)
		if len(parts) == 4 {

			// unmarshal signature and body
			err := json.Unmarshal(kve.Value, &_unmarshalled)
			if err != nil {
				logging.Infof("processing %v error %v unmarshalling entry", parts, err)
				name := parts[0] + ":" + parts[1] + "." + parts[2] + "." + parts[3]
				return errors.NewFunctionEncodingError("decode", name, err)
			}

			// determine language and create body from definition
			body, err := resolver.MakeBody(path, _unmarshalled.Definition)
			if err != nil {
				logging.Infof("processing %v error %v constructing function body", parts, err)
				return err
			}

			err = f(parts, body)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err1 != nil {
		return errors.NewMetaKVError("Error during scanning of function definitions", err1)
	}
	return nil
}

func Scan(b string, f func(path string) error) error {
	if b == "" {
		return metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
			path := kve.Path[len(_FUNC_PATH):]
			if algebra.PartsFromPath(path) == 2 {
				return f(path)
			}
			return nil
		})
	}
	return metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		path := kve.Path[len(_FUNC_PATH):]
		parts := algebra.ParsePath(path)
		if len(parts) == 4 && parts[1] == b {
			return f(path)
		}
		return nil
	})
}

func Get(path string) (value.Value, error) {
	body, _, err := metakv.Get(_FUNC_PATH + path)
	if err != nil {
		return nil, err
	}
	return value.NewValue(body), nil
}

func Count(b string) (int64, error) {
	var count int64

	if b == "" {
		err := metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
			path := kve.Path[len(_FUNC_PATH):]
			if algebra.PartsFromPath(path) == 2 {
				count++
			}
			return nil
		})
		return count, err
	}
	err := metakv.IterateChildren(_FUNC_PATH, func(kve metakv.KVEntry) error {
		path := kve.Path[len(_FUNC_PATH):]
		parts := algebra.ParsePath(path)
		if len(parts) == 4 && parts[1] == b {
			count++
		}
		return nil
	})
	return count, err
}

type metaEntry struct {
	path          algebra.Path
	changeCounter int32
	useSystem     bool
	systemEntry   functions.FunctionName // redirected entry after migration
}

func NewGlobalFunction(namespace string, name string) (functions.FunctionName, errors.Error) {
	rv := &metaEntry{}
	algebra.SetPathShort(namespace, name, &rv.path)
	ns, err := datastore.GetDatastore().NamespaceById(namespace)
	if ns == nil {
		return nil, errors.NewInvalidFunctionNameError(rv.path.FullName(), err)
	}
	return rv, nil
}

func NewScopeFunction(namespace string, bucket string, scope string, name string) (functions.FunctionName, errors.Error) {
	rv := &metaEntry{}
	algebra.SetPathLong(namespace, bucket, scope, name, &rv.path)
	sc, err := datastore.GetScope(namespace, bucket, scope)
	if sc == nil {
		return nil, errors.NewInvalidFunctionNameError(rv.path.FullName(), err)
	}
	return rv, nil
}

func (name *metaEntry) Path() []string {
	return name.path.Parts()
}

func (name *metaEntry) Name() string {
	return name.path.Keyspace()
}

func (name *metaEntry) Key() string {
	return name.path.FullName()
}

func (name *metaEntry) ProtectedKey() string {
	return name.path.ProtectedString()
}

func (name *metaEntry) IsGlobal() bool {
	return !name.path.IsCollection()
}

func (name *metaEntry) QueryContext() string {
	return name.path.QueryContext()
}

func (name *metaEntry) GetSystemEntry() functions.FunctionName {
	return name.systemEntry
}

func (name *metaEntry) SetSystemEntry(systemEntry functions.FunctionName) {
	name.systemEntry = systemEntry
}

func (name *metaEntry) SetUseSystem() {
	if !name.useSystem && name.systemEntry != nil {
		name.useSystem = true
		name.systemEntry.InheritStorage(name.changeCounter)
	}
}

func (name *metaEntry) Signature(object map[string]interface{}) {
	if name.path.IsCollection() {
		object["namespace"] = name.path.Namespace()
		object["bucket"] = name.path.Bucket()
		object["scope"] = name.path.Scope()
		object["name"] = name.path.Keyspace()
		object["type"] = "scope"
	} else {
		object["namespace"] = name.path.Namespace()
		object["name"] = name.path.Keyspace()
		object["type"] = "global"
	}
}

func (name *metaEntry) Load() (functions.FunctionBody, errors.Error) {
	if name.useSystem {
		return name.systemEntry.Load()
	}

	var _unmarshalled struct {
		Identity   json.RawMessage `json:"identity"`
		Definition json.RawMessage `json:"definition"`
	}

	// load the function
	val, _, err := metakv.Get(_FUNC_PATH + name.Key())

	// Get does not return a not found error - just nil, nil
	if val == nil && err == nil {
		return nil, nil
	} else if err != nil {
		return nil, errors.NewMetaKVError(name.Name(), err)
	}

	// unmarshal signature and body
	err = json.Unmarshal(val, &_unmarshalled)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode", name.Name(), err)
	}

	// determine language and create body from definition
	body, er := resolver.MakeBody(name.Name(), _unmarshalled.Definition)
	if er != nil {
		return nil, er
	}
	name.ResetStorage()
	return body, nil
}

func (name *metaEntry) Save(body functions.FunctionBody, replace bool) errors.Error {
	if name.useSystem {
		return name.systemEntry.Save(body, replace)
	}

	entry := make(map[string]interface{}, 2)
	identity := make(map[string]interface{})
	definition := make(map[string]interface{})
	name.Signature(identity)
	body.Body(definition)
	entry["identity"] = identity
	entry["definition"] = definition
	return name.SaveBodyEntry(entry, replace)
}

func (name *metaEntry) SaveBodyEntry(entry map[string]interface{}, replace bool) errors.Error {
	if name.useSystem {
		return name.systemEntry.SaveBodyEntry(entry, replace)
	}

	bytes, err := json.Marshal(entry)
	if err != nil {
		return errors.NewFunctionEncodingError("encode", name.Name(), err)
	}

	if replace {
		err = metakv.Set(_FUNC_PATH+name.Key(), bytes, nil)
	} else {
		err = metakv.Add(_FUNC_PATH+name.Key(), bytes)
	}
	if err == metakv.ErrRevMismatch {
		return errors.NewDuplicateFunctionError(name.Name())
	} else if err != nil {
		return errors.NewMetaKVError(name.Name(), err)
	}
	setChange()
	name.ResetStorage()
	return nil
}

func (name *metaEntry) Delete() errors.Error {
	if name.useSystem {
		return name.systemEntry.Delete()
	}

	// Delete() does not currently throw an error on missing key, so load first
	val, _, err := metakv.Get(_FUNC_PATH + name.Key())

	// Get does not return a not found error - just nil, nil
	if val == nil && err == nil {
		return errors.NewMissingFunctionError(name.Name())
	} else if err != nil {
		return errors.NewMetaKVError(name.Name(), err)
	}

	err = metakv.Delete(_FUNC_PATH+name.Key(), nil)
	if isNotFoundError(err) {
		return errors.NewMissingFunctionError(name.Name())
	} else if err != nil {
		return errors.NewMetaKVError(name.Name(), err)
	}
	setChange()
	name.ResetStorage()
	return nil
}

func (name *metaEntry) CheckStorage() bool {
	if name.useSystem {
		return name.systemEntry.CheckStorage()
	}

	return name.changeCounter != changeCounter
}

func (name *metaEntry) ResetStorage() {
	if name.useSystem {
		name.systemEntry.ResetStorage()
		return
	}

	name.changeCounter = changeCounter
}

func (name *metaEntry) InheritStorage(changeCnter int32) {
	name.changeCounter = changeCnter
}
