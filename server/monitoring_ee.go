//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise
// +build enterprise

package server

import (
	"github.com/couchbase/query/errors"
)

func setProfileAdmin(s *Server, o interface{}) errors.Error {
	value, _ := o.(string)
	prof, ok := ParseProfile(value, false)
	if ok {
		s.SetProfile(prof)
	}
	return nil
}

func setControlsAdmin(s *Server, o interface{}) errors.Error {
	value, _ := o.(bool)
	s.SetControls(value)
	return nil
}

func GetProfileAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	settings[PROFILE] = srvr.Profile().String()
	return settings
}

func GetControlsAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	settings[CONTROLS] = srvr.Controls()
	return settings
}

func checkProfileAdmin(val interface{}) (bool, errors.Error) {
	_, ok := val.(string)
	return ok, nil
}

func checkControlsAdmin(val interface{}) (bool, errors.Error) {
	_, ok := val.(bool)
	return ok, nil
}
