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
	"reflect"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const _DEFAULT_ATTACHMENT_SIZE = 6

type AnnotatedValues []AnnotatedValue

var annotatedPool util.LocklessPool
var EMPTY_ANNOTATED_OBJECT AnnotatedValue

var allocatedValuesCount atomic.AlignedInt64

func init() {
	util.NewLocklessPool(&annotatedPool, func() unsafe.Pointer {
		atomic.AddInt64(&allocatedValuesCount, 1)
		rv := &annotatedValue{}
		return unsafe.Pointer(rv)
	})
	av := newAnnotatedValue()
	av.flags |= _NO_RECYCLE
	av.refCnt = 2 // should ensure it is copied and not directly modified by LET etc.
	av.Value = NewValue(map[string]interface{}{})
	EMPTY_ANNOTATED_OBJECT = av
}

func AllocatedValuesCount() int64 {
	return atomic.LoadInt64(&allocatedValuesCount)
}

func newAnnotatedValue() *annotatedValue {
	rv := (*annotatedValue)(annotatedPool.Get())
	rv.refCnt = 1
	rv.seenCnt = 0
	rv.bit = 0
	rv.flags = 0
	rv.cachedSize = 0
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
	SetValue(Value)
	GetValue() Value
	GetParent() Value
	SetParent(Value) Value
	Attachments() map[string]interface{}
	GetAttachment(key string) interface{}
	SetAttachment(key string, val interface{})
	RemoveAttachment(key string)
	ResetAttachments()
	GetId() interface{}
	SetId(id interface{})
	NewMeta() map[string]interface{}
	GetMeta() map[string]interface{}
	SetMeta(meta map[string]interface{})
	ResetMeta()
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
	SetProjection(proj Value, order []string)
	ProjectionOrder() []string
	Original() AnnotatedValue
	RefCnt() int32
	ResetOriginal()
	RecalculateSize() uint64
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

const (
	_SHARED_ANNOTATIONS = 1 << iota
	_NO_RECYCLE
	_SELF
	_HAS_SELF_REF
)

type annotatedValue struct {
	Value
	attachments     map[string]interface{}
	meta            map[string]interface{}
	covers          Value
	original        Value
	annotatedOrig   AnnotatedValue
	projectionOrder []string // transient: no need to spill
	refCnt          int32
	seenCnt         int32
	cachedSize      uint32 // do not spill
	bit             uint8
	flags           uint8
}

func (this *annotatedValue) sharedAnnotations() bool {
	return this.flags&_SHARED_ANNOTATIONS != 0
}

func (this *annotatedValue) noRecycle() bool {
	return this.flags&_NO_RECYCLE != 0
}

func (this *annotatedValue) self() bool {
	return this.flags&_SELF != 0
}

func (this *annotatedValue) hasSelfRef() bool {
	return this.flags&_HAS_SELF_REF != 0
}

func (this *annotatedValue) String() string {
	return this.Value.String()
}

func (this *annotatedValue) MarshalJSON() ([]byte, error) {
	return this.Value.MarshalJSON()
}

func (this *annotatedValue) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return this.Value.WriteJSON(order, w, prefix, indent, fast)
}

func (this *annotatedValue) WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return this.Value.WriteXML(order, w, prefix, indent, fast)
}

func (this *annotatedValue) Copy() Value {
	rv := newAnnotatedValue()
	rv.Value = this.Value.Copy()
	rv.updateSelfReferences(this)
	copyAttachments(this.attachments, rv)
	copyMeta(this.meta, rv)
	rv.bit = this.bit
	if this.covers != nil {
		rv.covers = this.covers.Copy()
	}

	return rv
}

func (this *annotatedValue) CopyForUpdate() Value {
	rv := newAnnotatedValue()
	rv.Value = this.Value.CopyForUpdate()
	rv.updateSelfReferences(this)
	copyAttachments(this.attachments, rv)
	copyMeta(this.meta, rv)
	rv.bit = this.bit
	if this.covers != nil {
		rv.covers = this.covers.CopyForUpdate()
	}
	return rv
}

func (this *annotatedValue) updateSelfReferences(orig *annotatedValue) {
	if !orig.hasSelfRef() {
		return
	}
	osr := (*annotatedValueSelfReference)(orig)
	nsr := (*annotatedValueSelfReference)(this)
	flds := this.Value.Fields()
	for k, v := range flds {
		if avsr, ok := v.(*annotatedValueSelfReference); ok && avsr == osr {
			flds[k] = nsr
		}
	}
	this.flags |= _HAS_SELF_REF
}

func (this *annotatedValue) UnsetField(field string) error {
	return this.Value.UnsetField(field)
}

func (this *annotatedValue) SetField(field string, val interface{}) error {
	var err error
	if val == this {
		selfRef := (*annotatedValueSelfReference)(val.(*annotatedValue))
		err = this.Value.SetField(field, selfRef)
		if err == nil {
			this.flags |= _HAS_SELF_REF
		}
	} else {
		err = this.Value.SetField(field, val)
		if err == nil {
			v, ok := val.(Value)
			if ok {
				v.Track()
			}
		}
	}
	return err
}

