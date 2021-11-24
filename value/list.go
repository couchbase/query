//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

/*
List implements a slice of Values as []value.Value.
*/
type List struct {
	list Values
}

func NewList(size int) *List {
	return &List{
		list: make(Values, 0, size),
	}
}

func (this *List) Add(item Value) {
	this.list = append(this.list, item)
}

func (this *List) AddAll(items Values) {
	for _, item := range items {
		this.Add(item)
	}
}

func (this *List) Len() int {
	return len(this.list)
}

func (this *List) ItemAt(pos int) Value {
	if pos >= 0 && pos < len(this.list) {
		return this.list[pos]
	}
	return nil
}

func (this *List) ReplaceAt(pos int, item Value) bool {
	if pos >= 0 && pos < len(this.list) {
		this.list[pos] = item
		return true
	}
	return false
}

func (this *List) Insert(pos, nlen int, item Value) {
	l := len(this.list)
	if nlen != 0 && l == nlen {
		copy(this.list[pos+1:], this.list[pos:l-1])
		this.list[pos] = item
	} else {
		this.Add(item)
		copy(this.list[pos+1:], this.list[pos:l])
		this.list[pos] = item
	}
}

func (this *List) Values() []Value {
	return this.list
}

func (this *List) Clear() {
	this.list = nil
}

func (this *List) Copy() *List {
	rv := make(Values, len(this.list))
	for k, v := range this.list {
		rv[k] = v
	}
	return &List{rv}
}

func (this *List) Union(other *List) {
	this.AddAll(other.Values())
}
