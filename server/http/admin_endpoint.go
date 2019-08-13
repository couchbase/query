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

	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/errors"

	adt "github.com/couchbase/goutils/go-cbaudit"
)

const (
	adminPrefix = "/admin"
)

type apiFunc func(*HttpEndpoint, http.ResponseWriter, *http.Request, *audit.ApiAuditFields) (interface{}, errors.Error)

type handlerFunc func(http.ResponseWriter, *http.Request)

func (this *HttpEndpoint) wrapAPI(w http.ResponseWriter, req *http.Request, f apiFunc) {
	auditFields := audit.ApiAuditFields{
		GenericFields: adt.GetAuditBasicFields(req),
		RemoteAddress: req.RemoteAddr,
		HttpMethod:    req.Method,
	}

	obj, err := f(this, w, req, &auditFields)
	if err != nil {
		status := writeError(w, err)

		auditFields.HttpResultCode = status
		auditFields.ErrorCode = int(err.Code())
		auditFields.ErrorMessage = err.Error()
		audit.SubmitApiRequest(&auditFields)
		return
	}

	if obj == nil {
		w.WriteHeader(http.StatusNotFound)

		auditFields.HttpResultCode = http.StatusNotFound
		audit.SubmitApiRequest(&auditFields)
		return
	}

	buf, json_err := json.Marshal(obj)
	if json_err != nil {
		e := errors.NewAdminDecodingError(json_err)
		status := writeError(w, e)

		auditFields.HttpResultCode = status
		auditFields.ErrorCode = int(e.Code())
		auditFields.ErrorMessage = e.Error()
		audit.SubmitApiRequest(&auditFields)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)

	auditFields.HttpResultCode = http.StatusOK
	audit.SubmitApiRequest(&auditFields)
}

// Returns the HTTP error code, e.g. 500.
func writeError(w http.ResponseWriter, err errors.Error) int {
	w.Header().Set("Content-Type", "application/json")
	buf, er := json.Marshal(err)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return http.StatusInternalServerError
	}
	status := mapErrorToHttpStatus(err)
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
	return status
}

func mapErrorToHttpStatus(err errors.Error) int {
	switch err.Code() {
	case errors.ADMIN_AUTH_ERROR:
		return http.StatusUnauthorized
	case errors.ADMIN_SSL_NOT_ENABLED:
		return http.StatusNotFound
	case errors.DS_AUTH_ERROR:
		return http.StatusUnauthorized
	case errors.ADMIN_CREDS_ERROR:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func AdminPrefix() string {
	return adminPrefix
}
