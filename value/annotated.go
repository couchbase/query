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
)

type AnnotatedValues []AnnotatedValue

func (this AnnotatedValues) Append(val AnnotatedValue, pool *AnnotatedPool) AnnotatedValues {
	if len(this) == cap(this) {
		avs := make(AnnotatedValues, len(this), len(this)<<1)
		copy(avs, this)
		pool.Put(this)
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
	GetId() interface{}
	SetId(id interface{})
	Covers() Value
	GetCover(key string) Value
	SetCover(key string, val Value)
	InheritCovers(val Value)
	SetAnnotations(av AnnotatedValue)
	Bit() uint8
	SetBit(b uint8)
}

func NewAnnotatedValue(val interface{}) AnnotatedValue {
	switch val := val.(type) {
	case AnnotatedValue:
		return val
	case *ScopeValue:
		av := &annotatedValue{}
		av.Value = val
		av.InheritCovers(val.Parent())
		return av
	case Value:
		av := &annotatedValue{}
		av.Value = val
		return av
	default:
		av := &annotatedValue{}
		av.Value = NewValue(val)
		return av
	}
}

type annotatedValue struct {
	Value
	attachments map[string]interface{}
	covers      Value
	bit         uint8
	id          interface{}
}

func (this *annotatedValue) String() string {
	return this.Value.String()
}

func (this *annotatedValue) MarshalJSON() ([]byte, error) {
	return this.Value.MarshalJSON()
}

func (this *annotatedValue) WriteJSON(w io.Writer, prefix, indent string) error {
	return this.Value.WriteJSON(w, prefix, indent)
}

func (this *annotatedValue) Copy() Value {
	rv := &annotatedValue{}
	rv.Value = this.Value.Copy()
	rv.attachments = copyMap(this.attachments, self)
	rv.covers = this.covers
	rv.bit = this.bit
	if this.covers != nil {
		rv.covers = this.covers.Copy()
	}

	return rv
}

func (this *annotatedValue) CopyForUpdate() Value {
	rv := &annotatedValue{}
	rv.Value = this.Value.CopyForUpdate()
	rv.attachments = copyMap(this.attachments, self)
	rv.covers = this.covers
	rv.bit = this.bit
	return rv
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
		this.attachments = make(map[string]interface{})
	}

	this.attachments[key] = val
}

func (this *annotatedValue) RemoveAttachment(key string) {
	if this.attachments != nil {
		delete(this.attachments, key)
	}
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

func (this *annotatedValue) SetAnnotations(av AnnotatedValue) {
	this.attachments = av.Attachments()
	this.covers = av.Covers()
}

func (this *annotatedValue) Bit() uint8 {
	return this.bit
}

func (this *annotatedValue) SetBit(b uint8) {
	this.bit = b
}

func (this *annotatedValue) GetId() interface{} {
	return this.id
}

func (this *annotatedValue) SetId(id interface{}) {
	this.id = id
}

func (this *annotatedValue) Recycle() {
	this.Value.Recycle()
	this.Value = nil
	this.attachments = nil
	if this.covers != nil {
		this.covers.Recycle()
		this.covers = nil
	}
	this.bit = 0
}
