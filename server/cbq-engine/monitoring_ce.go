//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise
// +build !enterprise

package main

import (
	"flag"

	"github.com/couchbase/query/clustering"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
)

var PROFILE = flag.String("profile", "off", "EE only - Profiling state: off, phases, timings")
var CONTROLS = flag.Bool("controls", false, "Response to include controls section")

func monitoringInit(configstore clustering.ConfigurationStore) (server.Profile, bool, errors.Error) {
	var err errors.Error

	prof, _ := server.ParseProfile(*PROFILE, false)
	if prof != server.ProfOff {
		err = errors.NewNotImplemented("Profiling is an EE only feature")
	}
	return server.ProfOff, *CONTROLS, err
}
