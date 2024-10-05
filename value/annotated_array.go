//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package value

import (
	"bufio"
	"compress/zlib"
	"container/heap"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
)

const _SPILL_FILE_PATTERN = "av_spill_*"
const _MAX_PARENTS = 2000

type writerFlusher interface {
	io.Writer
	Flush() error
}

type spillFile struct {
	f       *os.File
	reader  io.Reader
	current AnnotatedValue
	sz      int64

	write time.Duration
	read  time.Duration

	lessFn func(AnnotatedValue, AnnotatedValue) bool

	compress bool
}

func (this *spillFile) rewind() error {
	_, err := this.f.Seek(0, os.SEEK_SET)
	if err != nil {
		return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
	}
	this.reader = bufio.NewReaderSize(this.f, 64*util.KiB)
	if this.compress {
		this.reader, _ = zlib.NewReader(this.reader)
	}
	this.current = nil
	return nil
}

func (this *spillFile) Read(b []byte) (int, error) {
	return io.ReadFull(this.reader, b)
}

func (this *spillFile) nextValue(valFunc func(av AnnotatedValue) bool) error {
	this.current = nil
	s := time.Now()
	v, err := readSpillValue(this, nil)
	this.read += time.Now().Sub(s)
	if err != nil {
		if err == io.EOF {
			return err
		}
		return errors.NewValueError(errors.E_VALUE_RECONSTRUCT, err)
	}
	av, ok := v.(AnnotatedValue)
	if !ok {
		return errors.NewValueError(errors.E_VALUE_INVALID)
	}
	if valFunc != nil && !valFunc(av) {
		return errors.NewValueError(errors.E_VALUE_INVALID)
	}
	av.Track()
	this.current = av
	return nil
}

func (this *spillFile) less(other *spillFile) bool {
	if this.current == nil && other.current != nil {
		return false
	} else if this.current != nil && other.current == nil {
		return true
	} else if this.current == nil {
		return false
	}
	return this.lessFn(this.current, other.current)
}

type spillFileHeap []*spillFile

func (this *spillFileHeap) Len() int           { return len(*this) }
func (this *spillFileHeap) Less(i, j int) bool { return (*this)[i].less((*this)[j]) }
func (this *spillFileHeap) Swap(i, j int)      { (*this)[i], (*this)[j] = (*this)[j], (*this)[i] }
func (this *spillFileHeap) Push(x interface{}) { *this = append(*this, x.(*spillFile)) }
func (this *spillFileHeap) Pop() interface{} {
	i := len(*this) - 1
	last := (*this)[i]
	*this = (*this)[:i]
	return last
}

type iterInfo struct {
	valid     bool
	fileIndex int
	memIndex  int
}

type AnnotatedArray struct {
	acquire     func(int) AnnotatedValues
	release     func(AnnotatedValues)
	less        func(AnnotatedValue, AnnotatedValue) bool
	shouldSpill func(uint64, uint64) bool
	trackMemory func(int64)

	mem      AnnotatedValues
	heapSize int
	heap     bool
	memSize  uint64
	length   int
	spill    spillFileHeap
	iterator iterInfo

	compress bool

	// we keep all the parent values in memory as they're typically shared between multiple items and as spilling multiple times
	// leads to excessive space & processing requirements.  Note however that this does mean that in a scenario where the parent is
	// (largely) unique and is significantly larger than the items, spilling will have a negligible positive impact on actual
	// memory use. (See ScopeValue.Size - we don't include parents in quota.)
	parentsMap map[string]Value
	valFunc    func(av AnnotatedValue) bool

	valIn uint64
}

