//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package errors

import (
	"fmt"
)

func NewInferInvalidOption(o string) Error {
	c := make(map[string]interface{})
	c["invalid_option"] = o
	return &err{level: EXCEPTION, ICode: E_INFER_INVALID_OPTION, IKey: "infer.invalid_option", cause: c,
		InternalMsg: fmt.Sprintf("Invalid option '%s'", o), InternalCaller: CallerN(1)}
}

func NewInferOptionMustBeNumeric(o string, t string) Error {
	c := make(map[string]interface{})
	c["option"] = o
	c["type"] = t
	return &err{level: EXCEPTION, ICode: E_INFER_OPTION_MUST_BE_NUMERIC, IKey: "infer.option.not_numeric", cause: c,
		InternalMsg: fmt.Sprintf("Option '%s' must be numeric.", o), InternalCaller: CallerN(1)}
}

func NewInferErrorReadingNumber(o string, v string) Error {
	c := make(map[string]interface{})
	c["option"] = o
	c["value"] = v
	return &err{level: EXCEPTION, ICode: E_INFER_READING_NUMBER, IKey: "infer.option.error_reading", cause: c,
		InternalMsg: fmt.Sprintf("Error reading option '%s'.", o), InternalCaller: CallerN(1)}
}

func NewInferNoKeyspaceDocuments(name string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = name
	return &err{level: EXCEPTION, ICode: E_INFER_NO_KEYSPACE_DOCUMENTS, IKey: "infer.keyspace.no_documents", cause: c,
		InternalMsg:    "Keyspace has no documents, schema inference not possible.",
		InternalCaller: CallerN(1)}
}

func NewInferCreateRetrieverFailed(errors ...Error) Error {
	c := make(map[string]interface{})
	c["errors"] = errors
	return &err{level: EXCEPTION, ICode: E_INFER_CREATE_RETRIEVER, IKey: "infer.create.retriever.failed", cause: c,
		InternalMsg: "Error creating document retriever.", InternalCaller: CallerN(1)}
}

func NewInferNoRandomEntryProvider(k string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	return &err{level: EXCEPTION, ICode: E_INFER_NO_RANDOM_ENTRY, IKey: "infer.keyspace.no_random_entry_provider", cause: c,
		InternalMsg:    "Keyspace does not support random document retrieval.",
		InternalCaller: CallerN(1)}
}

func NewInferNoRandomScanProvider(k string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	return &err{level: EXCEPTION, ICode: E_INFER_NO_RANDOM_SCAN, IKey: "infer.keyspace.no_random_scan_provider", cause: c,
		InternalMsg:    "Keyspace does not support random key scans.",
		InternalCaller: CallerN(1)}
}

func NewInferNoRandomDocuments(k string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	return &err{level: EXCEPTION, ICode: E_INFER_NO_RANDOM_DOCS, IKey: "infer.keyspace.no_random_docs", cause: c,
		InternalMsg: "Keyspace will not return random documents.", InternalCaller: CallerN(1)}
}

func NewInferMissingContext(t string) Error {
	c := make(map[string]interface{})
	c["context_type"] = t
	return &err{level: EXCEPTION, ICode: E_INFER_MISSING_CONTEXT, IKey: "infer.missing_context", cause: c,
		InternalMsg: "Missing expression context.", InternalCaller: CallerN(1)}
}

func NewInferExpressionEvalFailed(e error) Error {
	return &err{level: EXCEPTION, ICode: E_INFER_EXPRESSION_EVAL, IKey: "infer.expression_eval_failed", cause: e,
		InternalMsg: "Expression evaluation failed.", InternalCaller: CallerN(1)}
}

func NewInferKeyspaceError(k string, e error) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	c["cause"] = e
	return &err{level: EXCEPTION, ICode: E_INFER_KEYSPACE_ERROR, IKey: "infer.keyspace.error", cause: c,
		InternalMsg: "Keyspace error.", InternalCaller: CallerN(1)}
}

func NewInferNoSuitablePrimaryIndex(k string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	return &err{level: EXCEPTION, ICode: E_INFER_NO_SUITABLE_PRIMARY_INDEX, IKey: "infer.keyspace.no_primary", cause: c,
		InternalMsg: "No suitable primary index found.", InternalCaller: CallerN(1)}
}

func NewInferNoSuitableSecondaryIndex(k string) Error {
	c := make(map[string]interface{})
	c["keyspace"] = k
	return &err{level: EXCEPTION, ICode: E_INFER_NO_SUITABLE_SECONDARY_INDEX, IKey: "infer.keyspace.no_secondary", cause: c,
		InternalMsg: "No suitable secondary index found.", InternalCaller: CallerN(1)}
}

func NewInferTimeout(to int32) Error {
	c := make(map[string]interface{})
	c["infer_timeout"] = to
	return &err{level: WARNING, ICode: E_INFER_TIMEOUT, IKey: "infer.timeout", cause: c,
		InternalMsg: "Stopped after exceeding infer_timeout. Schema may be incomplete.", InternalCaller: CallerN(1)}
}

func NewInferSizeLimit(l int32) Error {
	c := make(map[string]interface{})
	c["max_schema_MB"] = l
	return &err{level: WARNING, ICode: E_INFER_SIZE_LIMIT, IKey: "infer.size_limit", cause: c,
		InternalMsg: "Stopped after exceeding max_schema_MB. Schema may be incomplete.", InternalCaller: CallerN(1)}
}

func NewInferNoDocuments() Error {
	return &err{level: EXCEPTION, ICode: E_INFER_NO_DOCUMENTS, IKey: "infer.no_documents",
		InternalMsg: "No documents found, unable to infer schema.", InternalCaller: CallerN(1)}
}

func NewInferConnectFailed(url string, e error) Error {
	c := make(map[string]interface{})
	c["server"] = url
	c["cause"] = e
	return &err{level: EXCEPTION, ICode: E_INFER_CONNECT, IKey: "infer.connect.failed", cause: c,
		InternalMsg: "Failed to connect to the server.", InternalCaller: CallerN(1)}
}

func NewInferGetPoolFailed(e error) Error {
	return &err{level: EXCEPTION, ICode: E_INFER_GET_POOL, IKey: "infer.pool_get.failed", cause: e,
		InternalMsg: "Failed to access pool 'default'.", InternalCaller: CallerN(1)}
}

func NewInferGetBucketFailed(b string, e error) Error {
	c := make(map[string]interface{})
	c["bucket"] = b
	c["cause"] = e
	return &err{level: EXCEPTION, ICode: E_INFER_GET_BUCKET, IKey: "infer.bucket_get.failed", cause: c,
		InternalMsg: "Failed to access bucket.", InternalCaller: CallerN(1)}
}

func NewInferIndexWarning() Error {
	return &err{level: WARNING, ICode: E_INFER_INDEX_WARNING, IKey: "infer.index_warning",
		InternalMsg:    "Index scanning only; document sample may not be representative.",
		InternalCaller: CallerN(1)}
}

func NewInferRandomError(e error) Error {
	return &err{level: EXCEPTION, ICode: E_INFER_GET_RANDOM, IKey: "infer.random_get.failed", cause: e,
		InternalMsg:    "Failed to get random document.",
		InternalCaller: CallerN(1)}
}
