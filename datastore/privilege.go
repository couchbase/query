//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
