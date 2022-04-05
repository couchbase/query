//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"github.com/couchbase/query/expression"
)

type JoinHint int32

const (
	JOIN_HINT_NONE = JoinHint(iota)
	USE_HASH_BUILD
	USE_HASH_PROBE
	USE_HASH_EITHER
	USE_NL
	NO_USE_HASH
	NO_USE_NL
)

var EMPTY_USE = NewUse(nil, nil, JOIN_HINT_NONE)

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

func PreferHash(joinHint JoinHint) bool {
	switch joinHint {
	case USE_HASH_BUILD, USE_HASH_PROBE, USE_HASH_EITHER:
		return true
	case NO_USE_NL:
		return true
	}
	return false
}

func PreferNL(joinHint JoinHint) bool {
	switch joinHint {
	case USE_NL:
		return true
	case NO_USE_HASH:
		return true
	}
	return false
}
