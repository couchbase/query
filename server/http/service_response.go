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
	"io"
	"net/http"
	"strconv"
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

func (this *httpRequest) Output() execution.Output {
	return this
}

func (this *httpRequest) Fail(err errors.Error) {
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
	case errors.E_SERVICE_SHUTTING_DOWN, errors.E_SERVICE_SHUT_DOWN:
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
	this.writeString("\n}\n")
	this.writer.noMoreData()

	this.Stop(server.FATAL)
}

func (this *httpRequest) markTimeOfCompletion(now time.Time) {
	this.executionTime = now.Sub(this.ServiceTime())
	this.elapsedTime = now.Sub(this.RequestTime())
	if !this.TransactionStartTime().IsZero() {
		this.transactionElapsedTime = now.Sub(this.TransactionStartTime())
	}
}

func (this *httpRequest) Execute(srvr *server.Server, context *execution.Context, reqType string, signature value.Value, startTx bool) {
	this.prefix, this.indent = this.prettyStrings(srvr.Pretty(), false)

	this.setHttpCode(http.StatusOK)
	this.writePrefix(srvr, signature, this.prefix, this.indent)

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
	this.writeSuffix(srvr, state, this.prefix, this.indent)
	this.writer.noMoreData()
}

func (this *httpRequest) Expire(state server.State, timeout time.Duration) {
	this.Error(errors.NewTimeoutError(timeout))
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

func (this *httpRequest) writeRequestID(prefix string) bool {
	return this.writeString(prefix) && this.writeString("\"requestID\": \"") && this.writeString(this.Id().String()) && this.writeString("\"")
}

func (this *httpRequest) writeClientContextID(prefix string) bool {
	if !this.ClientID().IsValid() {
		return true
	}
	return this.writeString(",\n") && this.writeString(prefix) &&
		this.writeString("\"clientContextID\": \"") && this.writeString(this.ClientID().String()) && this.writeString("\"")
}

func (this *httpRequest) writePrepared(prefix, indent string) bool {
	prepared := this.Prepared()
	if this.AutoExecute() != value.TRUE || prepared == nil {
		return true
	}
	host := tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())
	name := distributed.RemoteAccess().MakeKey(host, prepared.Name())
	return this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"prepared\": \"") && this.writeString(name) && this.writeString("\"")
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
	return this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"signature\": ") && this.writeValue(signature, prefix, indent, true)
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

	if this.resultCount == 0 {
		success = this.writer.write("\n")
	} else {
		success = this.writer.write(",\n")
	}
	if success {
		success = this.writer.write(this.prefix)
	}
	beforeResult := this.writer.mark()

	if success {
		order := item.ProjectionOrder()
		err := item.WriteJSON(order, this.writer.buf(), this.prefix, this.indent, item.Self())
		if err != nil {
			this.Error(errors.NewServiceErrorInvalidJSON(err))
			this.SetState(server.FATAL)
			success = false
		} else {
			this.resultSize += (this.writer.mark() - beforeResult)
			this.resultCount++
			this.writer.sizeFlush()
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

func (this *httpRequest) writeValue(item value.Value, prefix, indent string, fast bool) bool {
	if item == nil {
		return this.writeString("null")
	}
	beforeWriteJSON := this.writer.mark()
	var order []string
	if av, ok := item.(value.AnnotatedValue); ok {
		order = av.ProjectionOrder()
	}
	err := item.WriteJSON(order, this.writer.buf(), prefix, indent, fast)
	if err != nil {
		this.writer.truncate(beforeWriteJSON)
		return this.writer.printf("\"ERROR: %v\"", err)
	}
	return true
}

func (this *httpRequest) writeSuffix(srvr *server.Server, state server.State, prefix, indent string) bool {
	return this.writeString("\n") && this.writeString(prefix) && this.writeString("]") &&
		this.writeErrors(prefix, indent) &&
		this.writeWarnings(prefix, indent) &&
		this.writeState(state, prefix) &&
		this.writeMetrics(srvr.Metrics(), prefix, indent) &&
		this.writeServerless(srvr.Metrics(), prefix, indent) &&
		this.writeProfile(srvr.Profile(), prefix, indent) &&
		this.writeControls(srvr.Controls(), prefix, indent) &&
		this.writeString("\n}\n")
}

func (this *httpRequest) writeString(s string) bool {
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

			// MB-19307: please check the comments
			// in mapErrortoHttpResponse().
			// Ideally we should set the status code
			// only before calling writePrefix()
			// but this is too cumbersome, having
			// to check Execution errors as well.
			if this.State() != server.FATAL {
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
		this.writeString(this.elapsedTime.String()) &&
		this.writeString("\",") &&
		this.writeString(newPrefix) &&
		this.writeString("\"executionTime\": \"") &&
		this.writeString(this.executionTime.String()) &&
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
		fmt.Fprintf(buf, ",%s\"transactionElapsedTime\": \"%s\"", newPrefix, this.transactionElapsedTime.String())
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

func (this *httpRequest) writeControls(controls bool, prefix, indent string) bool {
	var newPrefix string
	var e []byte
	var err error

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

	if this.UseReplica() {
		if !this.writeString(",") {
			return false
		}
		if err != nil || !this.writer.printf("%s\"use_replica\": \"%v\"", newPrefix, this.UseReplica()) {
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

	if !this.writeString(",") || !this.writer.printf("%s\"n1ql_feat_ctrl\": \"%v\"", newPrefix, this.FeatureControls()) {
		logging.Infof("Error writing n1l_feat_ctrl")
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

func (this *httpRequest) writeTransactionInfo(prefix, indent string) bool {
	if this.TxId() != "" {
		if !this.writeString(",") || !this.writer.printf("%s\"txid\": \"%v\"", prefix, this.TxId()) {
			logging.Infof("Error writing txid")
		}

		if !this.writeString(",") || !this.writer.printf("%s\"tximplicit\": \"%v\"", prefix, this.TxImplicit()) {
			logging.Infof("Error writing tximplicit")
		}

		if !this.writeString(",") || !this.writer.printf("%s\"txstmtnum\": \"%v\"", prefix, this.TxStmtNum()) {
			logging.Infof("Error writing stmtnum")
		}

		if !this.writeString(",") || !this.writer.printf("%s\"txtimeout\": \"%v\"", prefix, this.TxTimeout()) {
			logging.Infof("Error writing txtimeout")
		}

		if !this.writeString(",") || !this.writer.printf("%s\"durability_level\": \"%v\"",
			prefix, datastore.DurabilityLevelToName(this.DurabilityLevel())) {
			logging.Infof("Error writing durability_level")
		}

		if !this.writeString(",") ||
			!this.writer.printf("%s\"durability_timeout\": \"%v\"", prefix, this.DurabilityTimeout()) {
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
	rv := this.writeString(",\n") && this.writeString(prefix) && this.writeString("\"profile\": {")
	if !rv {
		return false
	}
	if p != server.ProfOff {
		phaseTimes := this.FmtPhaseTimes()
		if phaseTimes != nil {
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
		if !this.writer.printf("%s\"requestTime\": \"%s\"", newPrefix, this.RequestTime().Format(expression.DEFAULT_FORMAT)) {
			logging.Infof("Error writing request time")
		}
		if !this.writer.printf(",%s\"servicingHost\": \"%s\"", newPrefix, tenant.EncodeNodeName(distributed.RemoteAccess().WhoAmI())) {
			logging.Infof("Error writing servicing host")
		}
		needComma = true
	}
	if p == server.ProfOn || p == server.ProfBench {
		timings := this.GetTimings()
		if timings != nil {
			if indent != "" {
				e, err = json.MarshalIndent(timings, "\t", indent)
			} else {
				e, err = json.Marshal(timings)
			}
			if err != nil || !this.writer.printf(",%s\"executionTimings\": %s", newPrefix, e) {
				logging.Infof("Error writing executionTimings: %v", err)
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
			this.SetFmtOptimizerEstimates(e)
		}
	}
	if prefix != "" && !(this.writeString("\n") && this.writeString(prefix)) {
		return false
	}
	return this.writeString("}")
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
}

const _PRINTF_THRESHOLD = 128

func NewBufferedWriter(w *bufferedWriter, r *httpRequest, bp BufferPool) {
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
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		io.Copy(w, this.buffer)
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
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		io.Copy(w, this.buffer)
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
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		io.Copy(w, this.buffer)
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}
}

// flush on a full buffer
func (this *bufferedWriter) sizeFlush() {
	if this.closed {
		return
	}

	// beyond capacity
	if this.buffer.Len() > this.buffer_pool.BufferCapacity() {
		w := this.req.resp // our request's response writer

		// write response header and data buffered so far using request's response writer:
		if this.header {
			w.WriteHeader(this.req.httpCode())
			this.header = false
		}

		// write out and empty the buffer
		io.Copy(w, this.buffer)
		this.buffer.Reset()

		// do the flushing
		this.lastFlush = util.Now()
		w.(http.Flusher).Flush()
	}
}

// mark the current write position
func (this *bufferedWriter) mark() int {
	return this.buffer.Len()
}

func (this *bufferedWriter) truncate(mark int) {
	this.buffer.Truncate(mark)
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
		w.WriteHeader(this.req.httpCode())
		this.header = false
	}

	io.Copy(w, this.buffer)
	// no more data in the response => return buffer to pool:
	this.buffer_pool.PutBuffer(this.buffer)
	r.Body.Close()
	this.closed = true
}
