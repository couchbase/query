//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
		return &err{level: EXCEPTION, ICode: 3100, IKey: "semantics_error", ICause: e,
			InternalMsg: msg, InternalCaller: CallerN(1)}
	}
}

const JOIN_NEST_NO_JOIN_HINT = 3110

func NewJoinNestNoJoinHintError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: JOIN_NEST_NO_JOIN_HINT, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have join hint (USE HASH or USE NL).", op, alias),
		InternalCaller: CallerN(1)}
}

const JOIN_NEST_NO_USE_KEYS = 3120

func NewJoinNestNoUseKeysError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: JOIN_NEST_NO_USE_KEYS, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have USE KEYS.", op, alias),
		InternalCaller: CallerN(1)}
}

const JOIN_NEST_NO_USE_INDEX = 3130

func NewJoinNestNoUseIndexError(op string, alias string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: JOIN_NEST_NO_USE_INDEX, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s on %s cannot have USE INDEX.", op, alias),
		InternalCaller: CallerN(1)}
}

// Error code 3140 is retired. Do not reuse.
// const SUBQ_TERM_NO_CORRELATION = 3140

const MERGE_INSERT_NO_KEY = 3150

func NewMergeInsertNoKeyError() Error {
	return &err{level: EXCEPTION, ICode: MERGE_INSERT_NO_KEY, IKey: "semantics.visit_merge.merge_insert_no_key",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have document key specification in INSERT action."),
		InternalCaller: CallerN(1)}
}

const MERGE_INSERT_MISSING_KEY = 3160

func NewMergeInsertMissingKeyError() Error {
	return &err{level: EXCEPTION, ICode: MERGE_INSERT_MISSING_KEY, IKey: "semantics.visit_merge.merge_insert_missing_key",
		InternalMsg:    fmt.Sprintf("MERGE with ON clause must have document key specification in INSERT action."),
		InternalCaller: CallerN(1)}
}

const MERGE_MISSING_SOURCE = 3170

func NewMergeMissingSourceError() Error {
	return &err{level: EXCEPTION, ICode: MERGE_MISSING_SOURCE, IKey: "semantics.visit_merge.merge_missing_source",
		InternalMsg:    fmt.Sprintf("MERGE is missing source."),
		InternalCaller: CallerN(1)}
}

const MERGE_NO_INDEX_HINT = 3180

func NewMergeNoIndexHintError() Error {
	return &err{level: EXCEPTION, ICode: MERGE_NO_INDEX_HINT, IKey: "semantics.visit_merge.merge_no_index_hint",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have USE INDEX hint specified on target."),
		InternalCaller: CallerN(1)}
}

const MERGE_NO_JOIN_HINT = 3190

func NewMergeNoJoinHintError() Error {
	return &err{level: EXCEPTION, ICode: MERGE_NO_JOIN_HINT, IKey: "semantics.visit_merge.merge_no_join_hint",
		InternalMsg:    fmt.Sprintf("MERGE with ON KEY clause cannot have join hint specified on source."),
		InternalCaller: CallerN(1)}
}

const ANSI_MIXED_JOIN = 3200

func NewMixedJoinError(op1 string, alias1 string, op2 string, alias2 string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: ANSI_MIXED_JOIN, IKey: iKey,
		InternalMsg:    fmt.Sprintf("Cannot mix %s on %s with %s on %s.", op1, alias1, op2, alias2),
		InternalCaller: CallerN(1)}
}

// Error code 3210 is retired. Do not reuse.
// const ANSI_KEYSPACE_ONLY = 3210

const WINDOW_SEMANTIC_ERROR = 3220

func NewWindowSemanticError(fname, wclause, cause, iKey string) Error {
	return &err{level: EXCEPTION, ICode: WINDOW_SEMANTIC_ERROR, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s window function %s%s", fname, wclause, cause),
		InternalCaller: CallerN(1)}
}

const ENTERPRISE_FEATURE = 3230

func NewEnterpriseFeature(opmsg, iKey string) Error {
	return &err{level: EXCEPTION, ICode: ENTERPRISE_FEATURE, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s is enterprise level feature.", opmsg),
		InternalCaller: CallerN(1)}
}

// Error code 3240 is retired. Do not reuse.
// const UNSUPPORTED_PATH_TYPE = 3240

const ADVISE_UNSUPPORTED_STMT = 3250

func NewAdviseUnsupportedStmtError(iKey string) Error {
	return &err{level: EXCEPTION, ICode: ADVISE_UNSUPPORTED_STMT, IKey: iKey,
		InternalMsg: fmt.Sprintf("Advise supports SELECT, MERGE, UPDATE and DELETE statements only."), InternalCaller: CallerN(1)}
}

const ADVISOR_PROJECTION_ONLY = 3255

func NewAdvisorProjOnly() Error {
	return &err{level: EXCEPTION, ICode: ADVISOR_PROJECTION_ONLY, IKey: "semantics_advisor_function",
		InternalMsg: fmt.Sprintf("Advisor function is only allowed in projection clause."), InternalCaller: CallerN(1)}
}

const ADVISOR_NO_FROM = 3256

func NewAdvisorNoFrom() Error {
	return &err{level: EXCEPTION, ICode: ADVISOR_NO_FROM, IKey: "semantics_advisor_function",
		InternalMsg: fmt.Sprintf("FROM clause is not allowed when Advisor function is present in projection clause."), InternalCaller: CallerN(1)}
}

const MH_DP_ONLY_FEATURE = 3260

func NewMHDPOnlyFeature(what, iKey string) Error {
	return &err{level: EXCEPTION, ICode: MH_DP_ONLY_FEATURE, IKey: iKey,
		InternalMsg:    fmt.Sprintf("%s is only supported in Developer Preview Mode.", what),
		InternalCaller: CallerN(1)}
}

func NewMissingUseKeysError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: 3261, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term must have USE KEYS", termType), InternalCaller: CallerN(1)}
}

func NewHasUseIndexesError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: 3262, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term should not have USE INDEX", termType), InternalCaller: CallerN(1)}
}

const UPDATE_STAT_INDEX_TYPE = 3270

func NewUpdateStatInvalidIndexTypeError() Error {
	return &err{level: EXCEPTION, ICode: UPDATE_STAT_INDEX_TYPE, IKey: "semantics_update_statistics",
		InternalMsg: "UPDATE STATISTICS (ANALYZE) supports GSI indexes only for INDEX option.", InternalCaller: CallerN(1)}
}

/* ---- BEGIN MOVED error numbers ----
   The following error numbers (in the 4000 range) originally reside in plan.go (before the introduction of the semantics package)
   although they are semantic errors. They are moved from plan.go to semantics.go but their original error numbers are kept.
*/
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

const UNKNOWN_FOR = 4025

func NewUnknownForError(termType string, keyFor string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: UNKNOWN_FOR, IKey: iKey,
		InternalMsg: fmt.Sprintf("Unknown %s for alias %s", termType, keyFor), InternalCaller: CallerN(1)}
}

const EXPR_NO_USE_KEYS_INDEX = 4110

func NewUseKeysUseIndexesError(termType string, iKey string) Error {
	return &err{level: EXCEPTION, ICode: EXPR_NO_USE_KEYS_INDEX, IKey: iKey,
		InternalMsg: fmt.Sprintf("%s term should not have USE KEYS or USE INDEX", termType), InternalCaller: CallerN(1)}
}

/* ---- END MOVED error numbers ----
   Please add new semantics error numbers in the 3000 number range above
*/
