//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
)

const (
	JOIN_HINT_NONE = iota
	USE_HASH_BUILD
	USE_HASH_PROBE
	USE_NL
)

var EMPTY_USE = NewUse(nil, nil, JOIN_HINT_NONE)

type JoinHint int

type Use struct {
	keys     expression.Expression
	indexes  IndexRefs
	joinHint JoinHint
}

func NewUse(keys expression.Expression, indexes IndexRefs, joinHint JoinHint) *Use {
	return &Use{keys, indexes, joinHint}
}

func (this *Use) Keys() expression.Expression {
	return this.keys
}

func (this *Use) SetKeys(keys expression.Expression) {
	this.keys = keys
}

func (this *Use) Indexes() IndexRefs {
	return this.indexes
}

func (this *Use) SetIndexes(indexes IndexRefs) {
	this.indexes = indexes
}

func (this *Use) JoinHint() JoinHint {
	return this.joinHint
}

func (this *Use) SetJoinHint(joinHint JoinHint) {
	this.joinHint = joinHint
}

// Hint Errors

const (
	HASH_JOIN_EE_ONLY     = "HASH JOIN is not supported in Community Edition"
	HASH_NEST_EE_ONLY     = "HASH NEST is not supported in Community Edition"
	USE_NL_NOT_FOLLOWED   = "USE NL hint cannot be followed"
	USE_HASH_NOT_FOLLOWED = "USE HASH hint cannot be followed"
)
