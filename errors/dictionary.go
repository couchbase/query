//  Copyright 2020-Present Couchbase, Inc.
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

func NewDictInternalError(msg string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_DICT_INTERNAL, IKey: "dictionary.internal", ICause: e,
		InternalMsg: "Unexpected error in dictionary: " + msg, InternalCaller: CallerN(1)}
}

func NewInvalidGSIIndexerError(s string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_GSI_INDEXER, IKey: "dictionary.invalid_indexer",
		InternalMsg: "GSI Indexer does not support collections - " + s, InternalCaller: CallerN(1)}
}

func NewInvalidGSIIndexError(s string) Error {
	return &err{level: EXCEPTION, ICode: E_INVALID_GSI_INDEX, IKey: "dictionary.invalid_index",
		InternalMsg: "GSI Index " + s + " does not support collections", InternalCaller: CallerN(1)}
}

func NewSystemCollectionError(s string, e error) Error {
	return &err{level: EXCEPTION, ICode: E_SYSTEM_COLLECTION, IKey: "dictionary.system_collection", ICause: e,
		InternalMsg: "Error accessing system collection - " + s, InternalCaller: CallerN(1)}
}

func NewDictionaryEncodingError(what string, name string, reason error) Error {
	return &err{level: EXCEPTION, ICode: E_DICTIONARY_ENCODING, IKey: "dictionary.encoding_error", ICause: reason,
		InternalMsg:    fmt.Sprintf("Cound not %s data dictionary entry for %s due to %v", what, name, reason),
		InternalCaller: CallerN(1)}
}

func NewDictKeyspaceMismatchError(ks1, ks2 string) Error {
	return &err{level: EXCEPTION, ICode: E_DICT_KEYSPACE_MISMATCH, IKey: "dictionary.keyspace_mismatch_error",
		InternalMsg:    fmt.Sprintf("Decoded dictionary entry for keyspace %s does not match %s", ks2, ks1),
		InternalCaller: CallerN(1)}
}

func NewDictMissingFieldError(entry, name, field string) Error {
	return &err{level: EXCEPTION, ICode: E_DICT_MISSING_FIELD, IKey: "dictionary.missing_field_error",
		InternalMsg:    fmt.Sprintf("Dictionary entry '%s' for '%s' is missing field '%s'", entry, name, field),
		InternalCaller: CallerN(1)}
}
