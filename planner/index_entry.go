//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package planner

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
)

type indexEntry struct {
	index      datastore.Index
	keys       expression.Expressions
	sargKeys   expression.Expressions
	minKeys    int
	sumKeys    int
	cond       expression.Expression
	origCond   expression.Expression
	spans      SargSpans
	exactSpans bool
}

func (this *indexEntry) Copy() *indexEntry {
	rv := &indexEntry{
		index:      this.index,
		keys:       expression.CopyExpressions(this.keys),
		sargKeys:   expression.CopyExpressions(this.sargKeys),
		minKeys:    this.minKeys,
		sumKeys:    this.sumKeys,
		cond:       expression.Copy(this.cond),
		origCond:   expression.Copy(this.origCond),
		spans:      CopySpans(this.spans),
		exactSpans: this.exactSpans,
	}

	return rv
}
