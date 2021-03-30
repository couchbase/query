//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package algebra

import (
	"github.com/couchbase/query/datastore"
)

type IndexRefs []*IndexRef

type IndexRef struct {
	name  string              `json:"name"`
	using datastore.IndexType `json:"using"`
}

func NewIndexRef(name string, using datastore.IndexType) *IndexRef {
	return &IndexRef{name, using}
}

func (this *IndexRef) Name() string {
	return this.name
}

func (this *IndexRef) Using() datastore.IndexType {
	return this.using
}
