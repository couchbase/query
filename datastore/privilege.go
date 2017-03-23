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
	"net/http"
	"strings"

	"github.com/couchbase/query/auth"
)

func CredsString(creds auth.Credentials, req *http.Request) string {
	credsLen := 1
	if creds != nil {
		credsLen += len(creds)
	}
	credsList := make([]string, 0, credsLen)
	if credsLen > 1 {
		for k := range creds {
			if k == "" {
				continue
			}
			credsList = append(credsList, k)
		}
	}
	ds := GetDatastore()
	if ds != nil {
		reqName := ds.CredsString(req)
		if reqName != "" {
			credsList = append(credsList, reqName)
		}
	}
	return strings.Join(credsList, ",")
}
