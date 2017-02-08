//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// +build !enterprise

package http

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
)

func checkProfileAdmin(val interface{}) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Profiling is an EE only feature")
}

func checkControlsAdmin(val interface{}) (bool, errors.Error) {
	return false, errors.NewNotImplemented("Controls is an EE only feature")
}

func setProfileAdmin(s *server.Server, o interface{}) {
}

func setControlsAdmin(s *server.Server, o interface{}) {
}

func getProfileAdmin(settings map[string]interface{}, srvr *server.Server) map[string]interface{} {
	return settings
}

func getControlsAdmin(settings map[string]interface{}, srvr *server.Server) map[string]interface{} {
	return settings
}

func getProfileRequest(a httpRequestArgs) (server.Profile, errors.Error) {
	return server.ProfUnset, nil
}

func getControlsRequest(a httpRequestArgs) (value.Tristate, errors.Error) {
	return value.NONE, nil
}
