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
	"container/list"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/execution"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/tenant"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

const (
	PRETTY_RESULT_PREFIX string = "        "
	PRETTY_RESULT_INDENT string = "    "
	PRETTY_PREFIX        string = "    "
	PRETTY_INDENT        string = "    "
	NO_PRETTY_PREFIX     string = ""
	NO_PRETTY_INDENT     string = ""
)

const _DEF_IO_WRITE_TIME_LIMIT = 75 * time.Second
const _IO_WRITE_MONITOR_PRECISION = 10 * time.Second
const _MIN_IO_WRITE_TIME_LIMIT = _IO_WRITE_MONITOR_PRECISION + 1

func (this *httpRequest) Output() execution.Output {
	return this
}

func (this *httpRequest) Fail(err errors.Error) {
	if this.ServiceTime().IsZero() {
		this.SetServiceTime()
	}
	this.SetState(server.FATAL)
	// Determine the appropriate http response code based on the error
	httpRespCode := mapErrorToHttpResponse(err, http.StatusInternalServerError)
	this.setHttpCode(httpRespCode)
	// Add error to the request
	this.Error(err)
}

func mapErrorToHttpResponse(err errors.Error, def int) int {

	// MB-19307: please note that setting the http status
	// only works if the http header has not been sent.
	// This is the case if the whole output document is
	// smaller than the threshold beyond which the http
	// server starts sending the output with a chunked
	// transfer encoding, or the first chunk has not been
	// put together yet.
	// For this reason, be mindful that error codes mapped
	// here should only be generated at a point in which
	// the request has not produced any results (ie failed
	// in some sort of non starter way)
	switch err.Code() {
	case errors.E_SERVICE_READONLY: // readonly violation
		return http.StatusForbidden
	case errors.E_SERVICE_HTTP_UNSUPPORTED_METHOD: // unsupported http method
		return http.StatusMethodNotAllowed
	case errors.E_SERVICE_NOT_IMPLEMENTED, errors.E_SERVICE_UNRECOGNIZED_VALUE, errors.E_SERVICE_BAD_VALUE,
		errors.E_SERVICE_MISSING_VALUE, errors.E_SERVICE_MULTIPLE_VALUES, errors.E_SERVICE_UNRECOGNIZED_PARAMETER,
		errors.E_SERVICE_TYPE_MISMATCH:
		return http.StatusBadRequest
	case errors.E_SERVICE_MEDIA_TYPE:
		return http.StatusNotAcceptable
	case errors.E_SERVICE_SHUTTING_DOWN, errors.E_SERVICE_SHUT_DOWN, errors.E_SERVICE_UNAVAILABLE:
		return http.StatusServiceUnavailable
	case errors.E_SERVICE_USER_REQUEST_EXCEEDED, errors.E_SERVICE_USER_REQUEST_RATE_EXCEEDED:
		return http.StatusTooManyRequests
	case errors.E_SERVICE_USER_REQUEST_SIZE_EXCEEDED, errors.E_SERVICE_USER_RESULT_SIZE_EXCEEDED:
		return http.StatusRequestEntityTooLarge
	case errors.E_DATASTORE_INSUFFICIENT_CREDENTIALS:
		return http.StatusUnauthorized
	case errors.E_PARSE_SYNTAX: // parse error range
		return http.StatusBadRequest
	case errors.E_PLAN, errors.E_NO_SUCH_PREPARED: // plan error range
		return http.StatusNotFound
	case errors.E_INDEX_ALREADY_EXISTS:
		return http.StatusConflict
	case errors.E_INTERNAL:
		return http.StatusInternalServerError
	case errors.E_SUBQUERY_BUILD:
		return http.StatusUnprocessableEntity
	case errors.E_DATASTORE_AUTHORIZATION:
		return http.StatusUnauthorized
	case errors.E_ADMIN_AUTH:
		return http.StatusUnauthorized
	case errors.E_ADMIN_SSL_NOT_ENABLED:
		return http.StatusNotFound
	case errors.E_ADMIN_CREDS:
		return http.StatusBadRequest
	case errors.E_CB_KEYSPACE_NOT_FOUND:
		return http.StatusFailedDependency
	case errors.E_CB_BUCKET_NOT_FOUND:
		return http.StatusFailedDependency
	case errors.E_CB_NAMESPACE_NOT_FOUND:
		return http.StatusFailedDependency
	case errors.E_SERVICE_TENANT_REJECTED:
		return http.StatusTooManyRequests
	case errors.E_SERVICE_TENANT_MISSING:
		return http.StatusUnauthorized
	case errors.E_SERVICE_TENANT_NOT_AUTHORIZED:
		return http.StatusUnauthorized
	case errors.E_SERVICE_REQUEST_QUEUE_FULL:
		return http.StatusTooManyRequests
	case errors.E_SERVICE_NO_CLIENT:
		return http.StatusBadRequest
	case errors.W_GSI_TRANSIENT:
		return http.StatusOK
	default:
		return def
	}
}

func (this *httpRequest) httpCode() int {
	this.RLock()
	rv := this.httpRespCode
	this.RUnlock()
	return rv
}

func (this *httpRequest) setHttpCode(httpRespCode int) {
	this.Lock()
	this.httpRespCode = httpRespCode
	this.Unlock()
}

func (this *httpRequest) Failed(srvr *server.Server) {
	if this.httpCode() == 0 && this.GetErrorCount() == 0 {
		// we've called Failed, have not set a status and have no errors to set the status for us
		this.setHttpCode(http.StatusInternalServerError)
	}
	switch this.format {
	case XML:
		this.failedXML(srvr)
	default:
		this.failedJSON(srvr)
	}
	this.writer.noMoreData()
	this.Stop(server.FATAL)
}

func (this *httpRequest) failedJSON(srvr *server.Server) {
	prefix, indent := this.prettyStrings(srvr.Pretty(), false)
	this.writeString("{\n")
	this.writeRequestID(prefix)
	this.writeClientContextID(prefix)
	this.writeErrors(prefix, indent)
	this.writeWarnings(prefix, indent)
	this.writeState(this.State(), prefix)

	this.markTimeOfCompletion(time.Now())

	this.writeMetrics(srvr.Metrics(), prefix, indent)

	this.writeServerless(srvr.Metrics(), prefix, indent)
	this.writeProfile(srvr.Profile(), prefix, indent)
	this.writeControls(srvr.Controls(), prefix, indent)
	this.writeLog(prefix, indent)
	this.writeString("\n}\n")
}

func (this *httpRequest) failedXML(srvr *server.Server) {
	prefix, indent := this.prettyStrings(srvr.Pretty(), false)
	this.writeString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<query>")

	this.writeRequestIDXML(prefix)
	this.writeClientContextIDXML(prefix)
	this.writeErrorsXML(prefix, indent)
	this.writeWarningsXML(prefix, indent)
	this.writeStateXML(this.State(), prefix)

	this.markTimeOfCompletion(time.Now())

	this.writeMetricsXML(srvr.Metrics(), prefix, indent)

	this.writeServerlessXML(srvr.Metrics(), prefix, indent)
	this.writeProfileXML(srvr.Profile(), prefix, indent)
	this.writeControlsXML(srvr.Controls(), prefix, indent)
	this.writeLogXML(prefix, indent)
	this.writeString("\n</query>")
}

func (this *httpRequest) markTimeOfCompletion(now time.Time) {
	if this.ServiceTime().IsZero() {
		this.executionTime = time.Duration(0)
	} else {
		this.executionTime = now.Sub(this.ServiceTime())
	}
	this.elapsedTime = now.Sub(this.RequestTime())
	if !this.TransactionStartTime().IsZero() {
		this.transactionElapsedTime = now.Sub(this.TransactionStartTime())
	}
}

func (this *httpRequest) Alive() bool {
	select {
	case <-this.req.Context().Done():
		return false
	default:
		return true
	}
}

