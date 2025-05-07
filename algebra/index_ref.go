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
	for _, i := range this {
		if i.name != "" {
			buf.WriteString(" ")
			buf.WriteString(i.name)
		}
		if i.using == datastore.GSI || i.using == datastore.FTS {
			buf.WriteString(" using ")
			buf.WriteString(string(i.using))
		}
		buf.WriteString(",")
	}
	s := buf.String()
	if len(s) > 0 {
		s = s[1:]
	}
	if len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}
