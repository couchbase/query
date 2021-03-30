//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package value

/*
Sorter sorts an ARRAY Value in place. It implements sort.Interface.
*/
type Sorter struct {
	value Value
}

func NewSorter(value Value) *Sorter {
	return &Sorter{value: NewValue(value)}
}

func (this *Sorter) Len() int {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		return len(actual)
	default:
		return 0
	}
}

func (this *Sorter) Less(i, j int) bool {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		return NewValue(actual[i]).Collate(NewValue(actual[j])) < 0
	default:
		return false
	}
}

/*
Swap elements in index i and j.
*/
func (this *Sorter) Swap(i, j int) {
	actual := this.value.Actual()
	switch actual := actual.(type) {
	case []interface{}:
		actual[i], actual[j] = actual[j], actual[i]
	}
}