func (this *httpRequest) Execute(srvr *server.Server, context *execution.Context, reqType string, signature value.Value, startTx bool) {
	this.prefix, this.indent = this.prettyStrings(srvr.Pretty(), false)

	this.setHttpCode(http.StatusOK)
	switch this.format {
	case XML:
		this.writePrefixXML(srvr, signature, this.prefix, this.indent)
	default:
		this.writePrefix(srvr, signature, this.prefix, this.indent)
	}

	// release writer
	this.writer.releaseExternal()

	// wait for somebody to tell us we're done, or toast
	select {
	case <-this.Results():
		this.Stop(server.COMPLETED)

		// No need to wait for writer
	case <-this.StopExecute():

		// wait for operator before continuing
		this.writer.getInternal()
	case <-this.req.Context().Done():
		this.Stop(server.CLOSED)

		// wait for operator before continuing
		this.writer.getInternal()
	}

	success := this.State() == server.COMPLETED && len(this.Errors()) == 0
	if err, _ := context.DoStatementComplete(reqType, success); err != nil {
		this.Error(err)
	} else if context.TxContext() != nil && startTx {
		this.SetTransactionStartTime(context.TxContext().TxStartTime())
		this.SetTxTimeout(context.TxContext().TxTimeout())
	}
	context.Release()
	if tenant.IsServerless() {
		units := tenant.RecordCU(context.TenantCtx(), this.CpuTime(), this.UsedMemory())
		this.AddTenantUnits(tenant.QUERY_CU, units)
	}

	now := time.Now()
	this.Output().AddPhaseTime(execution.RUN, now.Sub(this.ExecTime()))
	this.markTimeOfCompletion(now)

	this.refunded = tenant.NeedRefund(context.TenantCtx(), this.Errors(), this.Warnings())
	if this.refunded {

		// TODO wait for services requests to complete
		// TODO write that we have refunded
		tenant.RefundUnits(context.TenantCtx(), this.TenantUnits())
	}
	state := this.State()
	switch this.format {
	case XML:
		this.writeSuffixXML(srvr, state, this.prefix, this.indent)
	default:
		this.writeSuffix(srvr, state, this.prefix, this.indent)
	}
	this.writer.noMoreData()
}

func (this *httpRequest) Expire(state server.State, timeout time.Duration) {
	this.Error(errors.NewTimeoutError(util.FormatDuration(timeout, this.DurationStyle())))
	this.Stop(state)
}

func (this *httpRequest) writePrefix(srvr *server.Server, signature value.Value, prefix, indent string) bool {
	return this.writeString("{\n") &&
		this.writeRequestID(prefix) &&
		this.writeClientContextID(prefix) &&
		this.writePrepared(prefix, indent) &&
		this.writeSignature(srvr.Signature(), signature, prefix, indent) &&
		this.writeString(",\n") &&
		this.writeString(prefix) &&
		this.writeString("\"results\": [")
}

func (this *httpRequest) writePrefixXML(srvr *server.Server, signature value.Value, prefix string, indent string) bool {
	return this.writeString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<query>") &&
		this.writeRequestIDXML(prefix) &&
		this.writeClientContextIDXML(prefix) &&
		this.writePreparedXML(prefix, indent) &&
		this.writeSignatureXML(srvr.Signature(), signature, prefix, indent) &&
		this.writeString("\n") &&
		this.writeString(prefix) &&
		this.writeString("<results>")
}

func (this *httpRequest) writeRequestID(prefix string) bool {
	return this.writeString(prefix) && this.writeString("\"requestID\": \"") && this.writeString(this.Id().String()) &&
		this.writeString("\"")
}

func (this *httpRequest) writeRequestIDXML(prefix string) bool {
	if prefix != "" {
		if !this.writeString("\n") || !this.writeString(prefix) {
			return false
		}
	}
	return this.writeString("<requestID>") && this.writeString(this.Id().String()) && this.writeString("</requestID>")
}

func (this *httpRequest) writeClientContextID(prefix string) bool {
	if !this.ClientID().IsValid() {
		return true
	}
	return this.writeString(",\n") && this.writeString(prefix) &&
		this.writeString("\"clientContextID\": \"") && this.writeString(this.ClientID().String()) && this.writeString("\"")
}

func (this *httpRequest) writeClientContextIDXML(prefix string) bool {
	if !this.ClientID().IsValid() {
		return true
	}
	if prefix != "" {
		if !this.writeString("\n") || !this.writeString(prefix) {
			return false
		}
	}
	return this.writeString("<clientContextID>") && this.writeString(this.ClientID().String()) &&
		this.writeString("</clientContextID>")
}

func (this *httpRequest) writePrepared(prefix, indent string) bool {
	prepared := this.Prepared()
	if this.AutoExecute() != value.TRUE || prepared == nil {
		return true
	}
	host := tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())
	name := distributed.RemoteAccess().MakeKey(host, prepared.Name())
	return this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"prepared\": \"") &&
		this.writeString(name) && this.writeString("\"")
}

func (this *httpRequest) writePreparedXML(prefix string, indent string) bool {
	prepared := this.Prepared()
	if this.AutoExecute() != value.TRUE || prepared == nil {
		return true
	}
	host := tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())
	name := distributed.RemoteAccess().MakeKey(host, prepared.Name())
	if prefix != "" {
		if !this.writeString("\n") || !this.writeString(prefix) {
			return false
		}
	}
	return this.writeString("<prepared>") && this.writeString(name) && this.writeString("</prepared>")
}

func (this *httpRequest) writeSignature(server_flag bool, signature value.Value, prefix, indent string) bool {
	s := this.Signature()
	if s == value.FALSE || (s == value.NONE && !server_flag) {
		return true
	}
	if this.SortProjection() {
		if av, ok := signature.(value.AnnotatedValue); ok {
			av.SetProjection(signature, nil)
		}
	}
	return this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"signature\": ") &&
		this.writeValue(signature, prefix, indent, true)
}

func (this *httpRequest) writeSignatureXML(server_flag bool, signature value.Value, prefix string, indent string) bool {
	s := this.Signature()
	if s == value.FALSE || (s == value.NONE && !server_flag) {
		return true
	}
	if this.SortProjection() {
		if av, ok := signature.(value.AnnotatedValue); ok {
			av.SetProjection(signature, nil)
		}
	}
	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	return (newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1])) ||
		!this.writeString("<signature>") ||
		!this.writeValue(signature, newPrefix, indent, true) ||
		(newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1])) ||
		this.writeString("</signature>")
}

func (this *httpRequest) prettyStrings(serverPretty, result bool) (string, string) {
	p := this.Pretty()
	if p == value.FALSE || (p == value.NONE && !serverPretty) {
		return NO_PRETTY_PREFIX, NO_PRETTY_INDENT
	} else if result {
		return PRETTY_RESULT_PREFIX, PRETTY_RESULT_INDENT
	} else {
		return PRETTY_PREFIX, PRETTY_INDENT
	}
}

func (this *httpRequest) SetUp() {
}

func (this *httpRequest) Result(item value.AnnotatedValue) bool {
	var success bool

	if this.writer.getExternal() {
		return false
	}
	if this.Profile() == server.ProfBench {
		this.resultCount++
		this.writer.releaseInternal()
		return true
	}

	this.writer.timeFlush()
	beforeWrites := this.writer.mark()

	var err error
	var beforeResult int

	switch this.format {
	case XML:
		beforeResult = this.writer.mark()
		order := item.ProjectionOrder()
		err = item.WriteXML(order, this.writer.buf(), this.prefix+this.indent, this.indent, item.Self())
		success = err == nil
		if this.Pretty() != value.TRUE {
			success = this.writer.write("\n")
		}
	default:
		if this.resultCount == 0 {
			success = this.writer.write("\n")
		} else {
			success = this.writer.write(",\n")
		}
		if success {
			success = this.writer.write(this.prefix)
		}
		beforeResult = this.writer.mark()

		if success {
			order := item.ProjectionOrder()
			err = item.WriteJSON(order, this.writer.buf(), this.prefix, this.indent, item.Self())
		}
	}

	if success {
		if err != nil {
			this.Error(errors.NewServiceErrorInvalidJSON(err))
			this.SetState(server.FATAL)
			success = false
		} else {
			this.resultSize += (this.writer.mark() - beforeResult)
			this.resultCount++
			if !this.writer.sizeFlush() {
				this.SetState(server.FATAL)
				success = false
			}
		}
	} else {
		this.SetState(server.CLOSED)
	}

	// did not work out: remove last writes so that we have a well formed document
	if !success {
		this.writer.truncate(beforeWrites)
	}
	this.writer.releaseInternal()
	return success
}

