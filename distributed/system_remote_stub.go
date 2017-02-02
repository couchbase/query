//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package distributed

import (
	"github.com/couchbase/query/errors"
)

// a stub implementation of SystemRemoteAccess
type systemRemoteStub struct{}

func NewSystemRemoteStub() SystemRemoteAccess {
	return &systemRemoteStub{}
}

// construct a key from node name and local key
func (this systemRemoteStub) MakeKey(node string, key string) string {

	// always local
	return key
}

// split global key into name and local key
func (this systemRemoteStub) SplitKey(key string) (string, string) {

	// always local
	return "", key
}

// get remote keys from the specified nodes for the specified endpoint
func (this systemRemoteStub) GetRemoteKeys(nodes []string, endpoint string, keyFn func(id string), warnFn func(warn errors.Error)) {

	// nothing to see here
}

// get a specified remote document from a remote node
func (this systemRemoteStub) GetRemoteDoc(node string, key string, endpoint string, command string,
	docFn func(doc map[string]interface{}), warnFn func(warn errors.Error), creds Creds) {
	// ditto
}

// returns the local node identity, as known to the cluster
func (this systemRemoteStub) WhoAmI() string {

	// always local
	return ""
}

// get the node names
func (this systemRemoteStub) GetNodeNames() []string {
	var names []string

	return names
}
