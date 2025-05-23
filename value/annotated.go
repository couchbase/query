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
	"fmt"
	"io"
	"reflect"
	"unsafe"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const (
	META_ID int = 1 << iota
	META_CAS
	META_KEYSPACE
	META_TYPE
	META_FLAGS
	META_EXPIRATION
	META_XATTRS
	META_TXNMETA
	META_TXPLANS
	META_SUBQUERY_PLANS
	META_PLAN
	META_OPT_ESTIMATES
	META_DISTRIBUTIONS
	META_PLAN_VERSION
	META_BYSEQNO
	META_REVSEQNO
	META_LOCKTIME
	META_NRU
)

var metaNames = map[int]string{
	META_ID:             "id",
	META_CAS:            "cas",
	META_KEYSPACE:       "keyspace",
	META_TYPE:           "type",
	META_FLAGS:          "flags",
	META_EXPIRATION:     "expiration",
	META_XATTRS:         "xattrs",
	META_TXNMETA:        "txnMeta",
	META_TXPLANS:        "txPlans",
	META_SUBQUERY_PLANS: "subqueryPlans",
	META_PLAN:           "plan",
	META_OPT_ESTIMATES:  "optimizerEstimates",
	META_DISTRIBUTIONS:  "distributions",
	META_PLAN_VERSION:   "planVersion",
	META_BYSEQNO:        "byseqno",
	META_REVSEQNO:       "revseqno",
	META_LOCKTIME:       "locktime",
	META_NRU:            "nru",
}

