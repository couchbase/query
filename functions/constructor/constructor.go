//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package constructor

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/golang"
	"github.com/couchbase/query/functions/inline"
	"github.com/couchbase/query/functions/javascript"
	storage "github.com/couchbase/query/functions/metakv"
	"github.com/gorilla/mux"
)

func Init(mux *mux.Router) {
	functions.Constructor = newGlobalFunction
	storage.Init()
	golang.Init()
	inline.Init()
	javascript.Init(mux)
}

func newGlobalFunction(elem []string, namespace string, queryContext string) (functions.FunctionName, errors.Error) {
	var ns string

	if len(elem) == 1 && queryContext != "" {
		newElem := algebra.ParseQueryContext(queryContext)
		elem = append(newElem, elem[0])
	}
	if len(elem) == 1 || elem[0] == "" {
		ns = namespace
	} else {
		ns = elem[0]
	}

	if ns == "" {
		return nil, errors.NewInvalidFunctionNameError(elem[len(elem)-1], fmt.Errorf("namespace not specified"))
	}
	switch len(elem) {
	case 1:
		return storage.NewGlobalFunction(namespace, elem[0])
	case 2:
		return storage.NewGlobalFunction(ns, elem[1])
	case 4:
		return storage.NewScopeFunction(ns, elem[1], elem[2], elem[3])
	default:
		return nil, errors.NewInvalidFunctionNameError(elem[len(elem)-1], fmt.Errorf("invalid function path"))
	}
}
