//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

//go:build enterprise

package main

import (
	"flag"
	index_advisor "github.com/couchbase/query-ee/indexadvisor"
	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/server/http"
)

// Profiling
var PROFILE = flag.String("profile", "off", "Profiling state: off, phases, timings")
var CONTROLS = flag.Bool("controls", false, "Response to include controls section")

func monitoringInit(configstore clustering.ConfigurationStore) (server.Profile, bool, errors.Error) {
	distributed.SetRemoteAccess(http.NewSystemRemoteAccess(configstore))
	prof, _ := server.ParseProfile(*PROFILE, false)

	index_advisor.SetConfigStore(configstore)
	return prof, *CONTROLS, nil
}
