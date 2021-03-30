//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
	"strings"

	"github.com/couchbase/query/auth"
)

func CredsArray(creds *auth.Credentials) []string {
	credsLen := 1
	if creds != nil {
		credsLen += len(creds.Users)
	}
	credsList := make([]string, 0, credsLen)
	credsMap := make(map[string]bool, credsLen)
	if credsLen > 1 {
		for k := range creds.Users {
			if k == "" {
				continue
			}
			if _, found := credsMap[k]; found {
				continue
			}
			credsMap[k] = true
			credsList = append(credsList, k)
		}
	}
	ds := GetDatastore()
	if ds != nil && creds != nil {
		reqName := ds.CredsString(creds.HttpRequest)
		if reqName != "" {
			if _, found := credsMap[reqName]; !found {
				credsMap[reqName] = true
				credsList = append(credsList, reqName)
			}
		}
	}
	return credsList
}

func CredsString(creds *auth.Credentials) string {
	return strings.Join(CredsArray(creds), ",")
}