func NewAnnotatedArray(acquire func(int) AnnotatedValues, release func(AnnotatedValues),
	shouldSpill func(uint64, uint64) bool,
	trackMemory func(int64),
	less func(AnnotatedValue, AnnotatedValue) bool,
	compressSpill bool) *AnnotatedArray {

	rv := &AnnotatedArray{
		acquire:     acquire,
		release:     release,
		less:        less,
		shouldSpill: shouldSpill,
		trackMemory: trackMemory,
		compress:    compressSpill && logging.LogLevel() != logging.DEBUG,
		parentsMap:  make(map[string]Value),
	}
	rv.valFunc = func(av AnnotatedValue) bool {
		p := av.GetAttachment("~parent")
		if p == nil {
			return true
		}
		av.RemoveAttachment("~parent")
		if parent, ok := rv.parentsMap[p.(string)]; ok {
			av.SetParent(parent)
			return true
		} else {
			logging.Debugf("[%p] Unknown parent value: %v", rv, p)
			return false
		}
	}
	return rv
}

func (this *AnnotatedArray) Copy() *AnnotatedArray {
	rv := &AnnotatedArray{
		acquire:     this.acquire,
		release:     this.release,
		less:        this.less,
		shouldSpill: this.shouldSpill,
		trackMemory: this.trackMemory,
		compress:    this.compress,
		parentsMap:  make(map[string]Value),
	}
	rv.valFunc = func(av AnnotatedValue) bool {
		p := av.GetAttachment("~parent")
		if p == nil {
			return true
		}
		av.RemoveAttachment("~parent")
		if parent, ok := rv.parentsMap[p.(string)]; ok {
			av.SetParent(parent)
			return true
		} else {
			logging.Debugf("[%p] Unknown parent value: %v", rv, p)
			return false
		}
	}
	return rv
}

func (this *AnnotatedArray) Length() int {
	return this.length
}

func (this *AnnotatedArray) ShrinkHeapSize(l int) {
	if l < this.heapSize {
		this.SetHeapSize(l)
	}
}

func (this *AnnotatedArray) SetHeapSize(l int) {
	if this.less == nil || l < 0 {
		l = 0
	}
	if this.length > 0 || cap(this.mem) < l {
		this.Release()
	}
	if logging.LogLevel() == logging.DEBUG && this.heapSize != l {
		logging.Debugf("[%p] heap size set to: %v", this, l)
	}
	this.heapSize = l
}

func (this *AnnotatedArray) Append(v AnnotatedValue) errors.Error {
	this.iterator.valid = false
	if this.mem == nil {
		this.mem = this.acquire(this.heapSize)
	}
	sz := uint64(0)
	if this.shouldSpill != nil {
		sz = v.Size()
		if this.memSize > 0 && this.shouldSpill(this.memSize, sz) {
			logging.Debugf("[%p] need to spill: %v+%v, heapSize: %v", this, this.memSize, sz, this.heapSize)
			err := this.spillToDisk()
			if err != nil {
				return errors.NewValueError(errors.E_VALUE_SPILL_WRITE, err)
			}
		}
	}
	this.valIn++
	if len(this.mem) == cap(this.mem) {
		nm := this.acquire(len(this.mem) << 1)
		nm = nm[:len(this.mem)]
		copy(nm, this.mem)
		if this.release != nil {
			this.release(this.mem)
		}
		this.mem = nm
	}
	if this.heapSize > 0 {
		// Prune the item that does not need to enter the heap.
		if len(this.mem) == this.heapSize && !this.less(v, this.mem[0]) {
			if this.trackMemory != nil {
				this.trackMemory(-int64(v.Size()))
			}
			v.Recycle()
			return nil
		}
		heap.Push(this, v)
		this.length++
		if len(this.mem) > this.heapSize {
			ov := heap.Pop(this).(AnnotatedValue)
			sz := ov.Size()
			if this.shouldSpill != nil {
				this.memSize -= sz
			}
			this.length--
			if this.trackMemory != nil {
				this.trackMemory(-int64(sz))
			}
			ov.Recycle()
		}
	} else {
		this.mem = append(this.mem, v)
		this.length++
	}
	this.memSize += sz
	return nil
}

