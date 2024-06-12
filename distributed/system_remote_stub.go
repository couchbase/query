//  Copyright 2016-Present Couchbase, Inc.
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
func (this systemRemoteStub) GetRemoteKeys(nodes []string, endpoint string, keyFn func(id string) bool,
	warnFn func(warn errors.Error), creds Creds, authToken string) {

	// nothing to see here
}

// get a specified remote document from a remote node
func (this systemRemoteStub) GetRemoteDoc(node string, key string, endpoint string, command string,
	docFn func(doc map[string]interface{}), warnFn func(warn errors.Error), creds Creds, authToken string,
	formData map[string]interface{}) {
	// ditto
}

// perform operation on keys on the specified nodes for the specified endpoint
func (this systemRemoteStub) DoRemoteOps(nodes []string, endpoint string, command string, key string, data string,
	warnFn func(warn errors.Error), creds Creds, authToken string) {

	// nothing to see here
}

// returns the local node identity, as known to the cluster
func (this systemRemoteStub) WhoAmI() string {

	// always local
	return ""
}

func (this systemRemoteStub) NodeUUID(host string) string {
	return ""
}

func (this systemRemoteStub) UUIDToHost(uuid string) string {
	return ""
}

func (this systemRemoteStub) Starting() bool {

	// always local
	return false
}

func (this systemRemoteStub) StandAlone() bool {

	// always local
	return true
}

func (this systemRemoteStub) Clustered() bool {

	// always local
	return false
}

// get the node names
func (this systemRemoteStub) GetNodeNames() []string {
	var names []string

	return names
}

// check capability
// for standalone engines (where this applies) all capabilities are enabled by default for testing purposes
func (this systemRemoteStub) Enabled(capability Capability) bool {
	return true
}

// change settings
func (this systemRemoteStub) Settings(setting map[string]interface{}) errors.Error {
	return nil
}

// Update TLS or node-to-node encryption settings.
func (this systemRemoteStub) SetConnectionSecurityConfig(caFile string, certFile string, encryptNodeToNodeComms bool,
	clientCertAuthMandatory bool, internalClientCertFile string, internalClientKeyFile string,
	internalClientPrivateKeyPassphrase []byte) {
	// Do nothing.
}
