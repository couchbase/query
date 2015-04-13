//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package err provides user-visible errors and warnings. These errors
include error codes and will eventually provide multi-language
messages.

*/
package errors

import (
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/couchbase/query/value"
)

const (
	EXCEPTION = iota
	WARNING
	NOTICE
	INFO
	LOG
	DEBUG
)

type Errors []Error

// Error will eventually include code, message key, and internal error
// object (cause) and message
type Error interface {
	error
	Code() int32
	TranslationKey() string
	Cause() error
	Level() int
	IsFatal() bool
}

type ErrorChannel chan Error

func NewError(e error, internalMsg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 5000, IKey: "Internal Error", ICause: e,
			InternalMsg: internalMsg, InternalCaller: CallerN(1)}
	}
}

func NewWarning(internalMsg string) Error {
	return &err{level: WARNING, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewNotice(internalMsg string) Error {
	return &err{level: NOTICE, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewInfo(internalMsg string) Error {
	return &err{level: INFO, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewLog(internalMsg string) Error {
	return &err{level: LOG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

func NewDebug(internalMsg string) Error {
	return &err{level: DEBUG, InternalMsg: internalMsg, InternalCaller: CallerN(1)}
}

type err struct {
	ICode          int32
	IKey           string
	ICause         error
	InternalMsg    string
	InternalCaller string
	level          int
}

func (e *err) Error() string {
	switch {
	default:
		return "Unspecified error."
	case e.InternalMsg != "" && e.ICause != nil:
		return e.InternalMsg + " - cause: " + e.ICause.Error()
	case e.InternalMsg != "":
		return e.InternalMsg
	case e.ICause != nil:
		return e.ICause.Error()
	}
}

func (e *err) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"code":    e.ICode,
		"key":     e.IKey,
		"message": e.InternalMsg,
	}
	if e.ICause != nil {
		m["cause"] = e.ICause.Error()
	}
	if e.InternalCaller != "" &&
		!strings.HasPrefix("e.InternalCaller", "unknown:") {
		m["caller"] = e.InternalCaller
	}
	return json.Marshal(m)
}

func (e *err) Level() int {
	return e.level
}

func (e *err) IsFatal() bool {
	if e.level == EXCEPTION {
		return true
	}
	return false
}

func (e *err) Code() int32 {
	return e.ICode
}

func (e *err) TranslationKey() string {
	return e.IKey
}

func (e *err) Cause() error {
	return e.ICause
}

func NewParseError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 4100, IKey: "parse_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewSemanticError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 4200, IKey: "semantic_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewBucketDoesNotExist(bucket string) Error {
	return &err{level: EXCEPTION, ICode: 4040, IKey: "bucket_not_found", InternalMsg: fmt.Sprintf("Bucket %s does not exist", bucket), InternalCaller: CallerN(1)}
}

func NewPoolDoesNotExist(pool string) Error {
	return &err{level: EXCEPTION, ICode: 4041, IKey: "pool_not_found", InternalMsg: fmt.Sprintf("Pool %s does not exist", pool), InternalCaller: CallerN(1)}
}

func NewTimeoutError(timeout *time.Duration) Error {
	return &err{level: EXCEPTION, ICode: 4080, IKey: "timeout", InternalMsg: fmt.Sprintf("Timeout %v exceeded", timeout), InternalCaller: CallerN(1)}
}

func NewTotalRowsInfo(rows int) Error {
	return &err{level: INFO, ICode: 100, IKey: "total_rows", InternalMsg: fmt.Sprintf("%d", rows), InternalCaller: CallerN(1)}
}

func NewTotalElapsedTimeInfo(time string) Error {
	return &err{level: INFO, ICode: 101, IKey: "total_elapsed_time", InternalMsg: fmt.Sprintf("%s", time), InternalCaller: CallerN(1)}
}

func NewNotImplemented(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1001, IKey: "not_implemented", InternalMsg: fmt.Sprintf("Not yet implemented: %v", feature), InternalCaller: CallerN(1)}
}

// service level errors - errors that are created in the service package

func NewServiceErrorReadonly(msg string) Error {
	return &err{level: EXCEPTION, ICode: 1000, IKey: "service.io.readonly", InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewServiceErrorHTTPMethod(method string) Error {
	return &err{level: EXCEPTION, ICode: 1010, IKey: "service.io.http.unsupported_method",
		InternalMsg: fmt.Sprintf("Unsupported http method: %s", method), InternalCaller: CallerN(1)}
}

func NewServiceErrorNotImplemented(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: 1020, IKey: "service.io.request.unimplemented",
		InternalMsg: fmt.Sprintf("%s %s not yet implemented", value, feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorUnrecognizedValue(feature string, value string) Error {
	return &err{level: EXCEPTION, ICode: 1030, IKey: "service.io.request.unrecognized_value",
		InternalMsg: fmt.Sprintf("Unknown %s value: %s", feature, value), InternalCaller: CallerN(1)}
}

func NewServiceErrorBadValue(e error, feature string) Error {
	return &err{level: EXCEPTION, ICode: 1040, IKey: "service.io.request.bad_value", ICause: e,
		InternalMsg: fmt.Sprintf("Error processing %s", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMissingValue(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1050, IKey: "service.io.request.missing_value",
		InternalMsg: fmt.Sprintf("No %s value", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorMultipleValues(feature string) Error {
	return &err{level: EXCEPTION, ICode: 1060, IKey: "service.io.request.multiple_values",
		InternalMsg: fmt.Sprintf("Multiple values for %s.", feature), InternalCaller: CallerN(1)}
}

func NewServiceErrorTypeMismatch(feature string, expected string) Error {
	return &err{level: EXCEPTION, ICode: 1070, IKey: "service.io.request.type_mismatch",
		InternalMsg: fmt.Sprintf("%s has to be of type %s", feature, expected), InternalCaller: CallerN(1)}
}

func NewServiceErrorInvalidJSON(e error) Error {
	return &err{level: EXCEPTION, ICode: 1100, IKey: "service.io.response.invalid_json", ICause: e,
		InternalMsg: "Invalid JSON in results", InternalCaller: CallerN(1)}
}

func NewServiceErrorClientID(id string) Error {
	return &err{level: EXCEPTION, ICode: 1110, IKey: "service.io.response.client_id",
		InternalMsg: "forbidden character (\\ or \") in client_context_id", InternalCaller: CallerN(1)}
}

// admin level errors - errors that are created in the clustering and accounting packages

func NewAdminConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2000, IKey: "admin.clustering.connection_error", ICause: e,
		InternalMsg: "Error connecting to " + msg, InternalCaller: CallerN(1)}
}

func NewAdminInvalidURL(component string, url string) Error {
	return &err{level: EXCEPTION, ICode: 2010, IKey: "admin.invalid_url",
		InternalMsg: fmt.Sprintf("Invalid % url: %s", component, url), InternalCaller: CallerN(1)}
}

func NewAdminDecodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: 2020, IKey: "admin.json_decoding_error", ICause: e,
		InternalMsg: "Error in JSON decoding", InternalCaller: CallerN(1)}
}

func NewAdminEncodingError(e error) Error {
	return &err{level: EXCEPTION, ICode: 2030, IKey: "admin.json_encoding_error", ICause: e,
		InternalMsg: "Error in JSON encoding", InternalCaller: CallerN(1)}
}

func NewAdminGetClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2040, IKey: "admin.clustering.get_cluster_error", ICause: e,
		InternalMsg: "Error retrieving cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2050, IKey: "admin.clustering.add_cluster_error", ICause: e,
		InternalMsg: "Error adding cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveClusterError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2060, IKey: "admin.clustering.remove_cluster_error", ICause: e,
		InternalMsg: "Error removing cluster " + msg, InternalCaller: CallerN(1)}
}

func NewAdminGetNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2070, IKey: "admin.clustering.get_node_error", ICause: e,
		InternalMsg: "Error retrieving node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminNoNodeError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 2080, IKey: "admin.clustering.no_such_node",
		InternalMsg: "No such  node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminAddNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2090, IKey: "admin.clustering.add_node_error", ICause: e,
		InternalMsg: "Error adding node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminRemoveNodeError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2100, IKey: "admin.clustering.remove_node_error", ICause: e,
		InternalMsg: "Error removing node " + msg, InternalCaller: CallerN(1)}
}

func NewAdminMakeMetricError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 2110, IKey: "admin.accounting.metric.create", ICause: e,
		InternalMsg: "Error creating metric " + msg, InternalCaller: CallerN(1)}
}

const ADMIN_AUTH_ERROR = 2120

func NewAdminAuthError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: ADMIN_AUTH_ERROR, IKey: "admin.clustering.authorize", ICause: e,
		InternalMsg: "Error authorizing against cluster " + msg, InternalCaller: CallerN(1)}
}

const ADMIN_ENDPOINT_ERROR = 2130

func NewAdminEndpointError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: ADMIN_ENDPOINT_ERROR, IKey: "admin.service.HttpEndpoint", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

const ADMIN_SSL_NOT_ENABLED = 2140

func NewAdminNotSSLEnabledError() Error {
	return &err{level: EXCEPTION, ICode: ADMIN_SSL_NOT_ENABLED, IKey: "admin.service.ssl_cert",
		InternalMsg: "server is not ssl enabled", InternalCaller: CallerN(1)}
}

// Parse errors - errors that are created in the parse package
func NewParseSyntaxError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 3000, IKey: "parse.syntax_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

// Plan errors - errors that are created in the plan and algebra packages
func NewPlanError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: 4000, IKey: "plan_error", ICause: e, InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

const NO_TERM_NAME = 4010

func NewNoTermNameError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: NO_TERM_NAME, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term must have a name or alias", termType), InternalCaller: CallerN(1)}
}

const DUPLICATE_ALIAS = 4020

func NewDuplicateAliasError(termType string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: DUPLICATE_ALIAS, IKey: iKey,
		InternalMsg: fmt.Sprintf("Duplicate %s alias %s", termType, alias), InternalCaller: CallerN(1)}
}

