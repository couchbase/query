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
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

// currently nothing to do
func Init() {
}

// datastore and function store actions
func DropScope(namespace, bucket, scope string) {
	scanSystemCollection(bucket, func(key string, systemCollection datastore.Keyspace) error {
		s, fn := key2parts(key)

		// skip entries that are not udfs or different scopes
		if scope == s {
			path := algebra.PathFromParts(namespace, bucket, scope, fn)
			delete2(systemCollection, key, path)
			functions.FunctionClear(path, nil)
		}
		return nil
	})
}

func Foreach(b string, f func(path string, v value.Value) error) error {
	return scanSystemCollection(b, func(key string, systemCollection datastore.Keyspace) error {
		s, fn := key2parts(key)

		// skip entries that are not udfs
		if s == "" {
			return nil
		}
		name, _ := NewScopeFunction(systemCollection.NamespaceId(), b, s, fn)
		val, err := get2(name, systemCollection)
		if err != nil {
			return err
		}
		return f(algebra.PathFromParts(systemCollection.NamespaceId(), b, s, fn), val)
	})
}

func Scan(b string, f func(path string) error) error {
	return scanSystemCollection(b, func(key string, systemCollection datastore.Keyspace) error {
		s, fn := key2parts(key)

		// skip entries that are not udfs
		if s == "" {
			return nil
		}
		return f(algebra.PathFromParts(systemCollection.NamespaceId(), b, s, fn))
	})
}

func Get(path string) (value.Value, error) {
	parts := algebra.ParsePath(path)
	if len(parts) != 4 {
		return nil, errors.NewInvalidFunctionNameError(path, fmt.Errorf("name has %v parts", len(parts)))
	}
	systemCollection, err := getSystemCollection(parts[1], false)
	if err != nil {
		return nil, err
	}
	name, _ := NewScopeFunction(parts[0], parts[1], parts[2], parts[3])
	return get2(name, systemCollection)
}

func get2(name functions.FunctionName, systemCollection datastore.Keyspace) (value.Value, error) {
	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)
	key := parts2key(name.Path()...)
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

	err := scanSystemCollection(b, func(key string, systemCollection datastore.Keyspace) error {
		count++
		return nil
	})
	return count, err
}

func getSystemCollection(bucketName string, chkIndex bool) (datastore.Keyspace, errors.Error) {
	store := datastore.GetDatastore()
	if store == nil {
		return nil, errors.NewNoDatastoreError()
	}

	if chkIndex {
		// makre sure system collection exists, and create primary index if not existing
		requestId, _ := util.UUIDV4()
		err := store.CheckSystemCollection(bucketName, requestId)
		if err != nil {
			return nil, err
		}
	}

	return store.GetSystemCollection(bucketName)
}

func getPrimaryIndex(sysColl datastore.Keyspace) (datastore.PrimaryIndex3, errors.Error) {
	indexer, err := sysColl.Indexer(datastore.GSI)
	if err != nil {
		return nil, err
	}
	indexer3, ok := indexer.(datastore.Indexer3)
	if !ok {
		return nil, errors.NewInvalidGSIIndexerError("Cannot load from system collection")
	}
	index, err := indexer3.IndexByName("#primary")
	if err != nil {
		return nil, err
	}
	index3, ok := index.(datastore.PrimaryIndex3)
	if !ok {
		return nil, errors.NewInvalidGSIIndexError(index.Name())
	}

	return index3, nil
}

func scanSystemCollection(bucketName string, handler func(string, datastore.Keyspace) error) errors.Error {
	systemCollection, err := getSystemCollection(bucketName, true)
	if err != nil {
		return err
	}

	index3, err := getPrimaryIndex(systemCollection)
	if err != nil {
		return err
	}

	conn := datastore.NewIndexConnection(datastore.NULL_CONTEXT)
	defer conn.Dispose()
	defer conn.SendStop()

	requestId, err1 := util.UUIDV4()
	if err1 != nil {
		return errors.NewSystemCollectionError("error generating requestId", err1)
	}

	go index3.ScanEntries3(requestId, nil, 0, 0, nil, nil, datastore.UNBOUNDED, nil, conn)

	var item *datastore.IndexEntry
	ok := true
	for ok {
		// logic from execution/base.getItemEntry()
		item, ok = conn.Sender().GetEntry()
		if ok {
			if item != nil {
				err1 = handler(item.PrimaryKey, systemCollection)
				if err1 != nil {
					return errors.NewSystemCollectionError("error calling handler", err1)
				}
			} else {
				ok = false
			}
		}
	}

	return nil
}

