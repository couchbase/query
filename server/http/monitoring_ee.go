//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

//go:build enterprise

package http

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
)

func getProfileRequest(a httpRequestArgs, parm string, val interface{}) (server.Profile, errors.Error) {
	profile, err := a.getStringVal(parm, val)
	if err == nil && profile != "" {
		prof, ok := server.ParseProfile(profile)
		if ok {
			return prof, nil
		} else {
			err = errors.NewServiceErrorUnrecognizedValue(PROFILE, profile)
		}

	}
	return server.ProfUnset, err
}

func getControlsRequest(a httpRequestArgs, parm string, val interface{}) (value.Tristate, errors.Error) {
	return a.getTristateVal(parm, val)
}
