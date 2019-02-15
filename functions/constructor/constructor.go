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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	globalName "github.com/couchbase/query/functions/metakv"
)

func Init() {
	functions.Constructor = newGlobalFunction
}

// TODO switch to collections context
func newGlobalFunction(elem []string, namespace string) (functions.FunctionName, errors.Error) {
	if len(elem) == 2 {
		return globalName.NewGlobalFunction(elem[0], elem[1])
	} else if namespace == "" {
		return nil, errors.NewInvalidFunctionNameError(elem[0])
	}
	return globalName.NewGlobalFunction(namespace, elem[0])
}
