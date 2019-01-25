//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package globalName

import (
	"encoding/json"
	"strconv"

	"github.com/couchbase/cbauth/metakv"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/resolver"
	"github.com/couchbase/query/logging"
)

const _FUNC_PATH = "/query/functions"
const _CHANGE_COUNTER_PATH = "/query/functions_cache/"
const _CHANGE_COUNTER = _CHANGE_COUNTER_PATH + "counter"

var changeCounter int32
var whoAmI string

func init() {
	whoAmI = distributed.RemoteAccess().WhoAmI()

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
	if node != whoAmI {
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
	return err.Error() == "Not found"
}

// we propagate the node name so that if our own change counter gets back to us
// we don't act on it, and append the change counter so that repeated store
// changes by the same node get propagated, and not lumped as one
func fmtChangeCounter() []byte {
	return []byte(distributed.RemoteAccess().MakeKey(whoAmI, strconv.Itoa(int(changeCounter))))
}

// system wide functions (not bucket or scope dependent)
type globalName struct {
	name          string
	namespace     string
	changeCounter int32
}

func NewGlobalFunction(namespace string, name string) (functions.FunctionName, errors.Error) {
	ns, _ := datastore.GetDatastore().NamespaceByName(namespace)
	if ns == nil {
		return nil, errors.NewInvalidFunctionNameError(namespace + ":" + name)
	}
	return &globalName{name, namespace, 0}, nil
}

func (name *globalName) Name() string {
	return name.name
}

func (name *globalName) Key() string {
	return name.namespace + ":" + name.name
}

func (name *globalName) Signature(object map[string]interface{}) {
	object["name"] = name.name
	object["namespace"] = name.namespace
	object["global"] = true
}

func (name *globalName) Load() (functions.FunctionBody, errors.Error) {
	var _unmarshalled struct {
		Identity   json.RawMessage `json:"identity"`
		Definition json.RawMessage `json:"definition"`
	}

	// load the function
	val, _, err := metakv.Get(_FUNC_PATH + name.Key())
	if isNotFoundError(err) {
		return nil, errors.NewDuplicateFunctionError(name.Name())
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
	return resolver.MakeBody(name, _unmarshalled.Definition)
}

func (name *globalName) Save(body functions.FunctionBody) errors.Error {
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

	err = metakv.Add(_FUNC_PATH+name.Key(), bytes)
	if err == metakv.ErrRevMismatch {
		return errors.NewDuplicateFunctionError(name.Name())
	} else {
		return errors.NewMetaKVError(name.Name(), err)
	}
	setChange()
	name.changeCounter = changeCounter
	return nil
}

func (name *globalName) Delete() errors.Error {
	err := metakv.Delete(_FUNC_PATH+name.Key(), nil)
	if isNotFoundError(err) {
		return errors.NewDuplicateFunctionError(name.Name())
	} else if err != nil {
		return errors.NewMetaKVError(name.Name(), err)
	}
	setChange()
	name.changeCounter = changeCounter
	return nil
}

func (name *globalName) CheckStorage() bool {
	return name.changeCounter != changeCounter
}

func (name *globalName) ResetStorage() {
	name.changeCounter = changeCounter
}
