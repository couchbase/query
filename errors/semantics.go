//  Copyright 2018-Present Couchbase, Inc.
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

// semantics errors
// note the error number range here shares the same range (3000) as parser errors

func NewSemanticsError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		return &err{level: EXCEPTION, ICode: E_SEMANTICS, IKey: "semantics_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

func NewSemanticsWithCauseError(e error, msg string) Error {
	switch e := e.(type) {
	case Error: // if given error is already an Error, just return it:
		return e
	default:
		var c map[string]interface{}
		if e != nil {
			c = make(map[string]interface{})
			c["cause"] = e.Error()
		}
		return &err{level: EXCEPTION, ICode: E_SEMANTICS, IKey: "semantics_error", cause: c,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

func NewSemanticsInternalError(what string) Error {
	return &err{level: EXCEPTION, ICode: E_SEMANTICS_INTERNAL, IKey: "semantics.internal_error",
		InternalMsg: fmt.Sprintf("Internal semantics error: %v", what), InternalCaller: CallerN(1)}
}

func NewJoinNestNoJoinHintError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_JOIN_NEST_NO_JOIN_HINT, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have join hint (USE HASH or USE NL).", op, alias),
		InternalCaller: CallerN(1)}
}

func NewJoinNestNoUseKeysError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_JOIN_NEST_NO_USE_KEYS, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have USE KEYS.", op, alias),
		InternalCaller: CallerN(1)}
}

func NewJoinNestNoUseIndexError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_JOIN_NEST_NO_USE_INDEX, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have USE INDEX.", op, alias),
		InternalCaller: CallerN(1)}
}

// Error code 3140 is retired. Do not reuse.
func NewMergeInsertNoKeyError() Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_INSERT_NO_KEY, IKey: "semantics.visit_merge.merge_insert_no_key",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have document key specification in INSERT action."),
		InternalCaller: CallerN(1)}
}

func NewMergeInsertMissingKeyError() Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_INSERT_MISSING_KEY, IKey: "semantics.visit_merge.merge_insert_missing_key",
		InternalMsg:    fmt.Sprintf("MERGE with ON clause must have document key specification in INSERT action."),
		InternalCaller: CallerN(1)}
}

func NewMergeMissingSourceError() Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_MISSING_SOURCE, IKey: "semantics.visit_merge.merge_missing_source",
		InternalMsg:    fmt.Sprintf("MERGE is missing source."),
		InternalCaller: CallerN(1)}
}

func NewMergeNoIndexHintError() Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_NO_INDEX_HINT, IKey: "semantics.visit_merge.merge_no_index_hint",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have USE INDEX hint specified on target."),
		InternalCaller: CallerN(1)}
}

func NewMergeNoJoinHintError() Error {
	return &err{level: EXCEPTION, ICode: E_MERGE_NO_JOIN_HINT, IKey: "semantics.visit_merge.merge_no_join_hint",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have join hint specified on source."),
		InternalCaller: CallerN(1)}
}

func NewMixedJoinError(op1 string, alias1 string, op2 string, alias2 string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_MIXED_JOIN, IKey: iKey,
		InternalMsg:    fmt.Sprintf("Cannot mix %s on %s with %s on %s.", op1, alias1, op2, alias2),
		InternalCaller: CallerN(1)}
}

// Error code 3210 is retired. Do not reuse.
func NewWindowSemanticError(fname, wclause, cause, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_WINDOW_SEMANTIC, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s window function %s%s", fname, wclause, cause),
		InternalCaller: CallerN(1)}
}

func NewEnterpriseFeature(opmsg, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_ENTERPRISE_FEATURE, IKey: iKey,
		InternalMsg:    fmt.Sprintf("'%s' is an enterprise level feature.", opmsg),
		InternalCaller: CallerN(1)}
}

// Error code 3240 is retired. Do not reuse.
func NewAdviseUnsupportedStmtError(iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_ADVISE_UNSUPPORTED_STMT, IKey: iKey,
		InternalMsg: fmt.Sprintf("Advise supports SELECT, MERGE, UPDATE and DELETE statements only."), InternalCaller: CallerN(1)}
}

func NewAdvisorProjOnly() Error {
	return &err{level: EXCEPTION, ICode: E_ADVISOR_PROJ_ONLY, IKey: "semantics_advisor_function",
		InternalMsg: fmt.Sprintf("Advisor function is only allowed in projection clause."), InternalCaller: CallerN(1)}
}