const SUBQUERY_MISSING_KEYS = 4030

func NewSubqueryMissingKeysError(keyspace string) Error {
	return &err{level: EXCEPTION, ICode: SUBQUERY_MISSING_KEYS, IKey: "plan.build_select.subquery_missing_keys",
		InternalMsg: fmt.Sprintf("FROM in subquery must use KEYS clause: FROM %s.", keyspace), InternalCaller: CallerN(1)}
}

// Execution errors - errors that are created in the execution package

func NewEvaluationError(e error, termType string) Error {
	return &err{level: EXCEPTION, ICode: 5010, IKey: "execution.evaluation_error", ICause: e,
		InternalMsg: fmt.Sprintf("Error evaluating %s.", termType), InternalCaller: CallerN(1)}
}

func NewGroupUpdateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 5020, IKey: "execution.group_update_error", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewInvalidValueError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 5030, IKey: "execution.invalid_value_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewDuplicateFinalGroupError() Error {
	return &err{level: EXCEPTION, ICode: 5040, IKey: "execution.duplicate_final_group",
		InternalMsg: "Duplicate Final Group.", InternalCaller: CallerN(1)}
}

func NewInsertKeyError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5050, IKey: "execution.insert_key_error",
		InternalMsg: fmt.Sprintf("No INSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertValueError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5060, IKey: "execution.insert_value_error",
		InternalMsg: fmt.Sprintf("No INSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewInsertKeyTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5070, IKey: "execution.insert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot INSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewUpsertKeyError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5072, IKey: "execution.upsert_key_error",
		InternalMsg: fmt.Sprintf("No UPSERT key for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertValueError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5075, IKey: "execution.upsert_value_error",
		InternalMsg: fmt.Sprintf("No UPSERT value for %v", v), InternalCaller: CallerN(1)}
}

func NewUpsertKeyTypeError(v value.Value) Error {
	return &err{level: EXCEPTION, ICode: 5078, IKey: "execution.upsert_key_type_error",
		InternalMsg:    fmt.Sprintf("Cannot UPSERT non-string key %v of type %T.", v, v),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5080, IKey: "execution.missing_delete_alias",
		InternalMsg:    fmt.Sprintf("DELETE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewDeleteAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5090, IKey: "execution.delete_alias_metadata",
		InternalMsg:    fmt.Sprintf("DELETE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMissingError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5100, IKey: "execution.missing_update_alias",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s not found in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateAliasMetadataError(alias string) Error {
	return &err{level: EXCEPTION, ICode: 5110, IKey: "execution.update_alias_metadata",
		InternalMsg:    fmt.Sprintf("UPDATE alias %s has no metadata in item.", alias),
		InternalCaller: CallerN(1)}
}

func NewUpdateMissingClone() Error {
	return &err{level: EXCEPTION, ICode: 5120, IKey: "execution.update_missing_clone",
		InternalMsg: "Missing UPDATE clone.", InternalCaller: CallerN(1)}
}

// Authorization Errors
func NewDatastoreAuthorizationError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 10000, IKey: "datastore.couchbase.authorization_error", ICause: e,
		InternalMsg: "Authorization Failed " + msg, InternalCaller: CallerN(1)}
}

// System datastore error codes
func NewSystemDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11000, IKey: "datastore.system.generic_error", ICause: e,
		InternalMsg: "System datastore error " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11001, IKey: "datastore.system.namespace_not_found", ICause: e,
		InternalMsg: "Datastore : namespace not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11002, IKey: "datastore.system.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11003, IKey: "datastore.system.not_implemented", ICause: e,
		InternalMsg: "System datastore :  Not implemented " + msg, InternalCaller: CallerN(1)}

}

func NewSystemNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11004, IKey: "datastore.system.not_supported", ICause: e,
		InternalMsg: "System datastore : Not supported " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11005, IKey: "datastore.system.idx_not_found", ICause: e,
		InternalMsg: "System datastore : Index not found " + msg, InternalCaller: CallerN(1)}

}

func NewSystemIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 11006, IKey: "datastore.system.idx_no_drop", ICause: e,
		InternalMsg: "System datastore : This  index cannot be dropped " + msg, InternalCaller: CallerN(1)}

}

// Datastore/couchbase error codes
func NewCbConnectionError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12000, IKey: "datastore.couchbase.connection_error", ICause: e,
		InternalMsg: "Cannot connect " + msg, InternalCaller: CallerN(1)}

}

func NewCbUrlParseError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12001, IKey: "datastore.couchbase.url_parse", ICause: e,
		InternalMsg: "Cannot parse url " + msg, InternalCaller: CallerN(1)}
}

func NewCbNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12002, IKey: "datastore.couchbase.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12003, IKey: "datastore.couchbase.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbPrimaryIndexNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12004, IKey: "datastore.couchbase.primary_idx_not_found", ICause: e,
		InternalMsg: "Primary Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbIndexerNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12005, IKey: "datastore.couchbase.indexer_not_implemented", ICause: e,
		InternalMsg: "Indexer not implemented " + msg, InternalCaller: CallerN(1)}
}

