//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package constructor

import (
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/authorize"
	"github.com/couchbase/query/functions/bridge"
	"github.com/couchbase/query/functions/golang"
	"github.com/couchbase/query/functions/inline"
	"github.com/couchbase/query/functions/javascript"
	metaStorage "github.com/couchbase/query/functions/metakv"
	systemStorage "github.com/couchbase/query/functions/system"
	"github.com/couchbase/query/tenant"
	"github.com/gorilla/mux"
)

func Init(mux *mux.Router, threads int) {
	functionsBridge.NewFunctionName = newGlobalFunction
	functionsBridge.NewInlineBody = inline.NewInlineBody
	functionsBridge.NewGolangBody = golang.NewGolangBody
	functionsBridge.NewJavascriptBody = javascript.NewJavascriptBody
	authorize.Init()
	metaStorage.Init()
	systemStorage.Init()
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
		return metaStorage.NewGlobalFunction(namespace, elem[0])
	case 2:
		return metaStorage.NewGlobalFunction(ns, elem[1])
	case 4:
		if tenant.IsServerless() {
			return systemStorage.NewScopeFunction(ns, elem[1], elem[2], elem[3])
		} else {
			return metaStorage.NewScopeFunction(ns, elem[1], elem[2], elem[3])
		}
	default:
		return nil, errors.NewInvalidFunctionNameError(elem[len(elem)-1], fmt.Errorf("invalid function path"))
	}
}
