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

	"github.com/couchbaselabs/query/clustering"
	"github.com/couchbaselabs/query/clustering/stub"
	"github.com/couchbaselabs/query/clustering/zookeeper"

	"github.com/couchbaselabs/query/errors"
)

func NewConfigstore(uri string) (clustering.ConfigurationStore, errors.Error) {
	if strings.HasPrefix(uri, "zookeeper:") {
		return clustering_zk.NewConfigstore(uri)
	}

	if strings.HasPrefix(uri, "stub:") {
		return clustering_stub.NewConfigurationStore()
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid configstore uri: %s", uri))
}