const (
	ATT_UNNEST_POSITION int16 = iota
	ATT_LIST
	ATT_STARTPOS
	ATT_SET
	ATT_SUM
	ATT_SMETA
	ATT_AGGREGATES
	ATT_COUNT
	ATT_KEY
	ATT_VALUE
	ATT_OPTIONS
	ATT_CLONE
	ATT_WINDOW_ATTACHMENT
	ATT_SEQUENCES
	ATT_PROJECTION
	ATT_PARENT
	ATT_CUSTOM_INDEX // must be last
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
	SetValue(Value)
	GetValue() Value
	GetParent() Value
	SetParent(Value) Value
	Attachments() map[int16]interface{}
	GetAttachment(key int16) interface{}
	SetAttachment(key int16, val interface{})
	RemoveAttachment(key int16)
	ResetAttachments()
	GetId() interface{}
	SetId(id interface{})
	GetMetaMap() map[string]interface{} // only intended for returning metadata in the Meta() expression function
	CopyMeta(AnnotatedValue)
	GetMetaField(id int) interface{}
	SetMetaField(id int, v interface{})
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

type metaData struct {
	id            interface{}
	xAttrs        interface{}
	txnMeta       interface{}
	txPlans       interface{}
	sqPlans       interface{}
	plan          interface{}
	optEst        interface{}
	distributions interface{}
	keyspace      string
	typ           string
	cas           uint64
	bySeqNo       uint64
	revSeqNo      uint64
	valid         int // bits for valid (set) fields
	flags         uint32
	expiration    uint32
	lockTime      uint32
	planVersion   int32
	nru           byte
}

func (this *metaData) size() uint64 {
	sz := uint64(unsafe.Sizeof(*this))
	if this.valid&META_ID != 0 {
		sz += AnySize(this.id)
	}
	if this.valid&META_KEYSPACE != 0 {
		sz += AnySize(this.keyspace)
	}
	if this.valid&META_TYPE != 0 {
		sz += AnySize(this.typ)
	}
	if this.valid&META_XATTRS != 0 {
		sz += AnySize(this.xAttrs)
	}
	if this.valid&META_TXNMETA != 0 {
		sz += AnySize(this.txnMeta)
	}
	if this.valid&META_TXPLANS != 0 {
		sz += AnySize(this.txPlans)
	}
	if this.valid&META_SUBQUERY_PLANS != 0 {
		sz += AnySize(this.sqPlans)
	}
	if this.valid&META_PLAN != 0 {
		sz += AnySize(this.plan)
	}
	if this.valid&META_OPT_ESTIMATES != 0 {
		sz += AnySize(this.optEst)
	}
	if this.valid&META_DISTRIBUTIONS != 0 {
		sz += AnySize(this.distributions)
	}
	if this.valid&META_PLAN_VERSION != 0 {
		sz += AnySize(this.planVersion)
	}
	return sz
}

func (this *metaData) writeSpill(w io.Writer, buf []byte) error {
	err := writeSpillValue(w, this.valid, buf)
	if err != nil {
		return err
	}
	if this.valid&META_ID != 0 {
		err = writeSpillValue(w, this.id, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_CAS != 0 {
		err = writeSpillValue(w, this.cas, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_KEYSPACE != 0 {
		err = writeSpillValue(w, this.keyspace, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_TYPE != 0 {
		err = writeSpillValue(w, this.typ, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_FLAGS != 0 {
		err = writeSpillValue(w, this.flags, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_EXPIRATION != 0 {
		err = writeSpillValue(w, this.expiration, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_XATTRS != 0 {
		err = writeSpillValue(w, this.xAttrs, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_TXNMETA != 0 {
		err = writeSpillValue(w, this.txnMeta, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_TXPLANS != 0 {
		err = writeSpillValue(w, this.txPlans, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_SUBQUERY_PLANS != 0 {
		err = writeSpillValue(w, this.sqPlans, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_PLAN != 0 {
		err = writeSpillValue(w, this.plan, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_OPT_ESTIMATES != 0 {
		err = writeSpillValue(w, this.optEst, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_DISTRIBUTIONS != 0 {
		err = writeSpillValue(w, this.distributions, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_PLAN_VERSION != 0 {
		err = writeSpillValue(w, this.planVersion, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_BYSEQNO != 0 {
		err = writeSpillValue(w, this.bySeqNo, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_REVSEQNO != 0 {
		err = writeSpillValue(w, this.revSeqNo, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_LOCKTIME != 0 {
		err = writeSpillValue(w, this.lockTime, buf)
		if err != nil {
			return err
		}
	}
	if this.valid&META_NRU != 0 {
		err = writeSpillValue(w, this.nru, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *metaData) readSpill(trackMem func(int64) error, r io.Reader, buf []byte) error {
	v, err := readSpillValue(nil, r, buf) // the mask isn't part of the value so don't track size
	if err != nil {
		return err
	}
	this.valid = v.(int)
	if this.valid&META_ID != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.id = v
	}
	if this.valid&META_CAS != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.cas = v.(uint64)
	}
	if this.valid&META_KEYSPACE != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.keyspace = v.(string)
	}
	if this.valid&META_TYPE != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.typ = v.(string)
	}
	if this.valid&META_FLAGS != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.flags = v.(uint32)
	}
	if this.valid&META_EXPIRATION != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.expiration = v.(uint32)
	}
	if this.valid&META_XATTRS != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.xAttrs = v
	}
	if this.valid&META_TXNMETA != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.txnMeta = v
	}
	if this.valid&META_TXPLANS != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.txPlans = v
	}
	if this.valid&META_SUBQUERY_PLANS != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.sqPlans = v
	}
	if this.valid&META_PLAN != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.plan = v
	}
	if this.valid&META_OPT_ESTIMATES != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.optEst = v
	}
	if this.valid&META_DISTRIBUTIONS != 0 {
		v, err = readSpillValue(trackMem, r, buf)
		if err != nil {
			return err
		}
		this.distributions = v
	}
	if this.valid&META_PLAN_VERSION != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.planVersion = v.(int32)
	}
	if this.valid&META_BYSEQNO != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.bySeqNo = v.(uint64)
	}
	if this.valid&META_REVSEQNO != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.revSeqNo = v.(uint64)
	}
	if this.valid&META_LOCKTIME != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.lockTime = v.(uint32)
	}
	if this.valid&META_NRU != 0 {
		v, err = readSpillValue(nil, r, buf)
		if err != nil {
			return err
		}
		this.nru = v.(byte)
	}
	return nil
}

type annotatedValue struct {
	Value
	attachments     map[int16]interface{}
	meta            *metaData
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

const _DEF_META_SIZE = 18

func (this *annotatedValue) GetMetaMap() map[string]interface{} {
	if this.meta == nil {
		return nil
	}
	out := make(map[string]interface{}, _DEF_META_SIZE)
	for id := 1; id <= META_NRU; id <<= 1 {
		if this.meta.valid&id == id {
			v := this.GetMetaField(id)
			if name, ok := metaNames[id]; ok {
				out[name] = v
			} else {
				out[fmt.Sprintf("unknown:%v", id)] = v
			}
		}
	}
	return out
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

func (this *annotatedValue) CopyMeta(from AnnotatedValue) {
	if fav, ok := from.(*annotatedValue); ok && fav != nil {
		copyMeta(fav.meta, this)
	}
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
	sz := uint64(unsafe.Sizeof(*this))
	if this.Value != nil {
		sz += this.Value.Size()
	}
	if this.original != nil {
		sz += this.original.Size()
	}
	// even though these may be shared, count them for each annotatedValue (prefer over to under counting quota)
	if this.attachments != nil {
		sz += AnySize(this.attachments)
	}
	if this.meta != nil {
		sz += this.meta.size()
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
		return sc.ResetParent(p)
	}
	return nil
}

func (this *annotatedValue) ResetAttachments() {
	this.attachments = nil
}

func (this *annotatedValue) Attachments() map[int16]interface{} {
	return this.attachments
}

func (this *annotatedValue) GetAttachment(key int16) interface{} {
	if this.attachments != nil {
		return this.attachments[key]
	}
	return nil
}

func (this *annotatedValue) SetAttachment(key int16, val interface{}) {
	if this.attachments == nil {
		this.attachments = make(map[int16]interface{}, _DEFAULT_ATTACHMENT_SIZE)
	}
	this.attachments[key] = val
}

func (this *annotatedValue) RemoveAttachment(key int16) {
	if this.attachments != nil {
		delete(this.attachments, key)
	}
}

func (this *annotatedValue) GetMetaField(id int) interface{} {
	if this.meta == nil {
		return nil
	}
	if this.meta.valid&id != id {
		return nil
	}
	switch id {
	case META_ID:
		return this.meta.id
	case META_CAS:
		return this.meta.cas
	case META_KEYSPACE:
		return this.meta.keyspace
	case META_TYPE:
		return this.meta.typ
	case META_FLAGS:
		return this.meta.flags
	case META_EXPIRATION:
		return this.meta.expiration
	case META_XATTRS:
		return this.meta.xAttrs
	case META_TXNMETA:
		return this.meta.txnMeta
	case META_TXPLANS:
		return this.meta.txPlans
	case META_SUBQUERY_PLANS:
		return this.meta.sqPlans
	case META_PLAN:
		return this.meta.plan
	case META_OPT_ESTIMATES:
		return this.meta.optEst
	case META_DISTRIBUTIONS:
		return this.meta.distributions
	case META_PLAN_VERSION:
		return this.meta.planVersion
	case META_BYSEQNO:
		return this.meta.bySeqNo
	case META_REVSEQNO:
		return this.meta.revSeqNo
	case META_LOCKTIME:
		return this.meta.lockTime
	case META_NRU:
		return this.meta.nru
	default:
		return nil
	}
}

func (this *annotatedValue) SetMetaField(id int, v interface{}) {
	if this.meta == nil {
		this.meta = &metaData{}
	}
	switch id {
	case META_ID:
		this.meta.id = v
	case META_CAS:
		this.meta.cas, _ = v.(uint64)
	case META_KEYSPACE:
		this.meta.keyspace, _ = v.(string)
	case META_TYPE:
		this.meta.typ, _ = v.(string)
	case META_FLAGS:
		this.meta.flags, _ = v.(uint32)
	case META_EXPIRATION:
		this.meta.expiration, _ = v.(uint32)
	case META_XATTRS:
		this.meta.xAttrs = v
	case META_TXNMETA:
		this.meta.txnMeta = v
	case META_TXPLANS:
		this.meta.txPlans = v
	case META_SUBQUERY_PLANS:
		this.meta.sqPlans = v
	case META_PLAN:
		this.meta.plan = v
	case META_OPT_ESTIMATES:
		this.meta.optEst = v
	case META_DISTRIBUTIONS:
		this.meta.distributions = v
	case META_PLAN_VERSION:
		this.meta.planVersion, _ = v.(int32)
	case META_BYSEQNO:
		this.meta.bySeqNo, _ = v.(uint64)
	case META_REVSEQNO:
		this.meta.revSeqNo, _ = v.(uint64)
	case META_LOCKTIME:
		this.meta.lockTime, _ = v.(uint32)
	case META_NRU:
		this.meta.nru, _ = v.(byte)
	default:
		return
	}
	this.meta.valid |= id
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

	amp := av.meta
	tmp := this.meta
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
		this.meta = nil
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

func copyAttachments(source map[int16]interface{}, dest *annotatedValue) {
	if len(source) == 0 {
		return
	}
	if dest.attachments == nil {
		dest.attachments = make(map[int16]interface{}, len(source))
	}
	for k := range source {
		dest.attachments[k] = source[k]
	}
}

func copyMeta(source *metaData, dest *annotatedValue) {
	if source == nil {
		dest.meta = nil
		return
	}
	if dest.meta == nil {
		dest.meta = &metaData{}
	}
	*dest.meta = *source
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
	return this.GetMetaField(META_ID)
}

func (this *annotatedValue) SetId(id interface{}) {
	this.SetMetaField(META_ID, id)
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
		copyMeta(this.meta, av)
	case *ScopeValue:
		av.Value = val
		av.covers = this.covers
		av.attachments = this.attachments
		copyMeta(this.meta, av)
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
		this.annotatedOrig = nil
		val.Value = nil
		val.covers = nil
		val.attachments = nil
		val.meta = nil
		annotatedPool.Put(unsafe.Pointer(val))
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
		this.annotatedOrig = nil
		val.Value = nil
		val.covers = nil
		val.attachments = nil
		val.meta = nil
		annotatedPool.Put(unsafe.Pointer(val))
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
		this.meta = nil
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

	buf = buf[:1]
	if this.meta == nil {
		buf[0] = 0
	} else {
		buf[0] = 1
	}
	_, err = w.Write(buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if buf[0] != 0 {
		err = this.meta.writeSpill(w, buf)
		if err != nil {
			if free {
				_SPILL_POOL.Put(buf)
			}
			return err
		}
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

func (this *annotatedValue) ReadSpill(trackMemActual func(int64) error, r io.Reader, buf []byte) error {
	free := false
	if buf == nil {
		buf = _SPILL_POOL.Get()
		free = true
	}
	// update the cached size as we go so that when Size() is used to release memory, a matching value (equalling the sum of
	// all the individually recorded additions) is used.
	var trackMem func(int64) error
	if trackMemActual != nil {
		trackMem = func(n int64) error {
			this.cachedSize += uint32(n)
			return trackMemActual(n)
		}
	}
	var err error
	if trackMem != nil {
		if err = trackMem(int64(unsafe.Sizeof(*this))); err != nil {
			return err
		}
	}
	var v interface{}
	v, err = readSpillValue(trackMem, r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.Value = v.(Value)
		// restore any self references
		restoreSelfRef := true
		if p, ok := this.Value.(*parsedValue); ok {
			if len(p.raw) > 0 {
				// we can't have any self references so no need to force unwrapping
				restoreSelfRef = false
			}
		}
		if restoreSelfRef {
			for k, v := range this.Value.Fields() {
				if _, ok := v.(*annotatedValueSelfReference); ok {
					this.SetField(k, this)
				}
			}
		}
	} else {
		this.Value = nil
	}

	v, err = readSpillValue(trackMem, r, buf)
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

	v, err = readSpillValue(trackMem, r, buf)
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

	v, err = readSpillValue(trackMem, r, buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if v != nil {
		this.attachments = v.(map[int16]interface{})
	} else {
		this.attachments = nil
	}

	buf = buf[:1]
	_, err = r.Read(buf)
	if err != nil {
		if free {
			_SPILL_POOL.Put(buf)
		}
		return err
	}
	if buf[0] != 0 {
		this.meta = &metaData{}
		if trackMem != nil {
			err = trackMem(int64(unsafe.Sizeof(*this.meta)))
			if err != nil {
				return err
			}
		}
		err = this.meta.readSpill(trackMem, r, buf)
		if err != nil {
			if free {
				_SPILL_POOL.Put(buf)
			}
			return err
		}
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

func (this *annotatedValueSelfReference) Attachments() map[int16]interface{} {
	return (*annotatedValue)(this).Attachments()
}

func (this *annotatedValueSelfReference) GetAttachment(key int16) interface{} {
	return (*annotatedValue)(this).GetAttachment(key)
}

func (this *annotatedValueSelfReference) SetAttachment(key int16, val interface{}) {
	(*annotatedValue)(this).SetAttachment(key, val)
}

func (this *annotatedValueSelfReference) RemoveAttachment(key int16) {
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

func (this *annotatedValueSelfReference) GetMetaMap() map[string]interface{} {
	return (*annotatedValue)(this).GetMetaMap()
}

func (this *annotatedValueSelfReference) GetMetaField(id int) interface{} {
	return (*annotatedValue)(this).GetMetaField(id)
}

func (this *annotatedValueSelfReference) SetMetaField(id int, v interface{}) {
	(*annotatedValue)(this).SetMetaField(id, v)
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

func (this *annotatedValueSelfReference) ReadSpill(trackMem func(int64) error, r io.Reader, buf []byte) error {
	return nil
}

func (this *annotatedValueSelfReference) CopyMeta(from AnnotatedValue) {
	(*annotatedValue)(this).CopyMeta(from)
}

func (this *annotatedValueSelfReference) Copy() Value {
	return this
}

func (this *annotatedValueSelfReference) CopyForUpdate() Value {
	return this
}