func NewAdvisorNoFrom() Error {
	return &err{level: EXCEPTION, ICode: E_ADVISOR_NO_FROM, IKey: "semantics_advisor_function",
		InternalMsg:    fmt.Sprintf("FROM clause is not allowed when Advisor function is present in projection clause."),
		InternalCaller: CallerN(1)}
}

func NewMHDPOnlyFeature(what, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_MHDP_ONLY_FEATURE, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s is only supported in Developer Preview Mode.", what),
		InternalCaller: CallerN(1)}
}

func NewMissingUseKeysError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_MISSING_USE_KEYS, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term must have USE KEYS", termType), InternalCaller: CallerN(1)}
}

func NewHasUseIndexesError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_HAS_USE_INDEXES, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term should not have USE INDEX", termType), InternalCaller: CallerN(1)}
}

func NewUpdateStatInvalidIndexTypeError() Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_STAT_INVALID_INDEX_TYPE, IKey: "semantics_update_statistics",
		InternalMsg: "UPDATE STATISTICS (ANALYZE) supports GSI indexes only for INDEX option.", InternalCaller: CallerN(1)}
}

func NewUpdateStatIndexAllCollectionOnly() Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_STAT_INDEX_ALL_COLLECTION_ONLY, IKey: "semantics_update_statistics",
		InternalMsg:    "INDEX ALL option for UPDATE STATISTICS (ANALYZE) can only be used for a collection.",
		InternalCaller: CallerN(1)}
}

func NewUpdateStatSelf(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_UPDATE_STAT_SELF_NOTALLOWED, IKey: "semantics_update_statistics_self",
		InternalMsg:    fmt.Sprintf("UPDATE STATISTICS of '%s'%s is not allowed", msg, at),
		InternalCaller: CallerN(1)}
}

func NewCreateIndexNotIndexable(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_CREATE_INDEX_NOT_INDEXABLE, IKey: "semantics_create_index",
		InternalMsg: fmt.Sprintf("%s%s is not indexable.", msg, at), InternalCaller: CallerN(1)}
}

func NewCreateIndexAttributeMissing(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_CREATE_INDEX_ATTRIBUTE_MISSING, IKey: "semantics_create_index",
		InternalMsg:    fmt.Sprintf("%s%s MISSING attribute not allowed (Only allowed with gsi leading key).", msg, at),
		InternalCaller: CallerN(1)}
}

func NewCreateIndexAttribute(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_CREATE_INDEX_ATTRIBUTE, IKey: "semantics_create_index",
		InternalMsg:    fmt.Sprintf("Attributes are not allowed on %s%s of flatten_keys.", msg, at),
		InternalCaller: CallerN(1)}
}

func NewFlattenKeys(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_FLATTEN_KEYS, IKey: "semantics_flatten_keys",
		InternalMsg:    fmt.Sprintf("%s%s is not allowed in this context.", msg, at),
		InternalCaller: CallerN(1)}
}

func NewAllDistinctNotAllowed(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_ALL_DISTINCT_NOT_ALLOWED, IKey: "semantics_no_distinct",
		InternalMsg:    fmt.Sprintf("ALL/DISTINCT is not allowed in %s%s.", msg, at),
		InternalCaller: CallerN(1)}
}

func NewCreateIndexSelf(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_CREATE_INDEX_SELF_NOTALLOWED, IKey: "semantics_create_index_self",
		InternalMsg:    fmt.Sprintf("Index of '%s'%s is not allowed", msg, at),
		InternalCaller: CallerN(1)}
}

func NewIndexNotAllowed(msg, at string) Error {
	return &err{level: EXCEPTION, ICode: E_INDEX_NOT_ALLOWED, IKey: "semantics_index",
		InternalMsg: fmt.Sprintf("%s%s is not allowed.", msg, at), InternalCaller: CallerN(1)}
}

func NewFirstTermJoinHintError(alias string) Error {
	return &err{level: EXCEPTION, ICode: E_JOIN_HINT_FIRST_FROM_TERM, IKey: "semantics_joinhint_first_term",
		InternalMsg:    fmt.Sprintf("Join hint (USE HASH or USE NL) cannot be specified on the first from term '%s'.", alias),
		InternalCaller: CallerN(1)}
}

