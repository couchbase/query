//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package tenant

import (
	"github.com/couchbase/cbauth/service"
	"github.com/couchbase/regulator"
	"github.com/couchbase/regulator/factory"
	"github.com/gorilla/mux"
)

var isServerless bool

func Init(mux *mux.Router, nodeid string, cafile string, serverless bool) {
	isServerless = serverless
	if !serverless {
		return
	}
	handle := factory.InitRegulator(regulator.InitSettings{NodeID: service.NodeID(nodeid), TlsCAFile: cafile, Service: regulator.Query})
	mux.Handle(regulator.MeteringEndpoint, handle).Methods("GET")
}

func IsServerless() bool {
	return isServerless
}
