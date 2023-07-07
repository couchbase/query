//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"io"
	"reflect"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/logging"
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
	rv.seenCnt = 0
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
	Stash() int32
	Restore(lvl int32)
	Seen() bool
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
	ResetCovers(parent Value)
	InheritCovers(val Value)
	CopyAnnotations(av AnnotatedValue)
	ShareAnnotations(av AnnotatedValue)
	Bit() uint8
	SetBit(b uint8)
	Self() bool
	SetSelf(s bool)
	SetProjection(proj Value)
	Original() AnnotatedValue
	RefCnt() int32
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
	seenCnt           int32
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
	var err error
	if val == this {
		selfRef := (*annotatedValueSelfReference)(val.(*annotatedValue))
		err = this.Value.SetField(field, selfRef)
	} else {
		err = this.Value.SetField(field, val)
	}
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

func (this *annotatedValue) ResetCovers(parent Value) {
	if this.covers == nil {
		return
	} else if pav, ok := parent.(*annotatedValue); ok && pav != nil && pav.covers != nil {
		for k, _ := range this.covers.Fields() {
			if _, ok := pav.covers.Field(k); !ok {
				this.covers.UnsetField(k)
			}
		}
	} else {
		this.covers = nil
	}
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

func (this *annotatedValue) Stash() int32 {
	return atomic.AddInt32(&this.refCnt, 1) - 1
}

func (this *annotatedValue) Restore(lvl int32) {
	atomic.StoreInt32(&this.refCnt, lvl)
}

func (this *annotatedValue) Seen() bool {
	return atomic.AddInt32(&this.seenCnt, 1) > 1
}

func (this *annotatedValue) Track() {
	atomic.AddInt32(&this.refCnt, 1)
}

func (this *annotatedValue) Recycle() {
	this.recycle(-1)
}

func (this *annotatedValue) RefCnt() int32 {
	return atomic.LoadInt32(&this.refCnt)
}

func (this *annotatedValue) recycle(lvl int32) {

	// do no recycle if other scope values are using this value
	// or if this is an original document hanging off a projecton
	refcnt := atomic.AddInt32(&this.refCnt, lvl)
	if refcnt > 0 || this.noRecycle {
		return
	}
	if refcnt < 0 {

		// TODO enable
		// panic("annotated value already recycled")
		logging.Infof("annotated value already recycled")
		return
	}

	if this.Value != nil {
		this.Value.Recycle()
		this.Value = nil
	}
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
	this.id = nil
	annotatedPool.Put(unsafe.Pointer(this))
}

type annotatedValueSelfReference annotatedValue

func (this *annotatedValueSelfReference) Size() uint64 {
	return 0
}

func (this *annotatedValueSelfReference) String() string {
	return ""
}

func (this *annotatedValueSelfReference) ToString() string {
	return ""
}

func (this *annotatedValueSelfReference) MarshalJSON() ([]byte, error) {
	return []byte(nil), nil
}

func (this *annotatedValueSelfReference) WriteJSON(w io.Writer, prefix, indent string, fast bool) error {
	return nil
}

func (this *annotatedValueSelfReference) Equals(other Value) Value {
	return FALSE_VALUE
}

func (this *annotatedValueSelfReference) EquivalentTo(other Value) bool {
	return false
}

func (this *annotatedValueSelfReference) Compare(other Value) Value {
	return other
}

func (this *annotatedValueSelfReference) Collate(other Value) int {
	return 0
}

func (this *annotatedValueSelfReference) Descendants(buffer []interface{}) []interface{} {
	return buffer
}

func (this *annotatedValueSelfReference) DescendantPairs(buffer []util.IPair) []util.IPair {
	return buffer
}

func (this *annotatedValueSelfReference) Successor() Value {
	return _SMALL_OBJECT_VALUE
}

func (this *annotatedValueSelfReference) Tokens(set *Set, options Value) *Set {
	return set
}

func (this *annotatedValueSelfReference) ContainsToken(token, options Value) bool {
	return false
}

func (this *annotatedValueSelfReference) ContainsMatchingToken(matcher MatchFunc, options Value) bool {
	return false
}

func (this *annotatedValueSelfReference) Stash() int32 {
	return (*annotatedValue)(this).Stash()
}

func (this *annotatedValueSelfReference) Restore(lvl int32) {
	(*annotatedValue)(this).Restore(lvl)
}

func (this *annotatedValueSelfReference) Seen() bool {
	return (*annotatedValue)(this).Seen()
}

func (this *annotatedValueSelfReference) Track() {
	// deliberately empty
}

func (this *annotatedValueSelfReference) Recycle() {
	// deliberately empty
}

func (this *annotatedValueSelfReference) RefCnt() int32 {
	return 0
}

func (this *annotatedValueSelfReference) GetValue() Value {
	return (*annotatedValue)(this).GetValue()
}

func (this *annotatedValueSelfReference) Attachments() map[string]interface{} {
	return (*annotatedValue)(this).Attachments()
}

func (this *annotatedValueSelfReference) GetAttachment(key string) interface{} {
	return (*annotatedValue)(this).GetAttachment(key)
}

func (this *annotatedValueSelfReference) SetAttachment(key string, val interface{}) {
	(*annotatedValue)(this).SetAttachment(key, val)
}

func (this *annotatedValueSelfReference) RemoveAttachment(key string) {
	(*annotatedValue)(this).RemoveAttachment(key)
}

func (this *annotatedValueSelfReference) GetId() interface{} {
	return (*annotatedValue)(this).GetId()
}

func (this *annotatedValueSelfReference) SetId(id interface{}) {
	(*annotatedValue)(this).SetId(id)
}

func (this *annotatedValueSelfReference) NewMeta() map[string]interface{} {
	return (*annotatedValue)(this).NewMeta()
}

func (this *annotatedValueSelfReference) GetMeta() map[string]interface{} {
	return (*annotatedValue)(this).GetMeta()
}

func (this *annotatedValueSelfReference) Covers() Value {
	return (*annotatedValue)(this).Covers()
}

func (this *annotatedValueSelfReference) GetCover(key string) Value {
	return (*annotatedValue)(this).GetCover(key)
}

func (this *annotatedValueSelfReference) SetCover(key string, val Value) {
	(*annotatedValue)(this).SetCover(key, val)
}

func (this *annotatedValueSelfReference) ResetCovers(parent Value) {
	(*annotatedValue)(this).ResetCovers(parent)
}

func (this *annotatedValueSelfReference) InheritCovers(val Value) {
	(*annotatedValue)(this).InheritCovers(val)
}

func (this *annotatedValueSelfReference) CopyAnnotations(av AnnotatedValue) {
	(*annotatedValue)(this).CopyAnnotations(av)
}

func (this *annotatedValueSelfReference) ShareAnnotations(av AnnotatedValue) {
	(*annotatedValue)(this).ShareAnnotations(av)
}

func (this *annotatedValueSelfReference) Bit() uint8 {
	return (*annotatedValue)(this).Bit()
}

func (this *annotatedValueSelfReference) SetBit(b uint8) {
	(*annotatedValue)(this).SetBit(b)
}

func (this *annotatedValueSelfReference) Self() bool {
	return (*annotatedValue)(this).Self()
}

func (this *annotatedValueSelfReference) SetSelf(s bool) {
	(*annotatedValue)(this).SetSelf(s)
}

func (this *annotatedValueSelfReference) SetProjection(proj Value) {
	(*annotatedValue)(this).SetProjection(proj)
}

func (this *annotatedValueSelfReference) Original() AnnotatedValue {
	return (*annotatedValue)(this).Original()
}
