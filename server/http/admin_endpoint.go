//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/couchbase/query/audit"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"

	adt "github.com/couchbase/goutils/go-cbaudit"
)

const (
	adminPrefix = "/admin"
)

type apiFunc func(*HttpEndpoint, http.ResponseWriter, *http.Request, *audit.ApiAuditFields) (interface{}, errors.Error)

type handlerFunc func(http.ResponseWriter, *http.Request)

type textPlain string

func (this *HttpEndpoint) wrapAPI(w http.ResponseWriter, req *http.Request, f apiFunc, log bool) {
	auditFields := audit.ApiAuditFields{
		GenericFields: adt.GetAuditBasicFields(req),
		RemoteAddress: req.RemoteAddr,
		HttpMethod:    req.Method,
		LocalAddress:  req.Host,
	}

	var obj interface{}
	var err errors.Error

	if !log {
		obj, err = f(this, w, req, &auditFields)
	} else {
		logging.Infof("%v %v request received.", req.URL.String(), req.Method)
		start := time.Now()
		obj, err = f(this, w, req, &auditFields)
		dur := time.Now().Sub(start)
		em := ""
		if err != nil {
			em = fmt.Sprintf(" Error: %v", err)
		} else if obj == nil {
			em = " No output."
		}
		logging.Infof("%v %v access completed in %v.%v", req.URL.String(), req.Method, util.OutputDuration(dur), em)
	}
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

	text, ok := obj.(textPlain)
	if ok {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(text))
	} else {
		p := req.FormValue("pretty")
		pretty, _ := strconv.ParseBool(p)
		buf := &bytes.Buffer{}
		var json_err error
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if pretty {
			enc.SetIndent("", "\t")
		}
		json_err = enc.Encode(obj)
		if json_err != nil {
			logging.Infof("Error writing output for %v: %v", req.URL.String(), json_err)
			e := errors.NewAdminDecodingError(json_err)
			status := writeError(w, e)

			auditFields.HttpResultCode = status
			auditFields.ErrorCode = int(e.Code())
			auditFields.ErrorMessage = e.Error()
			audit.SubmitApiRequest(&auditFields)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		buf.WriteTo(w)
	}

	auditFields.HttpResultCode = http.StatusOK
	audit.SubmitApiRequest(&auditFields)
}

func (this *HttpEndpoint) WriteError(err errors.Error, w http.ResponseWriter, req *http.Request) {
	writeError(w, err)
}

// Returns the HTTP error code, e.g. 500.
func writeError(w http.ResponseWriter, err errors.Error) int {
	w.Header().Set("Content-Type", "application/json")
	buf, er := json.Marshal(err)
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return http.StatusInternalServerError
	}
	status := mapErrorToHttpResponse(err, http.StatusInternalServerError)
	if err.Code() != errors.E_ADMIN_LOG {
		w.WriteHeader(status)
	}
	w.Write(buf)
	return status
}

func AdminPrefix() string {
	return adminPrefix
}