func (this *httpRequest) writeValue(item value.Value, prefix string, indent string, fast bool) bool {
	if item == nil {
		return this.writeString("null")
	}
	beforeWrite := this.writer.mark()
	var order []string
	if av, ok := item.(value.AnnotatedValue); ok {
		order = av.ProjectionOrder()
	}
	var err error
	switch this.format {
	case XML:
		err = item.WriteXML(order, this.writer.buf(), prefix, indent, fast)
	default:
		err = item.WriteJSON(order, this.writer.buf(), prefix, indent, fast)
	}
	if err != nil {
		this.writer.truncate(beforeWrite)
		return this.writer.printf("\"ERROR: %v\"", err)
	}
	return true
}

func (this *httpRequest) writeSuffix(srvr *server.Server, state server.State, prefix string, indent string) bool {
	return this.writeString("\n") && this.writeString(prefix) && this.writeString("]") &&
		this.writeErrors(prefix, indent) &&
		this.writeWarnings(prefix, indent) &&
		this.writeState(state, prefix) &&
		this.writeMetrics(srvr.Metrics(), prefix, indent) &&
		this.writeServerless(srvr.Metrics(), prefix, indent) &&
		this.writeProfile(srvr.Profile(), prefix, indent) &&
		this.writeControls(srvr.Controls(), prefix, indent) &&
		this.writeLog(prefix, indent) &&
		this.writeString("\n}\n")
}

func (this *httpRequest) writeSuffixXML(srvr *server.Server, state server.State, prefix string, indent string) bool {
	if (this.Pretty() == value.TRUE || this.resultCount == 0) && !this.writeString("\n") {
		return false
	}
	return this.writeString(prefix) && this.writeString("</results>") &&
		this.writeErrorsXML(prefix, indent) &&
		this.writeWarningsXML(prefix, indent) &&
		this.writeStateXML(state, prefix) &&
		this.writeMetricsXML(srvr.Metrics(), prefix, indent) &&
		this.writeServerlessXML(srvr.Metrics(), prefix, indent) &&
		this.writeProfileXML(srvr.Profile(), prefix, indent) &&
		this.writeControlsXML(srvr.Controls(), prefix, indent) &&
		this.writeLogXML(prefix, indent) &&
		this.writeString("\n</query>\n")
}

func (this *httpRequest) writeString(s string) bool {
	return this.writer.writeBytes([]byte(s))
}

func (this *httpRequest) writeStringNL(s string) bool {
	if this.Pretty() == value.TRUE {
		return this.writer.writeBytes([]byte(s)) && this.writer.writeBytes([]byte{'\n'})
	}
	return this.writer.writeBytes([]byte(s))
}

func (this *httpRequest) writeState(state server.State, prefix string) bool {
	if state == server.COMPLETED {
		if this.GetErrorCount() == 0 {
			state = server.SUCCESS
		} else {
			state = server.ERRORS
		}
	}
	return this.writeString(",\n") &&
		this.writeString(prefix) &&
		this.writeString("\"status\": \"") &&
		this.writeString(state.StateName()) &&
		this.writeString("\"")
}

func (this *httpRequest) writeStateXML(state server.State, prefix string) bool {
	if state == server.COMPLETED {
		if this.GetErrorCount() == 0 {
			state = server.SUCCESS
		} else {
			state = server.ERRORS
		}
	}
	return this.writer.printf("\n%s<status>", prefix) &&
		this.writeString(state.StateName()) &&
		this.writeString("</status>")
}

func (this *httpRequest) writeErrors(prefix string, indent string) bool {
	var err errors.Error

	if this.GetErrorCount() == 0 {
		return true
	}

	first := true
	for _, err = range this.Errors() {
		if first {
			this.writeString(",\n")
			this.writeString(prefix)
			this.writeString("\"errors\": [")

			// MB-19307: please check the comments in mapErrortoHttpResponse().
			if this.httpCode() == 0 {
				this.setHttpCode(mapErrorToHttpResponse(err, http.StatusOK))
			}
		}
		if !this.writeError(err, first, prefix, indent) {
			break
		}
		first = false
	}

	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("]")
}

func (this *httpRequest) writeErrorsXML(prefix string, indent string) bool {
	var err errors.Error

	if this.GetErrorCount() == 0 {
		return true
	}

	if !this.writer.printf("\n%s<errors>", prefix) {
		return false
	}
	first := true
	for _, err = range this.Errors() {
		// MB-19307: please check the comments in mapErrortoHttpResponse().
		if first && this.httpCode() == 0 {
			this.setHttpCode(mapErrorToHttpResponse(err, http.StatusOK))
		}
		if !this.writeErrorXML(err, prefix, indent) {
			return false
		}
		first = false
	}
	return this.writer.printf("\n%s</errors>", prefix)
}

func (this *httpRequest) writeWarnings(prefix, indent string) bool {
	var err errors.Error

	if this.GetWarningCount() == 0 {
		return true
	}

	first := true
	for _, err = range this.Warnings() {
		if first {
			this.writeString(",\n")
			this.writeString(prefix)
			this.writeString("\"warnings\": [")

			if this.httpCode() == 0 || this.httpCode() == http.StatusOK {
				this.setHttpCode(mapErrorToHttpResponse(err, http.StatusOK))
			}
		}
		if !this.writeError(err, first, prefix, indent) {
			break
		}
		first = false
	}

	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("]")
}

func (this *httpRequest) writeWarningsXML(prefix string, indent string) bool {
	var err errors.Error

	if this.GetWarningCount() == 0 {
		return true
	}

	if !this.writer.printf("\n%s<warnings>", prefix) {
		return false
	}
	first := true
	for _, err = range this.Errors() {
		if first && (this.httpCode() == 0 || this.httpCode() == http.StatusOK) {
			this.setHttpCode(mapErrorToHttpResponse(err, http.StatusOK))
		}
		if !this.writeErrorXML(err, prefix, indent) {
			return false
		}
		first = false
	}
	return this.writer.printf("\n%s</warnings>", prefix)
}

func (this *httpRequest) writeError(err errors.Error, first bool, prefix, indent string) bool {

	newPrefix := prefix + indent

	if !first && !this.writeString(",") {
		return false
	}
	if prefix != "" && !this.writeString("\n") {
		return false
	}

	m := map[string]interface{}{
		"code": err.Code(),
		"msg":  err.Error(),
	}
	retry := checkForPossibleRetry(err, this.MutationCount() != 0)
	if retry != errors.NONE {
		m["retry"] = errors.ToBool(retry)
	}
	if err.Cause() != nil {
		if !errors.IsTransactionError(err) {
			m["reason"] = err.Cause()
		} else {
			m["cause"] = err.Cause()
		}
	}
	err.ExtractLineAndColumn(m)

	var er error
	var bb bytes.Buffer
	enc := json.NewEncoder(&bb)
	enc.SetEscapeHTML(false)
	if newPrefix != "" || indent != "" {
		enc.SetIndent(newPrefix, indent)
	}
	er = enc.Encode(m)
	if er != nil {
		return false
	}

	return this.writeString(newPrefix) && this.writeString(string(bytes.TrimSuffix(bb.Bytes(), []byte{'\n'})))
}

func (this *httpRequest) writeErrorXML(err errors.Error, prefix string, indent string) bool {
	tag := "error"
	if err.IsWarning() {
		tag = "warning"
	}
	sb := &strings.Builder{}
	xml.EscapeText(sb, []byte(err.Error()))
	return this.writer.printf("\n%s%s<%s code=%d>%s</%s>", prefix, indent, tag, err.Code(), sb.String(), tag)
}

