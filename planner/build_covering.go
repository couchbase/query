//  Copyright 2021-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package planner

import (
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

// Structure store all the covered index info until best index has decided.

type coveringEntry struct {
	idxEntry         *indexEntry
	rootUnnest       *algebra.Unnest
	leafUnnest       *algebra.Unnest
	covers           expression.Covers
	filterCovers     map[*expression.Cover]value.Value
	coveredUnnests   map[*algebra.Unnest]bool
	implicitAny      bool
	implcitIndexProj map[int]bool
	indexKeys        datastore.IndexKeys
}
