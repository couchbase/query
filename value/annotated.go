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
	"reflect"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/util"
)

const _DEFAULT_ATTACHMENT_SIZE = 10

type AnnotatedValues []AnnotatedValue

var annotatedPool util.LocklessPool
var EMPTY_ANNOTATED_OBJECT AnnotatedValue

func init() {
	util.NewLocklessPool(&annotatedPool, func() unsafe.Pointer {
		return unsafe.Pointer(&annotatedValue{})
	})
	EMPTY_ANNOTATED_OBJECT = NewAnnotatedValue(map[string]interface{}{})
	EMPTY_ANNOTATED_OBJECT.(*annotatedValue).noRecycle = true
}

func newAnnotatedValue() *annotatedValue {
	rv := (*annotatedValue)(annotatedPool.Get())
	rv.refCnt = 1
	rv.bit = 0
	rv.self = false
	rv.noRecycle = false
	rv.sharedAnnotations = false
	return rv
}

func (this AnnotatedValues) Append(val AnnotatedValue, pool *AnnotatedPool) AnnotatedValues {
	if len(this) == cap(this) {
		avs := make(AnnotatedValues, len(this), len(this)<<1)
		copy(avs, this)
		pool.Put(this[0:0])
		this = avs
	}

	this = append(this, val)
	return this
}

/*
AnnotatedValue is a Value that can hold attachments and can hold data
from covering indexes.
*/
type AnnotatedValue interface {
	Value
	GetValue() Value
	Attachments() map[string]interface{}
	GetAttachment(key string) interface{}
	SetAttachment(key string, val interface{})
	RemoveAttachment(key string)
	GetId() interface{}
	SetId(id interface{})
	NewMeta() map[string]interface{}
	GetMeta() map[string]interface{}
	Covers() Value
	GetCover(key string) Value
	SetCover(key string, val Value)
	InheritCovers(val Value)
	CopyAnnotations(av AnnotatedValue)
	ShareAnnotations(av AnnotatedValue)
	Bit() uint8
	SetBit(b uint8)
	Self() bool
	SetSelf(s bool)
	SetProjection(proj Value)
	Original() AnnotatedValue
}

func NewAnnotatedValue(val interface{}) AnnotatedValue {
	switch val := val.(type) {
	case AnnotatedValue:
		return val
	case *ScopeValue:
		av := newAnnotatedValue()
		av.Value = val
		av.InheritCovers(val.Parent())
		return av
	case Value:
		av := newAnnotatedValue()
		av.Value = val
		return av
	default:
		av := newAnnotatedValue()
		av.Value = NewValue(val)
		return av
	}
}

type annotatedValue struct {
	Value
	attachments       map[string]interface{}
	sharedAnnotations bool
	meta              map[string]interface{}
	covers            Value
	bit               uint8
	id                interface{}
	refCnt            int32
	self              bool
	original          Value
	annotatedOrig     AnnotatedValue
	noRecycle         bool
}

func (this *annotatedValue) String() string {
	return this.Value.String()
}

func (this *annotatedValue) MarshalJSON() ([]byte, error) {
	return this.Value.MarshalJSON()
}

func (this *annotatedValue) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	return this.Value.WriteJSON(w, prefix, indent, fast)
}

func (this *annotatedValue) Copy() Value {
	rv := newAnnotatedValue()
	rv.Value = this.Value.Copy()
	copyAttachments(this.attachments, rv)
	copyMeta(this.meta, rv)
	rv.id = this.id
	rv.bit = this.bit
	if this.covers != nil {
		rv.covers = this.covers.Copy()
	}

	return rv
}

func (this *annotatedValue) CopyForUpdate() Value {
	rv := newAnnotatedValue()
	rv.Value = this.Value.CopyForUpdate()
	copyAttachments(this.attachments, rv)
	copyMeta(this.meta, rv)
	rv.id = this.id
	rv.bit = this.bit
	if this.covers != nil {
		rv.covers = this.covers.CopyForUpdate()
	}
	return rv
}

