//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
		warnFn func(warn errors.Error), creds Creds, authToken string)

	// collect a document for a keyspace from a remote node
	// docFn processes the document
	// warnFn issues warnings
	GetRemoteDoc(node string, key string, endpoint string, command string,
		docFn func(doc map[string]interface{}),
		warnFn func(warn errors.Error), creds Creds, authToken string, formData map[string]interface{})

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
	SetConnectionSecurityConfig(caFile, certFile string, encryptNodeToNodeComms bool)

	// Prepare an opaque type that represents this admin endpoint and necessary authentication
	PrepareAdminOp(node string, endpoint string, key string, warnFn func(warn errors.Error), creds Creds,
		authToken string) (interface{}, errors.Error)

	// Execute an operation against previously prepared admin endpoint
	ExecutePreparedAdminOp(op interface{}, command string, data string, warnFn func(warn errors.Error), creds Creds,
		authToken string) ([]byte, errors.Error)

	// Retrieve a host's UUID
	NodeUUID(string) string

	// Retrieve hostname for given UUID
	UUIDToHost(string) string
}

// It would be convenient to use datastore/Credentials here, but that causes an import circularity,
// so we define an equivalent here.
type Creds string

type Capability int

const (
	NEW_PREPAREDS = Capability(iota)
	NEW_OPTIMIZER
	NEW_INDEXADVISOR
	NEW_JAVASCRIPT_FUNCTIONS
	NEW_INLINE_FUNCTIONS
	KV_RANGE_SCANS
	READ_FROM_REPLICA
)

var NO_CREDS = Creds("")

var _REMOTEACCESS SystemRemoteAccess = NewSystemRemoteStub()

func SetRemoteAccess(remoteAccess SystemRemoteAccess) {
	_REMOTEACCESS = remoteAccess
}

func RemoteAccess() SystemRemoteAccess {
	return _REMOTEACCESS
}