func (this *annotatedValue) Size() uint64 {
	if this.cachedSize != 0 {
		return uint64(this.cachedSize)
	}
	sz := this.Value.Size() + uint64(unsafe.Sizeof(*this))
	if this.original != nil {
		sz += this.original.Size()
	}
	// even though these may be shared, count them for each annotatedValue (prefer over to under counting quota)
	if this.attachments != nil {
		sz += AnySize(this.attachments)
	}
	if this.meta != nil {
		sz += AnySize(this.meta)
	}
	if this.projectionOrder != nil {
		sz += AnySize(this.projectionOrder)
	}
	if this.covers != nil {
		sz += this.covers.Size()
	}
	this.cachedSize = uint32(sz)
	return sz
}

func (this *annotatedValue) RecalculateSize() uint64 {
	this.cachedSize = 0
	return this.Size()
}

func (this *annotatedValue) SetValue(v Value) {
	this.Value = v
}

func (this *annotatedValue) GetValue() Value {
	return this.Value
}

func (this *annotatedValue) GetParent() Value {
	if sc, ok := this.Value.(*ScopeValue); ok {
		return sc.Parent()
	}
	return nil
}

func (this *annotatedValue) SetParent(p Value) Value {
	if sc, ok := this.Value.(*ScopeValue); ok {
		return sc.SetParent(p)
	}
	return nil
}

func (this *annotatedValue) ResetAttachments() {
	this.attachments = nil
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
	if this.meta == nil {
		this.meta = make(map[string]interface{}, _DEFAULT_ATTACHMENT_SIZE)
	}
	return this.meta
}

func (this *annotatedValue) SetMeta(meta map[string]interface{}) {
	k := this.GetId()
	this.meta = meta
	if k != nil {
		this.SetId(k)
	}
}

