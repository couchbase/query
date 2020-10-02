//  Copieright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package value

import (
	"io"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/util"
)

const _DEFAULT_OBJECT_SIZE = 10

/*
ScopeValue provides alias scoping for subqueries, ranging, LETs,
projections, etc. It is a type struct that inherits Value and
has a parent Value.
*/
type ScopeValue struct {
	Value
	nested bool
	refCnt int32
	parent Value
}

var scopePool util.LocklessPool
var nestedPool util.LocklessPool

func init() {
	util.NewLocklessPool(&scopePool, func() unsafe.Pointer {
		return unsafe.Pointer(&ScopeValue{})
	})
	util.NewLocklessPool(&nestedPool, func() unsafe.Pointer {
		return unsafe.Pointer(&ScopeValue{Value: objectValue(make(map[string]interface{}, _DEFAULT_OBJECT_SIZE))})
	})
}

func newScopeValue(nested bool) *ScopeValue {
	var rv *ScopeValue

	if nested {
		rv = (*ScopeValue)(nestedPool.Get())
	} else {
		rv = (*ScopeValue)(scopePool.Get())
	}
	rv.refCnt = 1
	return rv
}

func NewScopeValue(val map[string]interface{}, parent Value) *ScopeValue {
	rv := newScopeValue(false)
	rv.Value = objectValue(val)
	rv.parent = parent
	if parent != nil {
		parent.Track()
	}
	rv.nested = false
	return rv
}

// shorthand for a scope value that is going to contain nested annotated values
func NewNestedScopeValue(parent Value) *ScopeValue {
	rv := newScopeValue(true)
	rv.Value = objectValue(make(map[string]interface{}))
	rv.parent = parent
	if parent != nil {
		parent.Track()
	}
	rv.nested = true
	return rv
}

func (this *ScopeValue) MarshalJSON() ([]byte, error) {
	val := objectValue(this.Fields())
	return val.MarshalJSON()
}

func (this *ScopeValue) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	if this.parent == nil {
		return this.Value.WriteJSON(w, prefix, indent, fast)
	}
	val := objectValue(this.Fields())
	return val.WriteJSON(w, prefix, indent, false)
}

func (this *ScopeValue) Copy() Value {
	rv := newScopeValue(false)
	rv.Value = this.Value.Copy()
	rv.parent = this.parent
	if this.parent != nil {
		this.parent.Track()
	}

	// counterintuitive but nested values copies share fields,
	// hence they don't recycle
	rv.nested = false
	return rv
}

func (this *ScopeValue) CopyForUpdate() Value {
	rv := newScopeValue(this.nested)
	rv.Value = this.Value.CopyForUpdate()
	rv.parent = this.parent
	if this.parent != nil {
		this.parent.Track()
	}
	rv.nested = this.nested
	return rv
}

func (this *ScopeValue) SetField(field string, val interface{}) error {
	err := this.Value.SetField(field, val)
	if err == nil {
		v, ok := val.(Value)
		if ok {
			v.Track()
		}
	}
	return err
}

/*
Implements scoping. Checks field of the value in the receiver
into result, and if valid returns the result.  If the parent
is not nil call Field on the parent and return that. Else a
missingField is returned. It searches itself and then the
parent for the input parameter field.
*/
func (this *ScopeValue) Field(field string) (Value, bool) {
	result, ok := this.Value.Field(field)
	if ok {
		return result, true
	}

	if this.parent != nil {
		return this.parent.Field(field)
	}

	return missingField(field), false
}

func (this *ScopeValue) Fields() map[string]interface{} {
	if this.parent == nil {
		return this.Value.Fields()
	}

	p := this.parent.Fields()
	v := this.Value.Fields()
	rv := make(map[string]interface{}, len(p)+len(v))

	for pf, pv := range p {
		rv[pf] = pv
	}

	for vf, vv := range v {
		rv[vf] = vv
	}

	return rv
}

func (this *ScopeValue) FieldNames(buffer []string) []string {
	return sortedNames(this.Fields(), buffer)
}

/*
Return the immediate scope.
*/
func (this *ScopeValue) GetValue() Value {
	return this.Value
}

/*
Return the parent scope.
*/
func (this *ScopeValue) Parent() Value {
	return this.parent
}

/*
Return the immediate map.
*/
func (this *ScopeValue) Map() map[string]interface{} {
	return this.Value.(objectValue)
}

func (this *ScopeValue) Track() {
	atomic.AddInt32(&this.refCnt, 1)
}

func (this *ScopeValue) Recycle() {

	// do no recycle if other scope values are using this value
	refcnt := atomic.AddInt32(&this.refCnt, -1)
	if refcnt > 0 {
		return
	}
	if this.parent != nil {
		this.parent.Recycle()
		this.parent = nil
	}
	if this.nested {
		fields := this.Value.(objectValue)
		for _, v := range fields {
			switch v := v.(type) {
			case *ScopeValue:
				v.Recycle()
			case *annotatedValue:
				v.Recycle()
			}
		}
	} else {
		this.Value.Recycle()
		this.Value = nil
	}
	scopePool.Put(unsafe.Pointer(this))
}
