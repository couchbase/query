//  Copyright (c) 2017 Couchbase, Inc.
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

// Interface to access remote system data over various protocols
type SystemRemoteAccess interface {
	MakeKey(node string, key string) string // given a node and a local key, produce a cluster key
	SplitKey(key string) (string, string)   // given a cluster key, return node and local key

	GetRemoteKeys(nodes []string, endpoint string, keyFn func(id string),
		warnFn func(warn errors.Error)) // collect cluster keys for a keyspace from a set of remote nodes
	GetRemoteDoc(node string, key string, endpoint string, command string,
		docFn func(doc map[string]interface{}),
		warnFn func(warn errors.Error)) // collect a document for a keyspace from a remote node
	WhoAmI() string // local node name, if known
}

var _REMOTEACCESS SystemRemoteAccess = NewSystemRemoteStub()

func SetRemoteAccess(remoteAccess SystemRemoteAccess) {
	_REMOTEACCESS = remoteAccess
}

func RemoteAccess() SystemRemoteAccess {
	return _REMOTEACCESS
}
