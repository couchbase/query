//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

//go:build !enterprise

package server

import (
	"github.com/couchbase/query/errors"
)

func setProfileAdmin(s *Server, o interface{}) errors.Error {
	return nil
}

func setControlsAdmin(s *Server, o interface{}) errors.Error {
	return nil
}

func GetProfileAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	return settings
}

func GetControlsAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	return settings
}

func checkProfileAdmin(val interface{}) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Profiling is an EE only feature")
}

func checkControlsAdmin(val interface{}) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Controls is an EE only feature")
}