// For CAS mismatch errors where no mutations have taken place, we can explicitly set retry to true
func checkForPossibleRetry(err errors.Error, mutations bool) errors.Tristate {
	if mutations || err.Code() != errors.E_CB_DML || err.Cause() == nil {
		return err.Retry()
	}
	if c, ok := err.Cause().(errors.Error); ok && c.Code() == errors.E_CAS_MISMATCH {
		return errors.TRUE
	}
	return err.Retry()
}

func (this *httpRequest) writeMetrics(metrics bool, prefix, indent string) bool {
	m := this.Metrics()
	if m == value.FALSE || (m == value.NONE && !metrics) {
		return true
	}

	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}

	var b [64]byte
	beforeMetrics := this.writer.mark()
	if !(this.writeString(",\n") &&
		this.writeString(prefix) &&
		this.writeString("\"metrics\": {") &&

		this.writeString(newPrefix) &&
		this.writeString("\"elapsedTime\": \"") &&
		this.writeString(util.FormatDuration(this.elapsedTime, this.DurationStyle())) &&
		this.writeString("\",") &&
		this.writeString(newPrefix) &&
		this.writeString("\"executionTime\": \"") &&
		this.writeString(util.FormatDuration(this.executionTime, this.DurationStyle())) &&
		this.writeString("\",") &&
		this.writeString(newPrefix) &&
		this.writeString("\"resultCount\": ") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(this.resultCount), 10)) &&
		this.writeString(",") &&
		this.writeString(newPrefix) &&
		this.writeString("\"resultSize\": ") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(this.resultSize), 10)) &&
		this.writeString(",") &&
		this.writeString(newPrefix) &&
		this.writeString("\"serviceLoad\": ") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(server.ActiveRequestsLoad()), 10))) {

		this.writer.truncate(beforeMetrics)
		return false
	}

	buf := this.writer.buf()
	if this.UsedMemory() > 0 {
		fmt.Fprintf(buf, ",%s\"usedMemory\": %d", newPrefix, this.UsedMemory())
	}

	if this.MutationCount() > 0 {
		fmt.Fprintf(buf, ",%s\"mutationCount\": %d", newPrefix, this.MutationCount())
	}

	if this.transactionElapsedTime > 0 {
		fmt.Fprintf(buf, ",%s\"transactionElapsedTime\": \"%v\"", newPrefix,
			util.FormatDuration(this.transactionElapsedTime, this.DurationStyle()))
	}

	if transactionRemainingTime := this.TransactionRemainingTime(); transactionRemainingTime != "" {
		fmt.Fprintf(buf, ",%s\"transactionRemainingTime\": \"%s\"", newPrefix, transactionRemainingTime)
	}

	if this.SortCount() > 0 {
		fmt.Fprintf(buf, ",%s\"sortCount\": %d", newPrefix, this.SortCount())
	}

	if this.GetErrorCount() > 0 {
		fmt.Fprintf(buf, ",%s\"errorCount\": %d", newPrefix, this.GetErrorCount())
	}

	if this.GetWarningCount() > 0 {
		fmt.Fprintf(buf, ",%s\"warningCount\": %d", newPrefix, this.GetWarningCount())
	}

	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		this.writer.truncate(beforeMetrics)
		return false
	}
	return this.writeString("}")
}

func (this *httpRequest) writeMetricsXML(metrics bool, prefix string, indent string) bool {
	m := this.Metrics()
	if m == value.FALSE || (m == value.NONE && !metrics) {
		return true
	}

	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	var b [64]byte
	beforeMetrics := this.writer.mark()
	if newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1]) {
		return false
	}
	if !(this.writeString("<metrics>") &&
		this.writeString(newPrefix) &&
		this.writeString("<elapsedTime>") &&
		this.writeString(util.FormatDuration(this.elapsedTime, this.DurationStyle())) &&
		this.writeString("</elapsedTime>") &&
		this.writeString(newPrefix) &&
		this.writeString("<executionTime>") &&
		this.writeString(util.FormatDuration(this.executionTime, this.DurationStyle())) &&
		this.writeString("</executionTime>") &&
		this.writeString(newPrefix) &&
		this.writeString("<resultCount>") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(this.resultCount), 10)) &&
		this.writeString("</resultCount>") &&
		this.writeString(newPrefix) &&
		this.writeString("<resultSize>") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(this.resultSize), 10)) &&
		this.writeString("</resultSize>") &&
		this.writeString(newPrefix) &&
		this.writeString("<serviceLoad>") &&
		this.writer.writeBytes(strconv.AppendInt(b[:0], int64(server.ActiveRequestsLoad()), 10)) &&
		this.writeString("</serviceLoad>")) {

		this.writer.truncate(beforeMetrics)
		return false
	}

	buf := this.writer.buf()
	if this.UsedMemory() > 0 {
		fmt.Fprintf(buf, "%s<usedMemory>%d</usedMemory>", newPrefix, this.UsedMemory())
	}

	if this.MutationCount() > 0 {
		fmt.Fprintf(buf, "%s<mutationCount>%d</mutationCount>", newPrefix, this.MutationCount())
	}

	if this.transactionElapsedTime > 0 {
		fmt.Fprintf(buf, "%s<transactionElapsedTime>%s</transactionElapsedTime>", newPrefix,
			util.FormatDuration(this.transactionElapsedTime, this.DurationStyle()))
	}

	if transactionRemainingTime := this.TransactionRemainingTime(); transactionRemainingTime != "" {
		fmt.Fprintf(buf, "%s<transactionRemainingTime>%s</transactionRemainingTime>", newPrefix, transactionRemainingTime)
	}

	if this.SortCount() > 0 {
		fmt.Fprintf(buf, "%s<sortCount>%d</sortCount>", newPrefix, this.SortCount())
	}

	if this.GetErrorCount() > 0 {
		fmt.Fprintf(buf, "%s<errorCount>%d</errorCount>", newPrefix, this.GetErrorCount())
	}

	if this.GetWarningCount() > 0 {
		fmt.Fprintf(buf, "%s<warningCount>%d</warningCount>", newPrefix, this.GetWarningCount())
	}

	if newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1]) {
		return false
	}
	return this.writeString("</metrics>")
}

