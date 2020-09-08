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

	// given a node and a local key, produce a cluster key
	MakeKey(node string, key string) string

	// given a cluster key, return node and local key
	SplitKey(key string) (string, string)

	// collect cluster keys for a keyspace from a set of remote nodes
	// keyFn processes each key, returns false to stop processing keys
	// warnFn issues warnings
	GetRemoteKeys(nodes []string, endpoint string, keyFn func(id string) bool,
		warnFn func(warn errors.Error))

	// collect a document for a keyspace from a remote node
	// docFn processes the document
	// warnFn issues warnings
	GetRemoteDoc(node string, key string, endpoint string, command string,
		docFn func(doc map[string]interface{}),
		warnFn func(warn errors.Error), creds Creds, authToken string)

	// Perform an operation on a key on all nodes in the argument
	// data is sent to each remote node
	// warnFn is called on the result of each node, with no warnigs if succesful
	DoRemoteOps(nodes []string, endpoint string, command string, key string,
		data string, warnFn func(warn errors.Error), creds Creds, authToken string)

	// local node name, if known
	WhoAmI() string

	// node is not yet part of a cluster
	Starting() bool

	// node is part of a cluster
	Clustered() bool

	// node is not part of a clustered
	StandAlone() bool

	// all the node names
	GetNodeNames() []string

	// is a specific feature available in all clusters?
	Enabled(capability Capability) bool

	// dynamically change settings
	Settings(settings map[string]interface{}) errors.Error

	// Update TLS or node-to-node encryption settings.
	SetConnectionSecurityConfig(certFile string, encryptNodeToNodeComms bool)
}

// It would be convenient to use datastore/Credentials here, but that causes an import circularity,
// so we define an equivalent here.
type Creds map[string]string

type Capability int

const (
	NEW_PREPAREDS = Capability(iota)
	NEW_OPTIMIZER
	NEW_INDEXADVISOR
	NEW_JAVASCRIPT_FUNCTIONS
	NEW_INLINE_FUNCTIONS
)

var NO_CREDS = make(Creds, 0)

var _REMOTEACCESS SystemRemoteAccess = NewSystemRemoteStub()

func SetRemoteAccess(remoteAccess SystemRemoteAccess) {
	_REMOTEACCESS = remoteAccess
}

func RemoteAccess() SystemRemoteAccess {
	return _REMOTEACCESS
}
