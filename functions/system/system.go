//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package systemStorage

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	metaStorage "github.com/couchbase/query/functions/metakv"
	"github.com/couchbase/query/functions/resolver"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const _PREFIX = "udf::"

// currently nothing to do
func Init() {
}

// datastore and function store actions
func DropScope(namespace string, bucket string, scope string, uid string) {
	logging.Debugf("%v:%v.%v (%v)", namespace, bucket, scope, uid)
	datastore.ScanSystemCollection(bucket, _PREFIX+uid+"::"+scope, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			s, fn := key2parts(key)
			if s == "" {
				return nil
			}
			path := algebra.PathFromParts(namespace, bucket, scope, fn)
			logging.Debugf("%v", path)
			delete2(systemCollection, key, path)
			functions.FunctionClear(path, nil)
			return nil
		}, nil)
}

func Foreach(b string, f func(path string, v value.Value) error) error {
	return datastore.ScanSystemCollection(b, _PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			s, fn := key2parts(key)
			if s == "" {
				return nil
			}
			name, err := NewScopeFunction(systemCollection.NamespaceId(), b, s, fn)
			if err != nil {
				return err
			}
			val, err := get2(name, systemCollection)
			if err != nil {
				return err
			}
			err1 := f(algebra.PathFromParts(systemCollection.NamespaceId(), b, s, fn), val)
			if err1 != nil {
				return errors.NewSystemCollectionError("Error calling processing function", err1)
			}
			return nil
		}, nil)
}

func Scan(b string, f func(path string) error) error {
	return datastore.ScanSystemCollection(b, _PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			s, fn := key2parts(key)
			if s == "" {
				return nil
			}
			err := f(algebra.PathFromParts(systemCollection.NamespaceId(), b, s, fn))
			if err != nil {
				return errors.NewSystemCollectionError("Error calling processing function", err)
			}
			return nil
		}, nil)
}

func Get(path string) (value.Value, error) {
	parts := algebra.ParsePath(path)
	if len(parts) != 4 {
		return nil, errors.NewInvalidFunctionNameError(path, fmt.Errorf("name has %v parts", len(parts)))
	}
	systemCollection, err := getSystemCollection(parts[1])
	if err != nil {
		return nil, err
	}
	name, err := NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
	if err != nil {
		return nil, err
	}
	return get2(name, systemCollection)
}

func get2(name functions.FunctionName, systemCollection datastore.Keyspace) (value.Value, errors.Error) {
	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)

	scope, err := systemCollection.Scope().Bucket().ScopeByName(name.Path()[2])
	if err != nil {
		return nil, err
	}
	key := parts2key(scope.Uid(), name.Path()...)

	errs := systemCollection.Fetch([]string{key}, fetchMap, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	for _, err := range errs {
		if err.IsFatal() {
			return nil, errors.NewSystemCollectionError("Error fetching documents from system collection", err)
		}
	}

	// for historical reasons Get() returns nil, nil on not found
	definition := fetchMap[key]
	if definition == nil {
		return nil, nil
	}
	val := make(map[string]interface{}, 2)
	identity := make(map[string]interface{})
	name.Signature(identity)
	val["identity"] = identity
	val["definition"] = definition
	return value.NewValue(val), nil
}

func Count(b string) (int64, error) {
	var count int64

	err := datastore.ScanSystemCollection(b, _PREFIX, nil,
		func(key string, systemCollection datastore.Keyspace) errors.Error {
			count++
			return nil
		}, nil)
	return count, err
}

func getSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	store := datastore.GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}

	return store.GetSystemCollection(bucketName)
}

func key2parts(key string) (string, string) {
	if key[:len(_PREFIX)] != _PREFIX {
		return "", ""
	}
	key = key[len(_PREFIX)+10:] // strip prefix and scope UID
	dot := strings.IndexByte(key, '.')
	if dot <= 1 {
		return "", ""
	}
	return key[:dot], key[dot+1:]
}

func parts2key(uid string, parts ...string) string {
	return _PREFIX + uid + "::" + parts[2] + "." + parts[3]
}

type systemEntry struct {
	path          algebra.Path
	changeCounter int32
}

func NewScopeFunction(namespace string, bucket string, scope string, name string) (functions.FunctionName, errors.Error) {
	rv := &systemEntry{}
	algebra.SetPathLong(namespace, bucket, scope, name, &rv.path)
	sc, err := datastore.GetScope(namespace, bucket, scope)
	if sc == nil {
		return nil, errors.NewInvalidFunctionNameError(rv.path.FullName(), err)
	}
	return rv, nil
}

func (name *systemEntry) Path() []string {
	return name.path.Parts()
}

func (name *systemEntry) Name() string {
	return name.path.Keyspace()
}

func (name *systemEntry) Key() string {
	return name.path.FullName()
}

func (name *systemEntry) ProtectedKey() string {
	return name.path.ProtectedString()
}