func (this *httpRequest) writeControls(controls bool, prefix, indent string) bool {
	var newPrefix string
	var e []byte
	var err error
	var bytes []byte

	needComma := false
	c := this.Controls()
	if c == value.FALSE || (c == value.NONE && !controls) {
		return true
	}

	namedArgs := this.NamedArgs()
	positionalArgs := this.PositionalArgs()

	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	rv := this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"controls\": {")
	if !rv {
		return false
	}
	if namedArgs != nil {
		if indent != "" {
			e, err = json.MarshalIndent(namedArgs, "\t", indent)
		} else {
			e, err = json.Marshal(namedArgs)
		}
		if err != nil || !this.writer.printf("%s\"namedArgs\": %s", newPrefix, e) {
			logging.Infof("Error writing namedArgs. Error: %v", err)
		}
		needComma = true
	}
	if positionalArgs != nil {
		if needComma && !this.writeString(",") {
			return false
		}
		if indent != "" {
			e, err = json.MarshalIndent(positionalArgs, "\t", indent)
		} else {
			e, err = json.Marshal(positionalArgs)
		}
		if err != nil || !this.writer.printf("%s\"positionalArgs\": %s", newPrefix, e) {
			logging.Infof("Error writing positionalArgs. Error: %v", err)
		}
		needComma = true
	}

	if needComma && !this.writeString(",") {
		return false
	}
	if err != nil || !this.writer.printf("%s\"scan_consistency\": \"%s\"", newPrefix, string(this.ScanConsistency())) {
		logging.Infof("Error writing scan_consistency. Error: %v", err)
	}
	needComma = true

	if this.QueryContext() != "" {
		if needComma && !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"queryContext\": \"%s\"", newPrefix, this.QueryContext()) {
			logging.Infof("Error writing queryContext. Error: %v", err)
		}
	}

	if this.UseFts() {
		if !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"use_fts\": \"%v\"", newPrefix, this.UseFts()) {
			logging.Infof("Error writing use_fts. Error: %v", err)
		}
	}

	if this.UseCBO() {
		if !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"use_cbo\": \"%v\"", newPrefix, this.UseCBO()) {
			logging.Infof("Error writing use_cbo. Error: %v", err)
		}
	}

	if this.UseReplica() == value.TRUE {
		if !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"use_replica\": \"%v\"", newPrefix, value.TristateToString(this.UseReplica())) {
			logging.Infof("Error writing use_replica. Error: %v", err)
		}
	}

	memoryQuota := this.MemoryQuota()
	if memoryQuota != 0 {
		if !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"memoryQuota\": \"%v\"", newPrefix, memoryQuota) {
			logging.Infof("Error writing memoryQuota. Error: %v", err)
		}
	}

	if !this.writeString(",") || !this.writer.printf("%s\"n1ql_feat_ctrl\": \"%#x\"", newPrefix, this.FeatureControls()) {
		logging.Infof("Error writing n1ql_feat_ctrl")
	}

	// Disabled features in n1ql_feat_ctrl bitset
	if len(newPrefix) > 1 {
		bytes, err = json.MarshalIndent(util.DisabledFeatures(this.FeatureControls()), newPrefix[1:], indent)
	} else {
		bytes, err = json.MarshalIndent(util.DisabledFeatures(this.FeatureControls()), newPrefix, indent)
	}
	if err != nil || !this.writeString(",") || !this.writer.printf("%s\"disabledFeatures\":", newPrefix) ||
		!this.writer.writeBytes(bytes) {

		logging.Infof("Error writing disabledFeatures")
	}

	if !this.writeString(",") || !this.writer.printf("%s\"stmtType\": \"%v\"", newPrefix, this.Type()) {
		logging.Infof("Error writing stmtType")
	}

	this.writeTransactionInfo(newPrefix, indent)

	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("}")
}

func (this *httpRequest) writeControlsXML(controls bool, prefix string, indent string) bool {
	var err error

	c := this.Controls()
	if c == value.FALSE || (c == value.NONE && !controls) {
		return true
	}

	namedArgs := this.NamedArgs()
	positionalArgs := this.PositionalArgs()

	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}

	if (newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1])) || !this.writeString("<controls>") {
		return false
	}
	if namedArgs != nil {
		if !this.writeString(newPrefix) || !this.writeString("<namedArgs>") {
			return false
		}
		val := value.NewValue(namedArgs)
		err = val.WriteXML(nil, this.writer.buf(), newPrefix, indent, false)
		if err != nil {
			logging.Infof("Error writing namedArgs. Error: %v", err)
		}
		if !this.writeString(newPrefix) || !this.writeString("</namedArgs>") {
			return false
		}
	}
	if positionalArgs != nil {
		if !this.writeString(newPrefix) || !this.writeString("<positionalArgs>") {
			return false
		}
		val := value.NewValue(positionalArgs)
		err = val.WriteXML(nil, this.writer.buf(), newPrefix, indent, false)
		if err != nil {
			logging.Infof("Error writing positionalArgs. Error: %v", err)
		}
		if !this.writeString(newPrefix) || !this.writeString("</positionalArgs>") {
			return false
		}
	}

	if err != nil || !this.writer.printf("%s<scan_consistency>%s</scan_consistency>", newPrefix,
		string(this.ScanConsistency())) {

		logging.Infof("Error writing scan_consistency. Error: %v", err)
	}

	if this.QueryContext() != "" {
		if err != nil || !this.writer.printf("%s<queryContext>%s</queryContext>", newPrefix, this.QueryContext()) {
			logging.Infof("Error writing queryContext. Error: %v", err)
		}
	}

	if this.UseFts() {
		if err != nil || !this.writer.printf("%s<use_fts>%v</use_fts>", newPrefix, this.UseFts()) {
			logging.Infof("Error writing use_fts. Error: %v", err)
		}
	}

	if this.UseCBO() {
		if err != nil || !this.writer.printf("%s<use_cbo>%v</use_cbo>", newPrefix, this.UseCBO()) {
			logging.Infof("Error writing use_cbo. Error: %v", err)
		}
	}

	if this.UseReplica() == value.TRUE {
		if err != nil || !this.writer.printf("%s<use_replica>%v</use_replica>", newPrefix,
			value.TristateToString(this.UseReplica())) {

			logging.Infof("Error writing use_replica. Error: %v", err)
		}
	}

	memoryQuota := this.MemoryQuota()
	if memoryQuota != 0 {
		if err != nil || !this.writer.printf("%s<memoryQuota>%v</memoryQuota>", newPrefix, memoryQuota) {
			logging.Infof("Error writing memoryQuota. Error: %v", err)
		}
	}

	if !this.writer.printf("%s<n1ql_feat_ctrl>%#x</n1ql_feat_ctrl>", newPrefix, this.FeatureControls()) {
		logging.Infof("Error writing n1ql_feat_ctrl")
	}

	// Disabled features in n1ql_feat_ctrl bitset
	if !this.writer.printf("%s<disabledFeatures>", newPrefix) {
		logging.Infof("Error writing disabledFeatures")
	} else {
		df := util.DisabledFeatures(this.FeatureControls())
		for i := range df {
			val := value.NewValue(df[i])
			err = val.WriteXML(nil, this.writer.buf(), newPrefix+indent, indent, false)
			if err != nil {
				logging.Infof("Error writing disabledFeatures. Error: %v", err)
				break
			}
		}
		if !this.writer.printf("%s</disabledFeatures>", newPrefix) {
			logging.Infof("Error writing disabledFeatures")
		}
	}

	if !this.writer.printf("%s<stmtType>%v</stmtType>", newPrefix, this.Type()) {
		logging.Infof("Error writing stmtType")
	}

	this.writeTransactionInfoXML(newPrefix, indent)

	if newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1]) {
		return false
	}
	return this.writeString("</controls>")
}

func (this *httpRequest) writeTransactionInfo(prefix, indent string) bool {
	if this.TxId() != "" {
		if !this.writer.printf(",%s\"txid\": \"%v\"", prefix, this.TxId()) {
			logging.Infof("Error writing txid")
		}
		if !this.writer.printf(",%s\"tximplicit\": \"%v\"", prefix, this.TxImplicit()) {
			logging.Infof("Error writing tximplicit")
		}
		if !this.writer.printf(",%s\"txstmtnum\": \"%v\"", prefix, this.TxStmtNum()) {
			logging.Infof("Error writing stmtnum")
		}
		if !this.writer.printf(",%s\"txtimeout\": \"%v\"", prefix, util.FormatDuration(this.TxTimeout(), this.DurationStyle())) {
			logging.Infof("Error writing txtimeout")
		}
		if !this.writer.printf(",%s\"durability_level\": \"%v\"",
			prefix, datastore.DurabilityLevelToName(this.DurabilityLevel())) {
			logging.Infof("Error writing durability_level")
		}
		if !this.writer.printf(",%s\"durability_timeout\": \"%v\"", prefix,
			util.FormatDuration(this.DurabilityTimeout(), this.DurationStyle())) {

			logging.Infof("Error writing durability_timeout")
		}
	}
	return true
}

func (this *httpRequest) writeTransactionInfoXML(prefix string, indent string) bool {
	if this.TxId() != "" {
		if !this.writer.printf("%s<txid>%v<txid>", prefix, this.TxId()) {
			logging.Infof("Error writing txid")
		}
		if !this.writer.printf("%s<tximplicit>%v<tximplicit>", prefix, this.TxImplicit()) {
			logging.Infof("Error writing tximplicit")
		}
		if !this.writer.printf("%s<txstmtnum>%v</txstmtnum>", prefix, this.TxStmtNum()) {
			logging.Infof("Error writing stmtnum")
		}
		if !this.writer.printf("%s<txtimeout>%v</txtimeout>", prefix, util.FormatDuration(this.TxTimeout(), this.DurationStyle())) {
			logging.Infof("Error writing txtimeout")
		}
		if !this.writer.printf("%s<durability_level>%v</durability_level>", prefix,
			datastore.DurabilityLevelToName(this.DurabilityLevel())) {

			logging.Infof("Error writing durability_level")
		}
		if !this.writer.printf("%s<durability_timeout>%v</durability_timeout>", prefix,
			util.FormatDuration(this.DurabilityTimeout(), this.DurationStyle())) {

			logging.Infof("Error writing durability_timeout")
		}
	}
	return true
}

