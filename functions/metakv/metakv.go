//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
func callback(path string, val []byte, rev interface{}) error {

	// should never happen
	if path != _CHANGE_COUNTER {
		return nil
	}
	node, _ := distributed.RemoteAccess().SplitKey(string(val))

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
	metakv.IterateChildren(_FUNC_PATH, func(path string, value []byte, rev interface{}) error {
		if strings.HasPrefix(path, scopePath) {

			// technically, we don't need to clear the cache because clearing the
			// storage will invalidate the entry, but for completeness
			functions.FunctionClear(path[len(scopePath):], nil)
			metakv.Delete(path, nil)
		}
		return nil
	})
}

func Foreach(f func(path string, value []byte) error) error {
	return metakv.IterateChildren(_FUNC_PATH, func(p string, value []byte, rev interface{}) error {
		return f(p[len(_FUNC_PATH):], value)
	})
}

func Get(path string) ([]byte, error) {
	body, _, err := metakv.Get(_FUNC_PATH + path)
	return body, err
}

func Count() (int64, error) {
	children, err := metakv.ListAllChildren(_FUNC_PATH)
	if err != nil {
		return -1, err
	} else {
		return int64(len(children)), nil
	}
}

type metaEntry struct {
	path          algebra.Path
	changeCounter int32
}

func NewGlobalFunction(namespace string, name string) (functions.FunctionName, errors.Error) {
	rv := &metaEntry{}
	algebra.SetPathShort(namespace, name, &rv.path)
	ns, err := datastore.GetDatastore().NamespaceByName(namespace)
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

func (name *metaEntry) Name() string {
	return name.path.Keyspace()
}

func (name *metaEntry) Key() string {
	return name.path.FullName()
}

func (name *metaEntry) IsGlobal() bool {
	return !name.path.IsCollection()
}

func (name *metaEntry) QueryContext() string {
	return name.path.QueryContext()
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
	name.changeCounter = changeCounter

	// unmarshal signature and body
	err = json.Unmarshal(val, &_unmarshalled)
	if err != nil {
		return nil, errors.NewFunctionEncodingError("decode", name.Name(), err)
	}

	// determine language and create body from definition
	return resolver.MakeBody(name.Name(), _unmarshalled.Definition)
}

func (name *metaEntry) Save(body functions.FunctionBody, replace bool) errors.Error {
	entry := make(map[string]interface{}, 2)
	identity := make(map[string]interface{})
	definition := make(map[string]interface{})
	name.Signature(identity)
	body.Body(definition)
	entry["identity"] = identity
	entry["definition"] = definition
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
	name.changeCounter = changeCounter
	return nil
}

func (name *metaEntry) Delete() errors.Error {
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
	name.changeCounter = changeCounter
	return nil
}

func (name *metaEntry) CheckStorage() bool {
	return name.changeCounter != changeCounter
}

func (name *metaEntry) ResetStorage() {
	name.changeCounter = changeCounter
}