const _PREFIX = "udf::"

func key2parts(key string) (string, string) {
	if key[:len(_PREFIX)] != _PREFIX {
		return "", ""
	}
	key = key[len(_PREFIX):]
	dot := strings.IndexByte(key, '.')
	if dot <= 1 {
		return "", ""
	}
	return key[:dot], key[dot+1:]
}

func parts2key(parts ...string) string {
	return _PREFIX + parts[2] + "." + parts[3]
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

func (name *systemEntry) IsGlobal() bool {
	return false
}

func (name *systemEntry) QueryContext() string {
	return name.path.QueryContext()
}

func (name *systemEntry) Signature(object map[string]interface{}) {
	object["namespace"] = name.path.Namespace()
	object["bucket"] = name.path.Bucket()
	object["scope"] = name.path.Scope()
	object["name"] = name.path.Keyspace()
	object["type"] = "scope"
}

func (name *systemEntry) Load() (functions.FunctionBody, errors.Error) {
	systemCollection, err := getSystemCollection(name.path.Parts()[1], false)
	if err != nil {
		return nil, err
	}
	fetchMap := _STRING_ANNOTATED_POOL.Get()
	defer _STRING_ANNOTATED_POOL.Put(fetchMap)
	key := parts2key(name.Path()...)
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
	name.changeCounter = metaStorage.ChangeCounter()

	// unmarshal body
	body := value.ToJSON(val.GetValue())
	if body == nil {
		return nil, errors.NewFunctionEncodingError("decode", name.Name(), fmt.Errorf("function entry is not a document"))
	}

	// determine language and create body from definition
	b, e := resolver.MakeBody(name.Name(), body)
	val.Recycle()
	return b, e
}

func (name *systemEntry) Save(body functions.FunctionBody, replace bool) errors.Error {
	entry := make(map[string]interface{})
	body.Body(entry)

	parts := name.path.Parts()
	if len(parts) != 4 {
		return errors.NewInvalidFunctionNameError(name.Name(), fmt.Errorf("name has %v parts", len(parts)))
	}
	systemCollection, err := getSystemCollection(parts[1], false)
	if err != nil {
		return err
	}

	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = parts2key(parts...)
	dpairs[0].Value = value.NewValue(entry)
	var errs errors.Errors

	if replace {
		_, _, errs = systemCollection.Upsert(dpairs, datastore.NULL_QUERY_CONTEXT, false)
	} else {
		_, _, errs = systemCollection.Insert(dpairs, datastore.NULL_QUERY_CONTEXT, false)
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
	systemCollection, err := getSystemCollection(parts[1], false)
	if err != nil {
		return err
	}
	return delete2(systemCollection, parts2key(parts...), name.Name())
}

func delete2(systemCollection datastore.Keyspace, key string, name string) errors.Error {
	dpairs := make([]value.Pair, 1)
	dpairs[0].Name = key
	var errs errors.Errors
	var mCount int

	mCount, _, errs = systemCollection.Delete(dpairs, datastore.NULL_QUERY_CONTEXT, false)
	if len(errs) > 0 {
		return errors.NewMetaKVError(name, errs[0])
	}
	if mCount == 0 {
		errors.NewMissingFunctionError(name)
	}
	return nil
}

func (name *systemEntry) CheckStorage() bool {
	return name.changeCounter != metaStorage.ChangeCounter()
}

func (name *systemEntry) ResetStorage() {
	name.changeCounter = metaStorage.ChangeCounter()
}

var _STRING_ANNOTATED_POOL = value.NewStringAnnotatedPool(1)
