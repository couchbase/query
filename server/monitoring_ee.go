//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build enterprise

package server

import (
	paramSettings "github.com/couchbase/query/server/settings"
)

func setProfileAdmin(s *Server, o interface{}) {
	value, _ := o.(string)
	prof, ok := ParseProfile(value)
	if ok {
		s.SetProfile(prof)
	}
}

func setControlsAdmin(s *Server, o interface{}) {
	value, _ := o.(bool)
	s.SetControls(value)
}

func GetProfileAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	settings[paramSettings.PROFILE] = srvr.Profile().String()
	return settings
}

func GetControlsAdmin(settings map[string]interface{}, srvr *Server) map[string]interface{} {
	settings[paramSettings.CONTROLS] = srvr.Controls()
	return settings
}
