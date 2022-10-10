//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build enterprise

package server

import (
	"github.com/couchbase/query/errors"
)

func setProfileAdmin(s *Server, o interface{}) errors.Error {
	value, _ := o.(string)
	prof, ok := ParseProfile(value)
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
