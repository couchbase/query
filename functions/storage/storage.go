//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package storage

import (
	"encoding/json"
	go_errors "errors"
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/metakv"
	"github.com/couchbase/query/functions/system"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/value"
)

func MakeName(bytes []byte) (functions.FunctionName, error) {
	var name_type struct {
		Type string `json:"type"`
	}

	err := json.Unmarshal(bytes, &name_type)
	if err != nil {
		return nil, err
	}

	switch name_type.Type {
	case "global":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		}

		err = json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, err
		}
		return metaStorage.NewGlobalFunction(_unmarshalled.Namespace, _unmarshalled.Name)

	case "scope":
		var _unmarshalled struct {
			_         string `json:"type"`
			Namespace string `json:"namespace"`
			Bucket    string `json:"bucket"`
			Scope     string `json:"scope"`
			Name      string `json:"name"`
		}
		err := json.Unmarshal(bytes, &_unmarshalled)
		if err != nil {
			return nil, err
		}
		if _unmarshalled.Namespace == "" || _unmarshalled.Bucket == "" || _unmarshalled.Scope == "" || _unmarshalled.Name == "" {
			return nil, go_errors.New("incomplete function name")
		}
		if tenant.IsServerless() {
			return systemStorage.NewScopeFunction(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Name)
		} else {
			return metaStorage.NewScopeFunction(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Name)
		}
	default:
		return nil, fmt.Errorf("unknown name type %v", name_type.Type)
	}
}

func IsInternal(val interface{}) (bool, error) {
	var t string

	switch val := val.(type) {
	case []byte:
		var outer struct {
			Definition json.RawMessage `json:"definition"`
		}
		var language_type struct {
			Language string `json:"#language"`
		}

		err := json.Unmarshal(val, &outer)
		if err != nil {
			return false, err
		}

		err = json.Unmarshal(outer.Definition, &language_type)
		if err != nil {
			return false, err
		}
		t = language_type.Language
	default:
		logging.Infof("entry is %v", val)
		d, _ := val.(value.Value).Field("definition")
		if d != nil {
			v, _ := d.(value.Value).Field("#language")
			t, _ = v.Actual().(string)
		}
	}

	switch t {
	case "inline":
		return true, nil
	case "golang":
		return false, nil
	case "javascript":
		return false, nil
	default:
		return false, fmt.Errorf("unknown function type %v", t)
	}
}

func DropScope(namespace, bucket, scope string) {
	if tenant.IsServerless() {
		systemStorage.DropScope(namespace, bucket, scope)
	} else {
		metaStorage.DropScope(namespace, bucket, scope)
	}
}

func Count(bucket string) (int64, error) {
	if bucket != "" && tenant.IsServerless() {
		return systemStorage.Count(bucket)
	} else {
		return metaStorage.Count(bucket)
	}
}

func Get(key string) (value.Value, error) {
	if tenant.IsServerless() && algebra.PartsFromPath(key) == 4 {
		return systemStorage.Get(key)
	} else {
		return metaStorage.Get(key)
	}
}

func Foreach(bucket string, f func(path string, v value.Value) error) error {
	if bucket != "" && tenant.IsServerless() {
		return systemStorage.Foreach(bucket, f)
	} else {
		return metaStorage.Foreach(bucket, f)
	}
}

func Scan(bucket string, f func(path string) error) error {
	if bucket != "" && tenant.IsServerless() {
		return systemStorage.Scan(bucket, f)
	} else {
		return metaStorage.Scan(bucket, f)
	}
}

func ExternalBucketArchive() bool {
	return !tenant.IsServerless()
}