func (name *systemEntry) IsGlobal() bool {
	return false
}

func (name *systemEntry) QueryContext() string {
	return name.path.QueryContext()
}

func (name *systemEntry) GetSystemEntry() functions.FunctionName {
	return nil
}

func (name *systemEntry) SetSystemEntry(systemEntry functions.FunctionName) {
}

func (name *systemEntry) SetUseSystem() {
}

func (name *systemEntry) Signature(object map[string]interface{}) {
	object["namespace"] = name.path.Namespace()
	object["bucket"] = name.path.Bucket()
	object["scope"] = name.path.Scope()
	object["name"] = name.path.Keyspace()
	object["type"] = "scope"
}

func (name *systemEntry) Load() (functions.FunctionBody, errors.Error) {
	systemCollection, err := getSystemCollection(name.path.Parts()[1])
	if err != nil {
		return nil, err
	}
	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)
	scope, err := systemCollection.Scope().Bucket().ScopeByName(name.Path()[2])
	if err != nil {
		return nil, err
	}
	key := parts2key(scope.Uid(), name.Path()...)
	errs := systemCollection.Fetch([]string{key}, fetchMap, datastore.NULL_QUERY_CONTEXT, nil, nil, false)
	for _, err := range errs {
		if err.IsFatal() {
			return nil, errors.NewSystemCollectionError("Error fetching documents from system collection", err)
		}
	}

	val := fetchMap[key]
	if val == nil {
		return nil, nil
	}

	// unmarshal body
	body := value.ToJSON(val.GetValue())
	if body == nil {
		return nil, errors.NewFunctionEncodingError("decode", name.Name(), fmt.Errorf("function entry is not a document"))
	}

	// determine language and create body from definition
	b, e := resolver.MakeBody(name.Name(), body)
	if e != nil {
		return nil, e
	}
	name.changeCounter = metaStorage.ChangeCounter()
	val.Recycle()
	return b, nil
}

func (name *systemEntry) Save(body functions.FunctionBody, replace bool) errors.Error {
	entry := make(map[string]interface{})
	body.Body(entry)
	return name.SaveBodyEntry(entry, replace)
}

func (name *systemEntry) SaveBodyEntry(entry map[string]interface{}, replace bool) errors.Error {
	parts := name.path.Parts()
	if len(parts) != 4 {
		return errors.NewInvalidFunctionNameError(name.Name(), fmt.Errorf("name has %v parts", len(parts)))
	}
	systemCollection, err := getSystemCollection(parts[1])
	if err != nil {
		return err
	}

	scope, err := systemCollection.Scope().Bucket().ScopeByName(name.Path()[2])
	if err != nil {
		return errors.NewSystemCollectionError("Error getting scope UID", err)
	}

	queryContext := datastore.GetDurableQueryContextFor(systemCollection)

	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = parts2key(scope.Uid(), parts...)
	dpairs[0].Value = value.NewValue(entry)
	var errs errors.Errors

	if replace {
		_, _, errs = systemCollection.Upsert(dpairs, queryContext, false)
	} else {
		_, _, errs = systemCollection.Insert(dpairs, queryContext, false)
	}

	if len(errs) > 0 {
		if errs[0].HasCause(errors.E_DUPLICATE_KEY) {
			return errors.NewDuplicateFunctionError(name.Name())
		} else {
			return errors.NewMetaKVError(name.Name(), errs[0])
		}
	}
	name.changeCounter = metaStorage.NewChangeCounter()
	return nil
}

func (name *systemEntry) Delete() errors.Error {
	parts := name.path.Parts()
	if len(parts) != 4 {
		return errors.NewInvalidFunctionNameError(name.Name(), fmt.Errorf("name has %v parts", len(parts)))
	}
	systemCollection, err := getSystemCollection(parts[1])
	if err != nil {
		return err
	}
	scope, err := systemCollection.Scope().Bucket().ScopeByName(name.Path()[2])
	if err != nil {
		return err
	}
	return delete2(systemCollection, parts2key(scope.Uid(), parts...), name.Name())
}

func delete2(systemCollection datastore.Keyspace, key string, name string) errors.Error {
	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = key
	var errs errors.Errors
	var mCount int

	queryContext := datastore.GetDurableQueryContextFor(systemCollection)

	mCount, _, errs = systemCollection.Delete(dpairs, queryContext, false)
	if len(errs) > 0 {
		return errors.NewMetaKVError(name, errs[0])
	}
	if mCount == 0 {
		return errors.NewMissingFunctionError(name)
	}
	return nil
}

func (name *systemEntry) CheckStorage() bool {
	return name.changeCounter != metaStorage.ChangeCounter()
}

func (name *systemEntry) ResetStorage() {
	name.changeCounter = metaStorage.ChangeCounter()
}

func (name *systemEntry) InheritStorage(changeCnter int32) {
	name.changeCounter = changeCnter
}

var _STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(1)
