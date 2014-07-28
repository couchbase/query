//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package resolver

import (
	"fmt"
	"strings"

	"github.com/couchbaselabs/query/datastore"
	"github.com/couchbaselabs/query/datastore/file"
	"github.com/couchbaselabs/query/datastore/mock"
	"github.com/couchbaselabs/query/errors"
)

func NewDatastore(uri string) (datastore.Datastore, errors.Error) {
	if strings.HasPrefix(uri, ".") || strings.HasPrefix(uri, "/") {
		return file.NewDatastore(uri)
	}

	if strings.HasPrefix(uri, "dir:") {
		return file.NewDatastore(uri[4:])
	}

	if strings.HasPrefix(uri, "mock:") {
		return mock.NewDatastore(uri)
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid datastore uri: %s", uri))
}
