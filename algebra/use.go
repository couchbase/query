//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