func NewCbKeyspaceCountError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12006, IKey: "datastore.couchbase.keyspace_count_error", ICause: e,
		InternalMsg: "Failed to get keyspace count " + msg, InternalCaller: CallerN(1)}
}

func NewCbNoKeysFetchError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12007, IKey: "datastore.couchbase.no_keys_fetch", ICause: e,
		InternalMsg: "No keys to fetch " + msg, InternalCaller: CallerN(1)}
}

func NewCbBulkGetError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12008, IKey: "datastore.couchbase.bulk_get_error", ICause: e,
		InternalMsg: "Error performing buck get " + msg, InternalCaller: CallerN(1)}
}

func NewCbDMLError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12009, IKey: "datastore.couchbase.DML_error", ICause: e,
		InternalMsg: "DML Error" + msg, InternalCaller: CallerN(1)}
}

func NewCbNoKeysInsertError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12010, IKey: "datastore.couchbase.no_keys_insert", ICause: e,
		InternalMsg: "No keys to insert " + msg, InternalCaller: CallerN(1)}
}

func NewCbDeleteFailedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12011, IKey: "datastore.couchbase.delete_failed", ICause: e,
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewCbLoadIndexesError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12012, IKey: "datastore.couchbase.load_index_failed", ICause: e,
		InternalMsg: "Failed to load indexes " + msg, InternalCaller: CallerN(1)}
}

func NewCbBucketTypeNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 12013, IKey: "datastore.couchbase.bucket_type_not_supported", ICause: e,
		InternalMsg: "This bucket type is not supported " + msg, InternalCaller: CallerN(1)}
}

func NewCbIndexStateError(msg string) Error {
	return &err{level: EXCEPTION, ICode: 12014, IKey: "datastore.couchbase.index_state_error",
		InternalMsg: msg, InternalCaller: CallerN(1)}
}

func NewIndexScanSizeError(size int64) Error {
	return &err{level: EXCEPTION, ICode: 12015, IKey: "datastore.index.scan_size_error",
		InternalMsg: fmt.Sprintf("Unacceptable size for index scan: %d", size), InternalCaller: CallerN(1)}
}

// Datastore/couchbase/view index error codes
func NewCbViewCreateError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13000, IKey: "datastore.couchbase.view.create_failed", ICause: e,
		InternalMsg: "Failed to create view " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13001, IKey: "datastore.couchbase.view.not_found", ICause: e,
		InternalMsg: "View Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewExistsError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13003, IKey: "datastore.couchbase.view.exists", ICause: e,
		InternalMsg: "View index exists " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsWithNotAllowedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13004, IKey: "datastore.couchbase.view.with_not_allowed", ICause: e,
		InternalMsg: "Views not allowed for WITH keyword " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13005, IKey: "datastore.couchbase.view.not_supported", ICause: e,
		InternalMsg: "View indexes not supported " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsDropIndexError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13006, IKey: "datastore.couchbase.view.drop_index_error", ICause: e,
		InternalMsg: "Failed to drop index " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewsAccessError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13007, IKey: "datastore.couchbase.view.access_error", ICause: e,
		InternalMsg: "Failed to access view " + msg, InternalCaller: CallerN(1)}
}

