//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build !enterprise
// +build !enterprise

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
