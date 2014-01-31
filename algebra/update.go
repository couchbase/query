//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	_ "fmt"
	_ "github.com/couchbaselabs/query/value"
)

type Update struct {
	bucket    *BucketRef     `json:"bucket"`
	keys      Expression     `json:"keys"`
	set       *Set           `json:"set"`
	unset     *Unset         `json:"unset"`
	where     Expression     `json:"where"`
	limit     Expression     `json:"limit"`
	returning ResultTermList `json:"returning"`
}

type Set struct {
	paths []SetPath
}

type Unset struct {
	paths []UnsetPath
}

type SetPath struct {
	path    Path       `json:"path"`
	value   Expression `json:"value"`
	pathFor *PathFor   `json:"path-for"`
}

type UnsetPath struct {
	path    Path     `json:"path"`
	pathFor *PathFor `json:"path-for"`
}

type PathForBinding struct {
	variable string
	expr     Expression
}

type PathFor struct {
	bindings []*PathForBinding
	when     Expression
}

func NewUpdate(bucket *BucketRef, keys Expression, set *Set, unset *Unset,
	where, limit Expression, returning ResultTermList) *Update {
	return &Update{bucket, keys, set, unset, where, limit, returning}
}

func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}

func NewSet(paths []SetPath) *Set {
	return &Set{paths}
}

func NewUnset(paths []UnsetPath) *Unset {
	return &Unset{paths}
}
