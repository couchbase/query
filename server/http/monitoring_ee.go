//  Copyright 2017-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package http

import (
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
)

func getProfileRequest(a httpRequestArgs, parm string, val interface{}) (server.Profile, errors.Error) {
	profile, err := a.getStringVal(parm, val)
	if err == nil && profile != "" {
		prof, ok := server.ParseProfile(profile, true)
		if ok {
			return prof, nil
		} else {
			err = errors.NewServiceErrorUnrecognizedValue(PROFILE, profile)
		}

	}
	return server.ProfUnset, err
}