func NewCbViewIndexesLoadingError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 13008, IKey: "datastore.couchbase.view.not_found", ICause: e,
		InternalMsg: "Failed to load indexes for keyspace " + msg, InternalCaller: CallerN(1)}
}

// Datastore File based error codes

func NewFileDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15000, IKey: "datastore.file.generic_file_error", ICause: e,
		InternalMsg: "Error in file datastore " + msg, InternalCaller: CallerN(1)}
}

func NewFileNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15001, IKey: "datastore.file.namespace_not_found", ICause: e,
		InternalMsg: "Namespace not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15002, IKey: "datastore.file.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateNamespaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15003, IKey: "datastore.file.duplicate_namespace", ICause: e,
		InternalMsg: "Duplicate Namespace " + msg, InternalCaller: CallerN(1)}
}

func NewFileDuplicateKeyspaceError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15004, IKey: "datastore.file.duplicate_keyspace", ICause: e,
		InternalMsg: "Duplicate Keyspace " + msg, InternalCaller: CallerN(1)}
}

func NewFileNoKeysInsertError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15005, IKey: "datastore.file.no_keys_insert", ICause: e,
		InternalMsg: "No keys to insert " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyExists(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15006, IKey: "datastore.file.key_exists", ICause: e,
		InternalMsg: "Key Exists " + msg, InternalCaller: CallerN(1)}
}

func NewFileDMLError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15007, IKey: "datastore.file.DML_error", ICause: e,
		InternalMsg: "DML Error " + msg, InternalCaller: CallerN(1)}
}

func NewFileKeyspaceNotDirError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15008, IKey: "datastore.file.keyspacenot_dir", ICause: e,
		InternalMsg: "Keyspace path must be a directory " + msg, InternalCaller: CallerN(1)}
}

func NewFileIdxNotFound(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15009, IKey: "datastore.file.idx_not_found", ICause: e,
		InternalMsg: "Index not found " + msg, InternalCaller: CallerN(1)}
}

func NewFileNotSupported(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15010, IKey: "datastore.file.not_supported", ICause: e,
		InternalMsg: "Operation not supported " + msg, InternalCaller: CallerN(1)}
}

func NewFilePrimaryIdxNoDropError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 15011, IKey: "datastore.file.primary_idx_no_drop", ICause: e,
		InternalMsg: "Primary Index cannot be dropped " + msg, InternalCaller: CallerN(1)}
}

// Error codes for all other datastores, e.g Mock
func NewOtherDatastoreError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16000, IKey: "datastore.other.datastore_generic_error", ICause: e,
		InternalMsg: "Error in datastore " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNamespaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16001, IKey: "datastore.other.namespace_not_found", ICause: e,
		InternalMsg: "Namespace Not Found " + msg, InternalCaller: CallerN(1)}
}

func NewOtherKeyspaceNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16002, IKey: "datastore.other.keyspace_not_found", ICause: e,
		InternalMsg: "Keyspace Not Found " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNotImplementedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16003, IKey: "datastore.other.not_implemented", ICause: e,
		InternalMsg: "Not Implemented " + msg, InternalCaller: CallerN(1)}
}

func NewOtherIdxNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16004, IKey: "datastore.other.idx_not_found", ICause: e,
		InternalMsg: "Index not found  " + msg, InternalCaller: CallerN(1)}
}

func NewOtherIdxNoDrop(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16005, IKey: "datastore.other.idx_no_drop", ICause: e,
		InternalMsg: "Index Cannot be dropped " + msg, InternalCaller: CallerN(1)}
}

func NewOtherNotSupportedError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16006, IKey: "datastore.other.not_supported", ICause: e,
		InternalMsg: "Not supported for this datastore " + msg, InternalCaller: CallerN(1)}
}

func NewOtherKeyNotFoundError(e error, msg string) Error {
	return &err{level: EXCEPTION, ICode: 16007, IKey: "datastore.other.key_not_found", ICause: e,
		InternalMsg: "Key not found " + msg, InternalCaller: CallerN(1)}
}

// Returns "FileName:LineNum" of caller.
func Caller() string {
	return CallerN(1)
}

// Returns "FileName:LineNum" of the Nth caller on the call stack,
// where level of 0 is the caller of CallerN.
func CallerN(level int) string {
	_, fname, lineno, ok := runtime.Caller(1 + level)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d",
		strings.Split(path.Base(fname), ".")[0], lineno)
}
