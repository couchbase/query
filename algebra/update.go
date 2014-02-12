//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

type Update struct {
	bucket    *BucketRef  `json:"bucket"`
	keys      Expression  `json:"keys"`
	set       *Set        `json:"set"`
	unset     *Unset      `json:"unset"`
	where     Expression  `json:"where"`
	limit     Expression  `json:"limit"`
	returning ResultTerms `json:"returning"`
}

func NewUpdate(bucket *BucketRef, keys Expression, set *Set, unset *Unset,
	where, limit Expression, returning ResultTerms) *Update {
	return &Update{bucket, keys, set, unset, where, limit, returning}
}

func (this *Update) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitUpdate(this)
}

func (this *Update) BucketRef() *BucketRef {
	return this.bucket
}

func (this *Update) Keys() Expression {
	return this.keys
}

func (this *Update) Set() *Set {
	return this.set
}

func (this *Update) Unset() *Unset {
	return this.unset
}

func (this *Update) Where() Expression {
	return this.where
}

func (this *Update) Limit() Expression {
	return this.limit
}

func (this *Update) Returning() ResultTerms {
	return this.returning
}

type Set struct {
	paths []*SetPath
}

func NewSet(paths []*SetPath) *Set {
	return &Set{paths}
}

func (this *Set) Paths() []*SetPath {
	return this.paths
}

type Unset struct {
	paths []*UnsetPath
}

func NewUnset(paths []*UnsetPath) *Unset {
	return &Unset{paths}
}

func (this *Unset) Paths() []*UnsetPath {
	return this.paths
}

type SetPath struct {
	path    Path       `json:"path"`
	value   Expression `json:"value"`
	pathFor *PathFor   `json:"path-for"`
}

func (this *SetPath) Path() Path {
	return this.path
}

func (this *SetPath) Value() Expression {
	return this.value
}

func (this *SetPath) PathFor() *PathFor {
	return this.pathFor
}

func NewSetPath(path Path, value Expression, pathFor *PathFor) *SetPath {
	return &SetPath{path, value, pathFor}
}

type UnsetPath struct {
	path    Path     `json:"path"`
	pathFor *PathFor `json:"path-for"`
}

func NewUnsetPath(path Path, pathFor *PathFor) *UnsetPath {
	return &UnsetPath{path, pathFor}
}

func (this *UnsetPath) Path() Path {
	return this.path
}

func (this *UnsetPath) PathFor() *PathFor {
	return this.pathFor
}

type PathFor struct {
	bindings []*Binding
	when     Expression
}

func NewPathFor(bindings []*Binding, when Expression) *PathFor {
	return &PathFor{bindings, when}
}

func (this *PathFor) Bindings() []*Binding {
	return this.bindings
}

func (this *PathFor) When() Expression {
	return this.when
}
