//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package server

import (
	"github.com/couchbase/query/errors"
)

func setProfileAdmin(s *Server, o any) errors.Error {
	return nil
}

func setControlsAdmin(s *Server, o any) errors.Error {
	return nil
}

func GetProfileAdmin(settings map[string]any, srvr *Server) map[string]any {
	return settings
}

func GetControlsAdmin(settings map[string]any, srvr *Server) map[string]any {
	return settings
}

func checkProfileAdmin(val any) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Profiling is an EE only feature")
}

func checkControlsAdmin(val any) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Controls is an EE only feature")
}