func (this *httpRequest) writeServerless(metrics bool, prefix, indent string) bool {
	if !tenant.IsServerless() {
		return true
	}
	m := this.Metrics()
	if m == value.FALSE || (m == value.NONE && !metrics) {
		return true
	}

	v := tenant.Units2Map(this.TenantUnits())
	if len(v) == 0 {
		return true
	}

	var bytes []byte
	var err error
	if indent == "" {
		bytes, err = json.Marshal(v)
	} else {
		bytes, err = json.MarshalIndent(v, prefix, indent)
	}
	if err != nil {
		return false
	}

	beforeUnits := this.writer.mark()
	if this.ThrottleTime() > time.Duration(0) &&
		!(this.writeString(",\n") &&
			this.writeString(prefix) &&
			this.writeString("\"throttleTime\": \"") &&
			this.writeString(util.FormatDuration(this.ThrottleTime(), this.DurationStyle())) &&
			this.writeString("\"")) {
		this.writer.truncate(beforeUnits)
		return false
	}
	if !(this.writeString(",\n") &&
		this.writeString(prefix) &&
		this.writeString("\"billingUnits\": ")) {
		this.writer.truncate(beforeUnits)
		return false
	}
	if !this.writer.writeBytes(bytes) {
		this.writer.truncate(beforeUnits)
		return false
	}
	if !this.refunded {
		return true
	}
	if !(this.writeString(",\n") &&
		this.writeString(prefix) &&
		this.writeString("\"refundedUnits\": ")) {
		this.writer.truncate(beforeUnits)
		return false
	}
	if !this.writer.writeBytes(bytes) {
		this.writer.truncate(beforeUnits)
		return false
	}
	return true
}

func (this *httpRequest) writeServerlessXML(metrics bool, prefix string, indent string) bool {
	if !tenant.IsServerless() {
		return true
	}
	m := this.Metrics()
	if m == value.FALSE || (m == value.NONE && !metrics) {
		return true
	}

	um := tenant.Units2Map(this.TenantUnits())
	if len(um) == 0 {
		return true
	}

	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix
	}

	beforeUnits := this.writer.mark()
	if this.ThrottleTime() > time.Duration(0) &&
		!this.writer.printf("%s<throttleTime>%s</throttleTime>", newPrefix,
			util.FormatDuration(this.ThrottleTime(), this.DurationStyle())) {

		this.writer.truncate(beforeUnits)
		return false
	}
	if !this.writeString(newPrefix) || !this.writeString("<billingUnits>") {
		this.writer.truncate(beforeUnits)
		return false
	}
	b, err := json.Marshal(um)
	if err != nil {
		this.writer.truncate(beforeUnits)
		return false
	}
	mi := make(map[string]interface{})
	err = json.Unmarshal(b, &mi)
	if err != nil {
		this.writer.truncate(beforeUnits)
		return false
	}
	val := value.NewValue(m)
	err = val.WriteXML(nil, this.writer.buf(), newPrefix+indent, indent, false)
	if err != nil || !this.writeString(newPrefix) || !this.writeString("</billingUnits>") {
		this.writer.truncate(beforeUnits)
		return false
	}
	if !this.refunded {
		return true
	}
	if !this.writeString(newPrefix) || !this.writeString("<redundedUnits>") {
		this.writer.truncate(beforeUnits)
		return false
	}
	err = val.WriteXML(nil, this.writer.buf(), newPrefix+indent, indent, false)
	if err != nil || !this.writeString(newPrefix) || !this.writeString("</redundedUnits>") {
		this.writer.truncate(beforeUnits)
		return false
	}
	return true
}

func (this *httpRequest) writeProfile(profile server.Profile, prefix, indent string) bool {
	var newPrefix string
	var e []byte
	var err error

	needComma := false
	p := this.Profile()
	if p == server.ProfUnset {
		p = profile
	}
	if p == server.ProfOff {
		return true
	}

	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	if !this.writeString(",\n") || !this.writeString(prefix) || !this.writeString("\"profile\": {") {
		return false
	}
	if p != server.ProfOff {
		phaseTimes := this.RawPhaseTimes()
		if phaseTimes != nil {
			for k, v := range phaseTimes {
				phaseTimes[k] = util.FormatDuration(v.(time.Duration), this.DurationStyle())
			}
			if indent != "" {
				e, err = json.MarshalIndent(phaseTimes, "\t", indent)
			} else {
				e, err = json.Marshal(phaseTimes)
			}
			if err != nil || !this.writer.printf("%s\"phaseTimes\": %s", newPrefix, e) {
				logging.Infof("Error writing phaseTimes: %v", err)
			}
			needComma = true
		}
		phaseCounts := this.FmtPhaseCounts()
		if phaseCounts != nil {
			if needComma && !this.writeString(",") {
				return false
			}
			if indent != "" {
				e, err = json.MarshalIndent(phaseCounts, "\t", indent)
			} else {
				e, err = json.Marshal(phaseCounts)
			}
			if err != nil || !this.writer.printf("%s\"phaseCounts\": %s", newPrefix, e) {
				logging.Infof("Error writing phaseCounts: %v", err)
			}
			needComma = true
		}
		phaseOperators := this.FmtPhaseOperators()
		if phaseOperators != nil {
			if needComma && !this.writeString(",") {
				return false
			}
			if indent != "" {
				e, err = json.MarshalIndent(phaseOperators, "\t", indent)
			} else {
				e, err = json.Marshal(phaseOperators)
			}
			if err != nil || !this.writer.printf("%s\"phaseOperators\": %s", newPrefix, e) {
				logging.Infof("Error writing phaseOperators: %v", err)
			}
			needComma = true
		}

		if needComma && !this.writeString(",") {
			return false
		}
		if this.CpuTime() > time.Duration(0) &&
			!this.writer.printf("%s\"cpuTime\": \"%s\",", newPrefix, util.FormatDuration(this.CpuTime(), this.DurationStyle())) {

			logging.Infof("Error writing request CPU time")
		}
		if !this.writer.printf("%s\"requestTime\": \"%s\"", newPrefix, this.RequestTime().Format(expression.DEFAULT_FORMAT)) {
			logging.Infof("Error writing request time")
		}
		if !this.writer.printf(",%s\"servicingHost\": \"%s\"", newPrefix,
			tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())) {

			logging.Infof("Error writing servicing host")
		}
		needComma = true
	}
	if p == server.ProfOn || p == server.ProfBench {
		timings := this.GetTimings()
		if timings != nil {
			e, err = json.Marshal(timings)
			if err != nil {
				logging.Infof("Error writing executionTimings: %v", err)
			} else {
				v := value.ApplyDurationStyleToValue(this.DurationStyle(), func(s string) bool {
					return strings.HasSuffix(s, "Time")
				}, value.NewValue(e))
				if indent != "" {
					e, err = json.MarshalIndent(v, "\t", indent)
				} else {
					e, err = json.Marshal(v)
				}
				if err != nil || !this.writer.printf(",%s\"executionTimings\": %s", newPrefix, e) {
					logging.Infof("Error writing executionTimings: %v", err)
				}
			}
			this.SetFmtTimings(e)
			optEstimates := this.FmtOptimizerEstimates(timings)
			if optEstimates != nil {
				if indent != "" {
					e, err = json.MarshalIndent(optEstimates, "\t", indent)
				} else {
					e, err = json.Marshal(optEstimates)
				}
				if err != nil || !this.writer.printf(",%s\"optimizerEstimates\": %s", newPrefix, e) {
					logging.Infof("Error writing optimizerEstimates: %v", err)
				}
			}
			this.SetFmtOptimizerEstimates(optEstimates)
		}
	}
	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("}")
}

