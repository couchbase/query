//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
)

type SemChecker struct {
	expression.MapperBase
	semFlag  uint32
	stmtType string
}

func NewSemChecker(enterprise bool, stmtType string, txn bool) *SemChecker {
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
