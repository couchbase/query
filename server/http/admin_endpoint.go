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
	"strconv"
	"strings"

	"github.com/couchbaselabs/query/errors"
	"github.com/couchbaselabs/query/server"
	"github.com/gorilla/mux"
)

const (
	adminPrefix = "/admin"
)

type apiFunc func(*server.Server, http.ResponseWriter, *http.Request) (interface{}, errors.Error)

type handlerFunc func(http.ResponseWriter, *http.Request)

// admin_endpoint

func registerAdminHandlers(server *server.Server) {
	r := mux.NewRouter()
	registerClusterHandlers(r, server)
	registerAccountingHandlers(r, server)
	http.Handle(adminPrefix+"/", r)
}

func wrapAPI(s *server.Server, w http.ResponseWriter, req *http.Request, f apiFunc) {
	obj, err := f(s, w, req)
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
		writeError(w, errors.NewError(json_err, ""))
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
	return http.StatusInternalServerError
}

func GetAdminURL(host string, port int) string {
	urlParts := []string{"http://", host, ":", strconv.Itoa(port), adminPrefix}
	return strings.Join(urlParts, "")
}