func (this *AnnotatedArray) spillToDisk() error {
	if this.memSize == 0 || len(this.mem) == 0 {
		// nothing to spill
		return nil
	}
	if logging.LogLevel() == logging.DEBUG && this.heapSize > 0 {
		logging.Debugf("[%p] switching from heap to standard", this)
	}
	this.heapSize = 0
	if this.less != nil {
		sort.Sort(this)
	}
	start := util.Now()
	sf, err := util.CreateTemp(_SPILL_FILE_PATTERN, true)
	if err != nil {
		return errors.NewValueError(errors.E_VALUE_SPILL_CREATE, err)
	}
	logging.Debugf("[%p] spilling to %s (#:%v, sz:%v, compr:%v)", this, sf.Name(), len(this.mem), this.memSize, this.compress)
	spf := &spillFile{f: sf, lessFn: this.less, compress: this.compress}
	this.spill = append(this.spill, spf)
	var writer writerFlusher
	if this.compress {
		writer = zlib.NewWriter(sf)
	} else {
		writer = bufio.NewWriter(sf)
	}
	for i, v := range this.mem {
		vv := v.GetValue()
		if sv, ok := vv.(*ScopeValue); ok {
			parent := sv.Parent()
			ps := fmt.Sprintf("%p", parent)
			_, ok := this.parentsMap[ps]
			if !ok && len(this.parentsMap) < _MAX_PARENTS {
				parent.Track()
				this.parentsMap[ps] = parent
				ok = true
			}
			if ok {
				// avoid spilling the parent; spill the map key only
				sv.ResetParent(nil)
				v.SetAttachment("~parent", ps)
			}
		}
		s := time.Now()
		err := v.WriteSpill(writer, nil)
		spf.write += time.Now().Sub(s)
		if err != nil {
			return errors.NewValueError(errors.E_VALUE_SPILL_WRITE, err)
		}
		sz := v.Size()
		if this.trackMemory != nil {
			this.trackMemory(-int64(sz))
		}
		this.memSize -= sz
		this.mem[i].Recycle()
		this.mem[i] = nil
	}
	writer.Flush()
	spf.sz, err = sf.Seek(0, os.SEEK_END)
	if err != nil {
		this.Truncate(nil)
		return errors.NewValueError(errors.E_VALUE_SPILL_SIZE, err)
	}
	if !util.UseTemp(spf.f.Name(), spf.sz) {
		spf.sz = 0
		this.Truncate(nil)
		return errors.NewTempFileQuotaExceededError()
	}

	logging.Debuga(func() string {
		d := util.Since(start)
		return fmt.Sprintf("[%p] spill took: %v memSize: %v spf.sz: %v", this, d, this.memSize, spf.sz)
	})
	this.mem = this.mem[:0]
	return nil
}

func (this *AnnotatedArray) Foreach(f func(AnnotatedValue) bool) errors.Error {

	if this.mem == nil {
		this.mem = this.acquire(0)
		this.memSize = 0
	}
	this.iterator.valid = true
	this.iterator.fileIndex = 0
	this.iterator.memIndex = 0

	for i := range this.spill {
		err := this.spill[i].rewind()
		if err != nil {
			logging.Debugf("[%p] rewind failed on [%d] %s: %v", this, i, this.spill[i].f.Name(), err)
			return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
		}
		if this.less != nil {
			err = this.spill[i].nextValue(this.valFunc)
			if err != nil {
				logging.Debugf("[%p] initial read failed on [%d] %s: %v", this, i, this.spill[i].f.Name(), err)
				return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
			}
			if this.trackMemory != nil {
				this.trackMemory(int64(this.spill[i].current.Size()))
			}
		}
	}

	if this.less != nil {
		this.heapSize = 0
		sort.Sort(this)

		heap.Init(&this.spill)

		for {
			av, err, eof := this.nextSorted()
			if err != nil || eof {
				return err
			}
			if !f(av) {
				return nil
			}
		}
	} else {
		for {
			av, err, eof := this.nextUnsorted()
			if err != nil || eof {
				return err
			}
			if !f(av) {
				return nil
			}
		}
	}
}

