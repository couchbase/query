//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package rewrite

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/expression"
)

const (
	REWRITE_PHASE1 = 1 << iota
)

type Rewrite struct {
	expression.MapperBase

	rewriteFlag uint32
	windowTerms algebra.WindowTerms
}

func NewRewrite(flags uint32) *Rewrite {
	rv := &Rewrite{rewriteFlag: flags}
	rv.SetMapper(rv)
	return rv
}

func (this *Rewrite) setRewriteFlag(flag uint32) {
	this.rewriteFlag |= flag
}

func (this *Rewrite) unsetRewriteFlag(flag uint32) {
	this.rewriteFlag &^= flag
}

func (this *Rewrite) hasRewriteFlag(flag uint32) bool {
	return (this.rewriteFlag & flag) != 0
}