func (this *httpRequest) writeProfileXML(profile server.Profile, prefix string, indent string) bool {
	var e []byte
	var err error

	p := this.Profile()
	if p == server.ProfUnset {
		p = profile
	}
	if p == server.ProfOff {
		return true
	}

	var newPrefix string
	var newValuePrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
		newValuePrefix = newPrefix + indent
	}
	if (newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1])) || !this.writeString("<profile>") {
		return false
	}
	if p != server.ProfOff {
		phaseTimes := this.RawPhaseTimes()
		if phaseTimes != nil {
			if !this.writeString(newPrefix) || !this.writeString("<phaseTimes>") {
				logging.Infof("Error writing phaseTimes: %v", err)
			} else {
				for k, v := range phaseTimes {
					phaseTimes[k] = util.FormatDuration(v.(time.Duration), this.DurationStyle())
				}
				v := value.NewValue(phaseTimes)
				err = v.WriteXML(nil, this.writer.buf(), newValuePrefix, indent, false)
				if err != nil || (newPrefix != "" && !this.writeString(newPrefix)) || !this.writeString("</phaseTimes>") {
					logging.Infof("Error writing phaseTimes: %v", err)
				}
			}
		}
		phaseCounts := this.FmtPhaseCounts()
		if phaseCounts != nil {
			if !this.writeString(newPrefix) || !this.writeString("<phaseCounts>") {
				logging.Infof("Error writing phaseCounts: %v", err)
			} else {
				v := value.NewValue(phaseCounts)
				err = v.WriteXML(nil, this.writer.buf(), newValuePrefix, indent, false)
				if err != nil || (newPrefix != "" && !this.writeString(newPrefix)) || !this.writeString("</phaseCounts>") {
					logging.Infof("Error writing phaseCounts: %v", err)
				}
			}
		}
		phaseOperators := this.FmtPhaseOperators()
		if phaseOperators != nil {
			if !this.writeString(newPrefix) || !this.writeString("<phaseOperators>") {
				logging.Infof("Error writing phaseOperators: %v", err)
			} else {
				v := value.NewValue(phaseOperators)
				err = v.WriteXML(nil, this.writer.buf(), newValuePrefix, indent, false)
				if err != nil || (newPrefix != "" && !this.writeString(newPrefix)) || !this.writeString("</phaseOperators>") {
					logging.Infof("Error writing phaseOperators: %v", err)
				}
			}
		}

		if this.CpuTime() > time.Duration(0) &&
			(!this.writeString(newPrefix) ||
				!this.writer.printf("<cpuTime>%s</cpuTime>", util.FormatDuration(this.CpuTime(), this.DurationStyle()))) {

			logging.Infof("Error writing request CPU time")
		}
		if !this.writeString(newPrefix) ||
			!this.writer.printf("<requestTime>%s</requestTime>", this.RequestTime().Format(expression.DEFAULT_FORMAT)) {

			logging.Infof("Error writing request time")
		}
		if !this.writeString(newPrefix) ||
			!this.writer.printf("<servicingHost>%s</servicingHost>", tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())) {

			logging.Infof("Error writing servicing host")
		}
	}
	if p == server.ProfOn || p == server.ProfBench {
		timings := this.GetTimings()
		if timings != nil {
			e, err = json.Marshal(timings)
			if err != nil {
				logging.Infof("Error writing executionTimings: %v", err)
			} else {
				m := make(map[string]interface{})
				err = json.Unmarshal(e, &m)
				if err != nil || !this.writeString(newPrefix) || !this.writeString("<executionTimings>") {
					logging.Infof("Error writing executionTimings: %v", err)
				} else {
					v := value.ApplyDurationStyleToValue(this.DurationStyle(), func(s string) bool {
						return strings.HasSuffix(s, "Time")
					}, value.NewValue(m))
					err = v.WriteXML(nil, this.writer.buf(), newValuePrefix, indent, false)
					if err != nil || !this.writeString(newPrefix) || !this.writeString("</executionTimings>") {
						logging.Infof("Error writing executionTimings: %v", err)
					}
				}
				this.SetFmtTimings(e)
			}
			optEstimates := this.FmtOptimizerEstimates(timings)
			if optEstimates != nil {
				if !this.writeString(newPrefix) || !this.writeString("<optimizerEstimates>") {
					logging.Infof("Error writing optimizerEstimates: %v", err)
				} else {
					v := value.NewValue(optEstimates)
					err = v.WriteXML(nil, this.writer.buf(), newValuePrefix, indent, false)
					if err != nil || !this.writeString(newPrefix) || !this.writeString("</optimizerEstimates>") {
						logging.Infof("Error writing optimizerEstimates: %v", err)
					}
				}
			}
			this.SetFmtOptimizerEstimates(optEstimates)
		}
	}
	if newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1]) {
		return false
	}
	return this.writeString("</profile>")
}

func (this *httpRequest) Loga(l logging.Level, f func() string) {
	if this.logger == nil || this.logger.Level() < l {
		return
	}
	this.logger.Loga(l, f)
}

func (this *httpRequest) Logf(l logging.Level, f string, args ...interface{}) {
	if this.logger == nil || this.logger.Level() < l {
		return
	}
	this.logger.Loga(l, func() string { return fmt.Sprintf(f, args...) })
}

func (this *httpRequest) writeLog(prefix, indent string) bool {
	if this.logger == nil || this.logger.Level() == logging.NONE {
		return true
	}
	logger, ok := this.logger.(logging.RequestLogger)
	if !ok {
		return true
	}
	if !this.writeString(",\n") || !this.writeString(prefix) || !this.writeString("\"log\": [") {
		return false
	}
	var newPrefix string
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	first := true
	ok = logger.Foreach(func(text string) bool {
		if !first && !this.writeString(",") {
			return false
		}
		b, _ := json.Marshal(text)
		if !this.writeString(newPrefix) || !this.writeString(string(b)) {
			return false
		}
		first = false
		return true
	})
	logger.Close()
	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("]")
}

func (this *httpRequest) writeLogXML(prefix string, indent string) bool {
	if this.logger == nil || this.logger.Level() == logging.NONE {
		return true
	}
	logger, ok := this.logger.(logging.RequestLogger)
	if !ok {
		return true
	}
	newPrefix := "\n"
	if prefix != "" {
		newPrefix = "\n" + prefix + indent
	}
	if (newPrefix != "" && !this.writeString(newPrefix[:len(prefix)+1])) || !this.writeString("<log>") {
		return false
	}
	ok = logger.Foreach(func(text string) bool {
		if !this.writeString(newPrefix) || !this.writeString("<string>") {
			return false
		}
		err := xml.EscapeText(this.writer.buf(), []byte(text))
		if err != nil || !this.writeString("</string>") {
			return false
		}
		return true
	})
	logger.Close()
	if !this.writeString(newPrefix[:len(prefix)+1]) {
		return false
	}
	return this.writeString("</log>")
}

// the buffered writer writes the response data in chunks
// note that the access to the buffered writer is not controlled,
// and the executor and stream have to coordinate in between them
// not to mess up the output
type bufferedWriter struct {
	req         *httpRequest  // the request for the response we are writing
	buffer      *bytes.Buffer // buffer for writing response data to
	buffer_pool BufferPool    // buffer manager for our buffers
	closed      bool
	header      bool // headers required
	lastFlush   util.Time
	started     bool
	stopped     bool
	inUse       bool
	waiter      bool
	cond        sync.Cond
	lock        sync.Mutex

	ioTimeout time.Duration
	ioRemTime time.Duration
	monElem   *list.Element
}

const _PRINTF_THRESHOLD = 128