func (this *AnnotatedArray) nextUnsorted() (AnnotatedValue, errors.Error, bool) {
	for {
		if this.iterator.fileIndex >= len(this.spill) {
			if this.iterator.memIndex >= len(this.mem) {
				return nil, nil, true
			}
			rv := this.mem[this.iterator.memIndex]
			this.iterator.memIndex++
			return rv, nil, false
		}
		err := this.spill[this.iterator.fileIndex].nextValue(this.valFunc)
		if err == io.EOF {
			this.iterator.fileIndex++
			if this.iterator.fileIndex == len(this.spill) {
				this.iterator.memIndex = 0
			}
			continue
		}
		if err != nil {
			if err == io.EOF {
				return nil, nil, true
			}
			return nil, errors.NewValueError(errors.E_VALUE_SPILL_READ, err), false
		}
		if this.spill[this.iterator.fileIndex].current == nil {
			logging.Debugf("[%p] nil value for [%d] %s", this, this.iterator.fileIndex,
				this.spill[this.iterator.fileIndex].f.Name())
		} else if this.trackMemory != nil {
			this.trackMemory(int64(this.spill[this.iterator.fileIndex].current.Size()))
		}
		return this.spill[this.iterator.fileIndex].current, nil, false
	}
}

func (this *AnnotatedArray) nextSorted() (AnnotatedValue, errors.Error, bool) {
	var smallest *spillFile
	if this.spill != nil {
		smallest = this.spill[0]
	}
	if this.iterator.memIndex < len(this.mem) {
		if smallest == nil || smallest.current == nil || this.less(this.mem[this.iterator.memIndex], smallest.current) {
			rv := this.mem[this.iterator.memIndex]
			this.iterator.memIndex++
			return rv, nil, false
		}
	}
	if smallest == nil || smallest.current == nil {
		return nil, nil, true
	}
	rv := smallest.current
	err := smallest.nextValue(this.valFunc)
	if err != io.EOF && err != nil {
		return nil, errors.NewValueError(errors.E_VALUE_SPILL_READ, err), false
	}
	if this.trackMemory != nil && smallest.current != nil {
		this.trackMemory(int64(smallest.current.Size()))
	}
	heap.Fix(&this.spill, 0)
	return rv, nil, false
}

func (this *AnnotatedArray) Release() {
	this.Truncate(nil)
	if this.release != nil {
		this.release(this.mem)
	}
	this.mem = nil
}

func (this *AnnotatedArray) Len() int {
	return len(this.mem)
}

func (this *AnnotatedArray) Less(i, j int) bool {
	if this.heapSize > 0 {
		return this.less(this.mem[j], this.mem[i])
	}
	return this.less(this.mem[i], this.mem[j])
}

func (this *AnnotatedArray) Swap(i, j int) {
	this.mem[i], this.mem[j] = this.mem[j], this.mem[i]
}

func (this *AnnotatedArray) Push(i interface{}) {
	if this.heapSize > 0 {
		this.mem = append(this.mem, i.(AnnotatedValue))
	}
}

func (this *AnnotatedArray) Pop() interface{} {
	var rv interface{}
	if this.heapSize > 0 {
		rv = this.mem[len(this.mem)-1]
		this.mem = this.mem[:len(this.mem)-1]
	}
	return rv
}

func (this *AnnotatedArray) Truncate(onDiscard func(AnnotatedValue)) {
	for k, p := range this.parentsMap {
		p.Recycle()
		delete(this.parentsMap, k)
	}
	for i := range this.spill {
		if this.spill[i].f != nil {
			util.ReleaseTemp(this.spill[i].f.Name(), this.spill[i].sz)
			this.spill[i].f.Close()
		}
		this.spill[i].current = nil
	}
	this.spill = nil
	for i := range this.mem {
		if onDiscard != nil {
			onDiscard(this.mem[i])
		}
		this.mem[i] = nil
	}
	this.mem = this.mem[:0]
	this.length = 0
	this.iterator.valid = false
	this.memSize = 0
}

func (this *AnnotatedArray) Stats() string {
	s := ""
	var tr, tw time.Duration
	for _, sf := range this.spill {
		s += fmt.Sprintf(" [r:%v,w:%v]", sf.read, sf.write)
		tr += sf.read
		tw += sf.write
	}
	s = fmt.Sprintf("[%p,vals:%v,R:%v,W:%v,#parents:%d]", this, this.valIn, tr, tw, len(this.parentsMap)) + s
	return s
}