func (this *annotatedValue) SetField(field string, val interface{}) error {
	err := this.Value.SetField(field, val)
	if err == nil {
		v, ok := val.(Value)
		if ok {
			v.Track()
		}
	}
	return err
}

func (this *annotatedValue) GetValue() Value {
	return this.Value
}

func (this *annotatedValue) Attachments() map[string]interface{} {
	return this.attachments
}

func (this *annotatedValue) GetAttachment(key string) interface{} {
	if this.attachments != nil {
		return this.attachments[key]
	}

	return nil
}

func (this *annotatedValue) SetAttachment(key string, val interface{}) {
	if this.attachments == nil {
		this.attachments = make(map[string]interface{}, _DEFAULT_ATTACHMENT_SIZE)
	}

	this.attachments[key] = val
}

func (this *annotatedValue) RemoveAttachment(key string) {
	if this.attachments != nil {
		delete(this.attachments, key)
	}
}

func (this *annotatedValue) NewMeta() map[string]interface{} {
	if this.meta == nil {
		this.meta = make(map[string]interface{}, _DEFAULT_ATTACHMENT_SIZE)
	}
	return this.meta
}

func (this *annotatedValue) GetMeta() map[string]interface{} {
	if this.id != nil && this.meta == nil {
		this.meta = make(map[string]interface{}, _DEFAULT_ATTACHMENT_SIZE)
		this.meta["id"] = this.id
	}
	return this.meta
}

func (this *annotatedValue) Covers() Value {
	return this.covers
}

func (this *annotatedValue) GetCover(key string) Value {
	if this.covers != nil {
		rv, _ := this.covers.Field(key)
		return rv
	}

	return nil
}

func (this *annotatedValue) SetCover(key string, val Value) {
	if this.covers == nil {
		this.covers = NewScopeValue(make(map[string]interface{}), nil)
	}

	this.covers.SetField(key, val)
}

func (this *annotatedValue) InheritCovers(val Value) {
	if this.covers != nil || val == nil {
		return
	}

	switch val := val.(type) {
	case AnnotatedValue:
		this.covers = NewScopeValue(map[string]interface{}{}, val.Covers())
	case *ScopeValue:
		// Find the first ancestor that is not a ScopeValue
		var parent Value = val
		for p, ok := parent.(*ScopeValue); ok; p, ok = parent.(*ScopeValue) {
			parent = p.Parent()
			if parent == nil {
				return
			}
		}

		// Inherit covers from parent / ancestor
		if pv, ok := parent.(AnnotatedValue); ok && pv.Covers() != nil {
			this.covers = NewScopeValue(map[string]interface{}{}, pv.Covers())
		}
	}
}

// the erroneous case of two annotated values having the same attachments or meta but not sharing
// is not covered. You have been warned.
func (this *annotatedValue) CopyAnnotations(sv AnnotatedValue) {
	av := sv.(*annotatedValue) // we don't have any other annotated value implementation

	// no need to share with ourselves
	if av == this {
		return
	}

	amp := reflect.ValueOf(av.meta).Pointer()
	tmp := reflect.ValueOf(this.meta).Pointer()
	aap := reflect.ValueOf(av.attachments).Pointer()
	tap := reflect.ValueOf(this.attachments).Pointer()

	// get rid of previous attachments
	if this.sharedAnnotations {

		// if we are already sharing with the source no need to do anything
		if amp == tmp && aap == tap {
			return
		}
		this.attachments = nil
		this.meta = nil
	} else {

		// if the source is already sharing with us no need to do anything
		if av.sharedAnnotations && amp == tmp && aap == tap {
			return
		}
		for k := range this.attachments {
			delete(this.attachments, k)
		}
		for k := range this.meta {
			delete(this.meta, k)
		}
	}
	copyAttachments(av.attachments, this)
	copyMeta(av.meta, this)
	this.id = av.id
	if av.Covers() != nil {
		this.covers = av.covers.Copy()
	} else {
		this.covers = nil
	}
	this.sharedAnnotations = false
}

