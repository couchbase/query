//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package semantics

import (
	"github.com/couchbase/query/expression"
)

/* semantic flags */
const (
	_SEM_ENTERPRISE = 1 << iota
	_SEM_WHERE
	_SEM_ON
	_SEM_TRANSACTION
	_SEM_PROJECTION
	_SEM_ADVISOR_FUNC
	_SEM_FROM
	_SEM_WITH_RECURSIVE
	_SEM_ORDERBY_VECTOR_DIST
)

type SemChecker struct {
	expression.MapperBase
	semFlag  uint32
	stmtType string
}

func newSemChecker(enterprise bool, stmtType string, txn bool) *SemChecker {
	rv := &SemChecker{}
	rv.SetMapper(rv)
	rv.stmtType = stmtType
	if enterprise {
		rv.setSemFlag(_SEM_ENTERPRISE)
	}

	if txn {
		rv.setSemFlag(_SEM_TRANSACTION)
	}

	return rv
}

func (this *SemChecker) setSemFlag(flag uint32) {
	this.semFlag |= flag
}

func (this *SemChecker) unsetSemFlag(flag uint32) {
	this.semFlag &^= flag
}

func (this *SemChecker) hasSemFlag(flag uint32) bool {
	return (this.semFlag & flag) != 0
}

func (this *SemChecker) StmtType() string {
	return this.stmtType
}
