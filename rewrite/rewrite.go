//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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
