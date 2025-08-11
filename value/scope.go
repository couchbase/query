//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"encoding/binary"
	"io"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const _DEFAULT_OBJECT_SIZE = 2

/*
ScopeValue provides alias scoping for subqueries, ranging, LETs,
projections, etc. It is a type struct that inherits Value and
has a parent Value.
*/
type ScopeValue struct {
	Value
	parent Value
	refCnt int32
	nested bool
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
	rv.Value = newObjectValue(val)
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

func (this *ScopeValue) WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) error {
	if this.parent == nil {
		return this.Value.WriteXML(order, w, prefix, indent, fast)
	}
	val := objectValue(this.Fields())
	return val.WriteXML(order, w, prefix, indent, false)
}

func (this *ScopeValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	if this.parent == nil {
		return this.Value.WriteJSON(order, w, prefix, indent, fast)
	}
	val := objectValue(this.Fields())
	return val.WriteJSON(order, w, prefix, indent, false)
}

func (this *ScopeValue) WriteSpill(w io.Writer, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}
	buf = buf[:6]
	buf[0] = _SPILL_TYPE_VALUE_SCOPE
	if this.nested {
		buf[1] = 1
	} else {
		buf[1] = 0
	}
	binary.BigEndian.PutUint32(buf[2:], uint32(this.refCnt))
	_, err := w.Write(buf)
	if err == nil {
		err = writeSpillValue(w, this.Value, buf)
		if err == nil {
			err = writeSpillValue(w, this.parent, buf)
		}
	}
	if free {
		_SPILL_POOL.Put(buf)
	}
	return err
}

func (this *ScopeValue) ReadSpill(r io.Reader, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}
	buf = buf[:5]
	_, err := r.Read(buf)
	if err == nil {
		this.nested = (buf[0] != 0)
		this.refCnt = int32(binary.BigEndian.Uint32(buf[1:]))
		var v interface{}
		v, err = readSpillValue(r, buf)
		if err == nil && v != nil {
			this.Value = v.(Value)
		} else {
			this.Value = nil
		}
		if err == nil {
			v, err = readSpillValue(r, buf)
			if err == nil && v != nil {
				this.parent = v.(Value)
			} else {
				this.parent = nil
			}
		}
	}
	if free {
		_SPILL_POOL.Put(buf)
	}
	return err
}

func (this *ScopeValue) Copy() Value {
	rv := newScopeValue(false)
	if this.Value != nil {
		switch v := this.Value.(type) {
		case copiedObjectValue:
			// already know not to recycle contents
		case objectValue:
			// replace with a copied object value so we know not to recycle the contents as they're now shared
			this.Value = copiedObjectValue{objectValue: v}
		default:
		}
		rv.Value = this.Value.Copy()
	}
	rv.parent = this.parent
	if this.parent != nil {
		this.parent.Track()
	}

	rv.nested = false
	return rv
}

func (this *ScopeValue) CopyForUpdate() Value {
	rv := newScopeValue(this.nested)
	rv.Value = this.Value.CopyForUpdate()
	if this.parent != nil {
		rv.parent = this.parent.Copy()
		this.parent.Track()
	}
	rv.nested = this.nested
	if this.nested {
		fields := rv.Value.(objectValue)
		for _, v := range fields {
			switch v := v.(type) {
			case *ScopeValue:
				if v.RefCnt() == 1 {
					v.Track()
				}
			case *annotatedValue:
				if v.RefCnt() == 1 {
					v.Track()
				}
			}
		}
	}
	return rv
}

func (this *ScopeValue) SetField(field string, val interface{}) error {
	err := this.Value.SetField(field, val)
	if err == nil {
		switch this.Value.(type) {
		case objectValue:
			// Track() already done, no-op
		default:
			v, ok := val.(Value)
			if ok {
				v.Track()
			}
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

func (this *ScopeValue) ResetParent(newParent Value) Value {
	var p Value
	p, this.parent = this.parent, newParent
	if this.parent != nil {
		this.parent.Track()
	}
	if p != nil {
		p.Recycle()
	}
	return p
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
	this.recycle(-1)
}

func (this *ScopeValue) recycle(lvl int32) {

	// do no recycle if other scope values are using this value
	refcnt := atomic.AddInt32(&this.refCnt, lvl)
	if refcnt > 0 {
		return
	}
	if refcnt < 0 {

		// TODO enable
		// panic("scope value already recycled")
		logging.Infof("scope value already recycled")
		return
	}
	if this.parent != nil {
		this.parent.Recycle()
		this.parent = nil
	}
	if this.nested {
		// we must respect that the content type may be restricting recycling
		var fields objectValue
		switch v := this.Value.(type) {
		case copiedObjectValue:
			fields = v.objectValue
		case objectValue:
			fields = v
			// do this here to avoid calling recycle on values where it is a no-op
			for _, v := range fields {
				switch v := v.(type) {
				case *ScopeValue:
					v.recycle(-2)
				case *annotatedValue:
					v.recycle(-2)
				}
			}
		}

		// pool the map
		clear(fields)
		this.Value = fields
		nestedPool.Put(unsafe.Pointer(this))
	} else {
		this.Value.Recycle()
		this.Value = nil
		scopePool.Put(unsafe.Pointer(this))
	}
}

func (this *ScopeValue) RefCnt() int32 {
	return this.refCnt
}

func (this *ScopeValue) Size() uint64 {
	// parent values are a possibly unique case in that they can be large but are often shared a lot so counting more than once
	// really bloats the quota figures, so we don't count them at all
	return this.Value.Size()
}
