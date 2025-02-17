//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

// This pool is currently used by only indexer during include column filtering
type StringValuePoolForIndex struct {
	size int
	pool []*stringValue
}

func NewStringValuePoolForIndex(size int) *StringValuePoolForIndex {
	p := &StringValuePoolForIndex{
		size: size,
		pool: make([]*stringValue, size),
	}
	for i := 0; i < size; i++ {
		p.pool[i] = new(stringValue)
	}
	return p
}

// Currently used by indexer when converting collateJSON encoded values to
// n1ql values for filtering on include columns. Indexer knows the position
// of the object being decoded to n1ql values - Hence, it uses the index of
// object to ensure the object is not shared with other decoded values
func (p *StringValuePoolForIndex) Get(index int, val string) Value {
	if index >= p.size {
		return NewValue(val)
	}
	sv := p.pool[index]
	*sv = stringValue(val)
	return sv
}

func (p *StringValuePoolForIndex) Reset(index int) {
	if index >= p.size {
		return
	}

	sv := p.pool[index]
	*sv = EMPTY_STRING_VALUE.(stringValue)
}