func NewOrderByValidationError(what string, expr string) Error {
	return &err{level: EXCEPTION, ICode: E_ORDER_BY_VALIDATION_FAIL, IKey: "semantics_order_by_validation",
		InternalMsg: fmt.Sprintf("%s '%s' is not a valid constant, named, positional or function parameter.",
			what, expr), InternalCaller: CallerN(1)}
}

func NewVectorFunctionError(msg string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_SEMANTIC, IKey: "semantics_vector_function",
		InternalMsg:    msg,
		InternalCaller: CallerN(1)}
}

func NewVectorIndexAttrError(key, attr string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_INDEX_ATTRIBUTE, IKey: "semantics_vector_index_attribute",
		InternalMsg:    fmt.Sprintf("Cannot mix index attribute VECTOR with %s for index key %s", attr, key),
		InternalCaller: CallerN(1)}
}

func NewVectorIndexSingleVector(name string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_INDEX_SINGLE_VECTOR, IKey: "semantics_vector_index_key",
		InternalMsg:    fmt.Sprintf("Cannot have more than one vector index key for index %s", name),
		InternalCaller: CallerN(1)}
}

func NewVectorIndexSingleKey(name string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_INDEX_SINGLE_KEY, IKey: "semantics_vector_index_key",
		InternalMsg:    fmt.Sprintf("Cannot have more than one index key for VECTOR index %s", name),
		InternalCaller: CallerN(1)}
}

func NewVectorIndexNoVector(name string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_INDEX_NO_VECTOR, IKey: "semantics_vector_index_key",
		InternalMsg:    fmt.Sprintf("VECTOR index %s is created with no vector index key", name),
		InternalCaller: CallerN(1)}
}

func NewVectorOrderConst(term, option string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_FUNC_ORDER_CONST, IKey: "semantics_orderby_vector_func",
		InternalMsg:    fmt.Sprintf("Vector function (%s) in ORDER BY clause must use a constant for %s", term, option),
		InternalCaller: CallerN(1)}
}

func NewVectorOrderOption(term, option string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_FUNC_ORDER_OPTION, IKey: "semantics_orderby_vector_func",
		InternalMsg:    fmt.Sprintf("Invalid option (%s) for vector function (%s) in ORDER BY clause", option, term),
		InternalCaller: CallerN(1)}
}

func NewVectorDistinctArrayKey() Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_DISTINCT_ARRAY_KEY,
		IKey:           "semantics_distinct_array_key",
		InternalMsg:    "Cannot have DISTINCT array index key for VECTOR expression",
		InternalCaller: CallerN(1)}
}

func NewVectorConstantIndexKey(name string) Error {
	return &err{level: EXCEPTION, ICode: E_VECTOR_CONSTANT_INDEX_KEY,
		IKey:           "semantics_constant_index__key",
		InternalMsg:    fmt.Sprintf("Vector expression '%s' cannot use a constant construct (object or array)", name),
		InternalCaller: CallerN(1)}
}

/*
---- BEGIN MOVED error numbers ----

	The following error numbers (in the 4000 range) originally reside in plan.go (before the introduction of the semantics package)
	although they are semantic errors. They are moved from plan.go to semantics.go but their original error numbers are kept.
*/
func NewNoTermNameError(termType string, errContext string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_NO_TERM_NAME, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term%s must have a name or alias", termType, errContext), InternalCaller: CallerN(1)}
}

func NewDuplicateAliasError(termType string, alias string, errContext string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_ALIAS, IKey: iKey,
		InternalMsg: fmt.Sprintf("Duplicate %s alias '%s'%s", termType, alias, errContext), InternalCaller: CallerN(1)}
}

func NewDuplicateWithAliasError(termType string, alias string, errContext string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_DUPLICATE_WITH_ALIAS, IKey: iKey, InternalCaller: CallerN(1),
		InternalMsg: fmt.Sprintf("Duplicate WITH alias reference in %s: '%s'%s", termType, alias, errContext)}
}

func NewUnknownForError(termType string, keyFor string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_UNKNOWN_FOR, IKey: iKey,
		InternalMsg: fmt.Sprintf("Unknown %s for alias '%s'", termType, keyFor), InternalCaller: CallerN(1)}
}

func NewUseKeysUseIndexesError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: E_USE_KEYS_USE_INDEXES, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term should not have USE KEYS or USE INDEX", termType), InternalCaller: CallerN(1)}
}

/* ---- END MOVED error numbers ----
   Please add new semantics error numbers in the 3000 number range above
*/