func (this *annotatedValue) ResetMeta() {
	k := this.GetId()
	this.meta = nil
	if k != nil {
		this.SetId(k)
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
	if this.sharedAnnotations() {

		// if we are already sharing with the source no need to do anything
		if amp == tmp && aap == tap {
			return
		}
		this.attachments = nil
		this.meta = nil
	} else {

		// if the source is already sharing with us no need to do anything
		if av.sharedAnnotations() && amp == tmp && aap == tap {
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
	if av.Covers() != nil {
		this.covers = av.covers.Copy()
	} else {
		this.covers = nil
	}
	this.flags &^= _SHARED_ANNOTATIONS
}

func (this *annotatedValue) ShareAnnotations(sv AnnotatedValue) {
	av := sv.(*annotatedValue) // we don't have any other annotated value implementation
	this.attachments = av.attachments
	this.meta = av.meta
	this.covers = av.covers
	// So we don't clean-up under the shared user, both "sides" must be marked as shared
	av.flags |= _SHARED_ANNOTATIONS
	this.flags |= _SHARED_ANNOTATIONS
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
	return this.self()
}

func (this *annotatedValue) SetSelf(s bool) {
	if s {
		this.flags |= _SELF
	} else {
		this.flags &^= _SELF
	}
}

func (this *annotatedValue) GetId() interface{} {
	if this.meta == nil {
		return nil
	}
	return this.meta["id"]
}

func (this *annotatedValue) SetId(id interface{}) {
	this.NewMeta()["id"] = id
}

func (this *annotatedValue) SetProjection(proj Value, order []string) {
	if this != proj {
		this.original = this.Value
		this.Value = proj
	}
	this.projectionOrder = order
}

func (this *annotatedValue) ProjectionOrder() []string {
	return this.projectionOrder
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
	av.flags |= _NO_RECYCLE
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

func (this *annotatedValue) ResetOriginal() {
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
	}
	this.original = nil
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
	if !this.noRecycle() {
		atomic.AddInt32(&this.refCnt, 1)
	}
}

func (this *annotatedValue) Recycle() {
	this.recycle(-1)
}

func (this *annotatedValue) RefCnt() int32 {
	return atomic.LoadInt32(&this.refCnt)
}

func (this *annotatedValue) recycle(lvl int32) {

	// don't change the refCnt if marked as not-recycleable
	if this.noRecycle() {
		return
	}
	// do no recycle if other scope values are using this value
	refcnt := atomic.AddInt32(&this.refCnt, lvl)
	if refcnt > 0 {
		return
	}
	if refcnt < 0 {

		// TODO enable
		// panic("annotated value already recycled")
		logging.Infof("[%p] annotated value already recycled", this)
		return
	}

	if this.Value != nil {
		this.Value.Recycle()
		this.Value = nil
	}
	if this.covers != nil {
		if !this.sharedAnnotations() {
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

	if this.sharedAnnotations() {
		this.attachments = nil
		this.meta = nil
	} else {
		// We must not do this if this object's annotations have been shared - the shared user may still be using them - so
		// whenever annotations are shared both sides are marked as such and this is avoided.  It does limit the cases when these
		// maps will be pooled.

		// pool the maps if they exist. These are optimized as map clear operations by golang.
		for k := range this.attachments {
			delete(this.attachments, k)
		}
		for k := range this.meta {
			delete(this.meta, k)
		}
	}
	this.flags = 0
	this.projectionOrder = nil
	annotatedPool.Put(unsafe.Pointer(this))
}

func (this *annotatedValue) WriteSpill(w io.Writer, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}

	buf = buf[:1]
	buf[0] = _SPILL_TYPE_VALUE_ANNOTATED
	_, err := w.Write(buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	err = writeSpillValue(w, this.Value, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	err = writeSpillValue(w, this.original, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	err = writeSpillValue(w, this.covers, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	err = writeSpillValue(w, this.attachments, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	err = writeSpillValue(w, this.meta, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	buf = buf[:4]
	binary.BigEndian.PutUint32(buf, uint32(this.refCnt))
	_, err = w.Write(buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}

	buf = buf[:3]
	buf[0] = this.bit
	if this.self() {
		buf[1] = 1
	} else {
		buf[1] = 0
	}
	if this.noRecycle() {
		buf[2] = 1
	} else {
		buf[2] = 0
	}
	_, err = w.Write(buf)
	if free {
		_SPILL_POOL.Put(buf)
	}
	return err
}

func (this *annotatedValue) ReadSpill(r io.Reader, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}
	var err error
	var v interface{}
	v, err = readSpillValue(r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.Value = v.(Value)
		// restore any self references
		for k, v := range this.Value.Fields() {
			if _, ok := v.(*annotatedValueSelfReference); ok {
				this.SetField(k, this)
			}
		}
	} else {
		this.Value = nil
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.original = v.(Value)
	} else {
		this.original = nil
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.covers = v.(Value)
	} else {
		this.covers = nil
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.attachments = v.(map[string]interface{})
	} else {
		this.attachments = nil
	}

	v, err = readSpillValue(r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.meta = v.(map[string]interface{})
	} else {
		this.meta = nil
	}

	buf = buf[:4]
	_, err = r.Read(buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	this.refCnt = int32(binary.BigEndian.Uint32(buf))

	buf = buf[:3]
	_, err = r.Read(buf)
	if err == nil {
		this.bit = buf[0]
		this.flags |= _SELF
		if buf[1] != 1 {
			this.flags ^= _SELF
		}
		this.flags |= _NO_RECYCLE
		if buf[2] != 1 {
			this.flags ^= _NO_RECYCLE
		}
	}
	if free {
		_SPILL_POOL.Put(buf)
	}
	return err
}

type annotatedValueSelfReference annotatedValue

func (this *annotatedValueSelfReference) Type() Type {
	return NULL
}

func (this *annotatedValueSelfReference) Size() uint64 {
	return 0
}

func (this *annotatedValueSelfReference) RecalculateSize() uint64 {
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

func (this *annotatedValueSelfReference) WriteJSON(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return nil
}

func (this *annotatedValueSelfReference) WriteXML(order []string, w io.Writer, prefix, indent string, fast bool) error {
	return nil
}

func (this *annotatedValueSelfReference) Equals(other Value) Value {
	return FALSE_VALUE
}

func (this *annotatedValueSelfReference) EquivalentTo(other Value) bool {
	// always compare as true so referred value is seen as equivalent
	return true
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

func (this *annotatedValueSelfReference) SetValue(v Value) {
	(*annotatedValue)(this).SetValue(v)
}

func (this *annotatedValueSelfReference) GetValue() Value {
	return (*annotatedValue)(this).GetValue()
}

func (this *annotatedValueSelfReference) GetParent() Value {
	return (*annotatedValue)(this).GetParent()
}

func (this *annotatedValueSelfReference) SetParent(p Value) Value {
	return (*annotatedValue)(this).SetParent(p)
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

func (this *annotatedValueSelfReference) ResetAttachments() {
	(*annotatedValue)(this).ResetAttachments()
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

func (this *annotatedValueSelfReference) SetMeta(meta map[string]interface{}) {
	(*annotatedValue)(this).SetMeta(meta)
}

func (this *annotatedValueSelfReference) ResetMeta() {
	(*annotatedValue)(this).ResetMeta()
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

func (this *annotatedValueSelfReference) SetProjection(proj Value, order []string) {
	(*annotatedValue)(this).SetProjection(proj, order)
}
func (this *annotatedValueSelfReference) ProjectionOrder() []string {
	return (*annotatedValue)(this).ProjectionOrder()
}

func (this *annotatedValueSelfReference) Original() AnnotatedValue {
	return (*annotatedValue)(this).Original()
}

func (this *annotatedValueSelfReference) ResetOriginal() {
	(*annotatedValue)(this).ResetOriginal()
}

func (this *annotatedValueSelfReference) WriteSpill(w io.Writer, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}

	buf = buf[:1]
	buf[0] = _SPILL_TYPE_VALUE_ANNOTATED_SELFREF
	_, err := w.Write(buf)
	if free {
		_SPILL_POOL.Put(buf)
	}
	return err
}

func (this *annotatedValueSelfReference) ReadSpill(r io.Reader, buf []byte) error {
	return nil
}

func (this *annotatedValueSelfReference) Copy() Value {
	return this
}

func (this *annotatedValueSelfReference) CopyForUpdate() Value {
	return this
}