func NewBufferedWriter(w *bufferedWriter, r *httpRequest, bp BufferPool) {
	w.ioRemTime = 0
	w.req = r
	w.buffer = bp.GetBuffer()
	w.buffer_pool = bp
	w.closed = false
	w.header = true
	w.lastFlush = util.Now()
	w.started = false
	w.stopped = false
	w.inUse = false
	w.waiter = false
	w.cond.L = &w.lock

	w.ioTimeout = r.Timeout()
	if w.ioTimeout <= 0 {
		w.ioTimeout = _DEF_IO_WRITE_TIME_LIMIT
	} else if w.ioTimeout < _MIN_IO_WRITE_TIME_LIMIT {
		w.ioTimeout = _MIN_IO_WRITE_TIME_LIMIT
	}
}

func (this *bufferedWriter) getExternal() bool {
	this.lock.Lock()

	// if the writer is no longer available, return control to the operator, so that
	// it can cleanup after itself
	if this.stopped {
		this.lock.Unlock()
		return true
	}

	// wait until ServeHTTP() has stopped using the writer
	if !this.started {
		this.waiter = true
		this.cond.Wait()
	}
	this.inUse = true
	this.lock.Unlock()
	return false
}

func (this *bufferedWriter) releaseExternal() {
	this.lock.Lock()
	this.started = true
	if this.waiter {
		this.waiter = false
		this.lock.Unlock()
		this.cond.Signal()
	} else {
		this.lock.Unlock()
	}
}

func (this *bufferedWriter) getInternal() {
	this.lock.Lock()
	if this.inUse {
		this.waiter = true
		this.cond.Wait()
	}
	this.stopped = true
	this.lock.Unlock()
}

func (this *bufferedWriter) releaseInternal() {
	this.lock.Lock()
	this.inUse = false
	if this.waiter {
		this.waiter = false
		this.lock.Unlock()
		this.cond.Signal()
	} else {
		this.lock.Unlock()
	}
}

func (this *bufferedWriter) writeBytes(s []byte) bool {
	if this.closed {
		return false
	}
	if len(s) == 0 {
		return true
	}

	// threshold exceeded
	if len(s)+this.buffer.Len() > this.buffer_pool.BufferCapacity() {
		w := this.req.resp // our request's response writer

		// write response header and data buffered so far using request's response writer:
		if this.header {
			if this.req.httpCode() == 0 {
				this.req.setHttpCode(http.StatusOK)
			}
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		if !this.copyWithTimeout(w, this.buffer) {
			return false
		}
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}

	// under threshold - write the string to our buffer
	_, err := this.buffer.Write([]byte(s))
	return err == nil
}

func (this *bufferedWriter) printf(f string, args ...interface{}) bool {
	if this.closed {
		return false
	}

	// threshold exceeded
	if _PRINTF_THRESHOLD+this.buffer.Len() > this.buffer_pool.BufferCapacity() {
		w := this.req.resp // our request's response writer

		// write response header and data buffered so far using request's response writer:
		if this.header {
			if this.req.httpCode() == 0 {
				this.req.setHttpCode(http.StatusOK)
			}
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		if !this.copyWithTimeout(w, this.buffer) {
			return false
		}
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}

	// under threshold - write the string to our buffer
	_, err := fmt.Fprintf(this.buffer, f, args...)
	return err == nil
}

// these are only used by Result() handling
// fast write
func (this *bufferedWriter) write(s string) bool {
	_, err := this.buffer.Write([]byte(s))
	return err == nil
}

// flush in a timely manner
func (this *bufferedWriter) timeFlush() {

	// time flushing only happens after we have sent the first buffer
	if this.closed || this.header {
		return
	}

	// flush only if time has exceeded
	if util.Since(this.lastFlush) > 100*time.Millisecond {
		w := this.req.resp // our request's response writer

		// write response header and data buffered so far using request's response writer:
		if this.header {
			if this.req.httpCode() == 0 {
				this.req.setHttpCode(http.StatusOK)
			}
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		if !this.copyWithTimeout(w, this.buffer) {
			return
		}
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}
}

// flush on a full buffer
func (this *bufferedWriter) sizeFlush() bool {
	if this.closed {
		return true
	}

	// beyond capacity
	if this.buffer.Len() > this.buffer_pool.BufferCapacity() {
		w := this.req.resp // our request's response writer

		// write response header and data buffered so far using request's response writer:
		if this.header {
			if this.req.httpCode() == 0 {
				this.req.setHttpCode(http.StatusOK)
			}
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		if !this.copyWithTimeout(w, this.buffer) {
			return false
		}
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}
	return true
}

// mark the current write position
func (this *bufferedWriter) mark() int {
	return this.buffer.Len()
}

func (this *bufferedWriter) truncate(mark int) {
	if !this.closed {
		this.buffer.Truncate(mark)
	}
}

func (this bufferedWriter) buf() io.Writer {
	return this.buffer
}

// empty and dispose of writer
func (this *bufferedWriter) noMoreData() {
	if this.closed {
		return
	}

	w := this.req.resp // our request's response writer
	r := this.req.req  // our request's http request

	if this.header {
		// calculate and set the Content-Length header:
		content_len := strconv.Itoa(len(this.buffer.Bytes()))
		w.Header().Set("Content-Length", content_len)
		// write response header and data buffered so far:
		if this.req.httpCode() == 0 {
			this.req.setHttpCode(http.StatusOK)
		}
		w.WriteHeader(this.req.httpCode())
		this.header = false
	}

	this.copyWithTimeout(w, this.buffer)
	this.buffer_pool.PutBuffer(this.buffer)
	r.Body.Close()
	this.closed = true
}

/*
This is to avoid malicious attacks from clients that deliberately do not read from their connection.
Connections are monitored and forcibly closed when we detect they've stalled.
go-1.19: If there was a simple write with timeout or an individual request write operation deadline API we could use it instead.
*/
func (this *bufferedWriter) copyWithTimeout(w io.Writer, s io.Reader) bool {
	this.ioRemTime = this.ioTimeout // deliberately not atomic
	io.Copy(w, s)
	this.ioRemTime = 0
	return !this.closed
}

func (this *bufferedWriter) cancelIO() {
	if this.ioRemTime >= 0 || this.closed || this.req == nil || this.req.resp == nil {
		return
	}
	this.closed = true
	this.ioRemTime = 0
	var conn net.Conn
	if h, ok := this.req.resp.(http.Hijacker); ok {
		conn, _, _ = h.Hijack()
	}
	if conn != nil {
		logging.Errorf("Detected slow/stalled client. Aborting request: %s (%s)", this.req.Id(), conn.RemoteAddr().String())
		conn.Close()
	} else {
		logging.Errorf("Detected slow/stalled client. Unable to close connection for request: %s", this.req.Id())
	}
}

type ioMonitor struct {
	sync.Mutex
	monitored *list.List
}

func (this *ioMonitor) monitor(w *bufferedWriter) {
	this.Lock()
	w.monElem = this.monitored.PushFront(w)
	this.Unlock()
}

func (this *ioMonitor) remove(w *bufferedWriter) {
	if w.monElem != nil {
		this.Lock()
		this.monitored.Remove(w.monElem)
		this.Unlock()
	}
}

var ioIntr = &ioMonitor{monitored: list.New()}

func (this *ioMonitor) driver() {
	defer func() {
		r := recover()
		if r != nil {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, false)
			s := string(buf[0:n])
			logging.Severef("I/O interrupt driver panic: %v\n%v", r, s)
			go this.driver()
		}
	}()

	ticker := time.NewTicker(_IO_WRITE_MONITOR_PRECISION)
	toCancel := list.New()
	for {
		<-ticker.C

		this.Lock()
		for e := this.monitored.Front(); e != nil; e = e.Next() {
			k := e.Value.(*bufferedWriter)
			if k.ioRemTime > 0 { // non synchronised access is deliberate
				k.ioRemTime -= _IO_WRITE_MONITOR_PRECISION
				if k.ioRemTime <= 0 {
					k.ioRemTime-- // ensure never zero when calling cancel function
					toCancel.PushFront(k.cancelIO)
				}
			}
		}
		this.Unlock()
		for e := toCancel.Front(); e != nil; {
			n := e.Next()
			go e.Value.(func())()
			toCancel.Remove(e)
			e = n
		}
	}
}

func init() {
	go ioIntr.driver()
}
