//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"strings"

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

func (this IndexRefs) String() string {
	var buf strings.Builder
	this.writeSyntaxString(&buf)
	return buf.String()
}

func (this IndexRefs) writeSyntaxString(s *strings.Builder) {
	for i, idx := range this {
		if idx.name != "" {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(idx.name)
		}
		if idx.using == datastore.GSI || idx.using == datastore.FTS {
			s.WriteString(" using ")
			s.WriteString(string(idx.using))
		}
	}
}
