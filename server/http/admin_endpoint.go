//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package http

import (
	"encoding/json"
	"net/http"

	"github.com/couchbase/query/errors"
)

const (
	adminPrefix = "/admin"
)

type apiFunc func(*HttpEndpoint, http.ResponseWriter, *http.Request) (interface{}, errors.Error)

type handlerFunc func(http.ResponseWriter, *http.Request)

func (this *HttpEndpoint) wrapAPI(w http.ResponseWriter, req *http.Request, f apiFunc) {
	obj, err := f(this, w, req)
	if err != nil {
		writeError(w, err)
		return
	}

	if obj == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	buf, json_err := json.Marshal(obj)
	if json_err != nil {
		writeError(w, errors.NewAdminDecodingError(json_err))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func writeError(w http.ResponseWriter, err errors.Error) {
	w.Header().Set("Content-Type", "application/json")
	buf, er := json.Marshal(err)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	status := mapErrorToHttpStatus(err)
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func mapErrorToHttpStatus(err errors.Error) int {
	switch err.Code() {
	case errors.ADMIN_AUTH_ERROR:
		return http.StatusUnauthorized
	case errors.ADMIN_SSL_NOT_ENABLED:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func AdminPrefix() string {
	return adminPrefix
}