func (this *annotatedValue) ShareAnnotations(sv AnnotatedValue) {
	av := sv.(*annotatedValue) // we don't have any other annotated value implementation
	this.attachments = av.attachments
	this.meta = av.meta
	this.id = av.id
	if av.Covers() != nil {
		this.covers = av.covers
	} else {
		this.covers = nil
	}
	this.sharedAnnotations = true
}

func copyAttachments(source map[string]interface{}, dest *annotatedValue) {
	if len(source) == 0 {
		return
	}
	if dest.attachments == nil {
		dest.attachments = make(map[string]interface{}, len(source))
	}
	for k := range source {
		dest.attachments[k] = source[k]
	}
}

func copyMeta(source map[string]interface{}, dest *annotatedValue) {
	if len(source) == 0 {
		return
	}
	if dest.meta == nil {
		dest.meta = make(map[string]interface{}, len(source))
	}
	for k := range source {
		dest.meta[k] = source[k]
	}
}

func (this *annotatedValue) Bit() uint8 {
	return this.bit
}

func (this *annotatedValue) SetBit(b uint8) {
	this.bit = b
}

func (this *annotatedValue) Self() bool {
	return this.self
}

func (this *annotatedValue) SetSelf(s bool) {
	this.self = s
}

func (this *annotatedValue) GetId() interface{} {
	return this.id
}

func (this *annotatedValue) SetId(id interface{}) {
	this.id = id
	if this.meta != nil {
		this.meta["id"] = id
	}
}

func (this *annotatedValue) SetProjection(proj Value) {
	this.original = this.Value
	this.Value = proj
}

// Originals are not to be recycled
// For performance purposes, this can only be partially checked.
func (this *annotatedValue) Original() AnnotatedValue {
	if this.annotatedOrig != nil {
		return this.annotatedOrig
	}
	val := this.original
	if val == nil {
		return this
	}

	av := newAnnotatedValue()
	av.noRecycle = true
	switch val := val.(type) {
	case *annotatedValue:
		av.Value = val.Value
		av.covers = this.covers
		av.attachments = this.attachments
		av.meta = this.meta
	case *ScopeValue:
		av.Value = val
		av.covers = this.covers
		av.attachments = this.attachments
		av.meta = this.meta
	case Value:
		av.Value = val
	default:
		av.Value = NewValue(val)
	}
	this.annotatedOrig = av
	return av
}

func (this *annotatedValue) Track() {
	atomic.AddInt32(&this.refCnt, 1)
}

func (this *annotatedValue) Recycle() {

	// do no recycle if other scope values are using this value
	// or if this is an original document hanging off a projecton
	refcnt := atomic.AddInt32(&this.refCnt, -1)
	if refcnt > 0 || this.noRecycle {
		return
	}

	this.Value.Recycle()
	this.Value = nil
	if this.covers != nil {
		if !this.sharedAnnotations {
			this.covers.Recycle()
		}
		this.covers = nil
	}
	if this.annotatedOrig != nil {
		val := this.annotatedOrig.(*annotatedValue)
		val.covers = nil
		val.attachments = nil
		val.meta = nil
		annotatedPool.Put(unsafe.Pointer(val))
		this.annotatedOrig = nil
	}
	if this.original != nil {
		this.original.Recycle()
		this.original = nil
	}

	if this.sharedAnnotations {
		this.attachments = nil
		this.meta = nil
	} else {

		// pool the maps if they exist
		// these are optimized as maps clear by golang
		for k := range this.attachments {
			delete(this.attachments, k)
		}
		for k := range this.meta {
			delete(this.meta, k)
		}
	}
	this.sharedAnnotations = false
	annotatedPool.Put(unsafe.Pointer(this))
}
