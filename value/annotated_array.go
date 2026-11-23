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
	"cmp"
	"compress/zlib"
	"container/heap"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/sort"
	"github.com/couchbase/query/util"
)

const _SPILL_FILE_PATTERN = "av_spill_*"

// All spill files of a sorted array are open together while reading it (one _BUFFER_SIZE reader each), so this bounds the
// file descriptors and the read buffers per array. An unsorted array uses a single spill file. Vars only so that tests can
// change them.
var _MAX_SPILL_FILES = 500

// When a sorted array reaches _MAX_SPILL_FILES spill files, its _SPILL_MERGE_FILES smallest spill files are merged into
// one, so it never fails for the number of spill files. Arrays with fewer spills are not affected. The smallest files are
// the cheapest to rewrite, and the large files written by earlier merges are rewritten only rarely.
// Past the limit a merge follows every _SPILL_MERGE_FILES - 1 spills and rewrites those spills plus about the smallest
// earlier merged file, so the data written grows slowly with the number of spills S (roughly 1 + S/100000 times the
// data spilled): negligible for thousands of spills, about twice for a hundred thousand.
var _SPILL_MERGE_FILES = 128

const _MAX_PARENTS = 2000
const _BUFFER_SIZE = 64 * util.KiB

func init() {
	util.RegisterTempPattern(_SPILL_FILE_PATTERN)
}

type writerFlusher interface {
	io.Writer
	Flush() error
}

type spillFile struct {
	f       *util.TempFile
	reader  io.Reader
	current AnnotatedValue
	sz      int64

	write time.Duration
	read  time.Duration

	lessFn func(AnnotatedValue, AnnotatedValue) bool

	compress bool

	encryptionKey *encryption.EaRKey

	// memory tracked for current while merging
	mergeTracked int64
}

func (this *spillFile) rewind() error {
	_, err := this.f.Seek(0, os.SEEK_SET)
	if err != nil {
		return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
	}

	if this.encryptionKey == nil {
		this.reader = bufio.NewReaderSize(this.f, _BUFFER_SIZE)
		if this.compress {
			this.reader, _ = zlib.NewReader(this.reader)
		}
	} else {
		this.reader, err = encryption.NewCBEFReader(this.f, func(keyId string) (*encryption.EaRKey, errors.Error) {
			return this.encryptionKey, nil
		})
		if err != nil {
			return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
		}
	}

	this.current = nil
	return nil
}

func (this *spillFile) Read(b []byte) (int, error) {
	return io.ReadFull(this.reader, b)
}

func (this *spillFile) nextValue(trackMem func(int64) error, valFunc func(av AnnotatedValue) bool) error {
	this.current = nil
	s := time.Now()
	v, err := readSpillValue(trackMem, this, nil)
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
	trackMemory func(int64) error

	mem      AnnotatedValues
	heapSize int
	heap     bool
	memSize  uint64
	length   int
	spill    spillFileHeap
	iterator iterInfo

	// unsorted arrays only: the writer of their single spill file, which each spill appends to
	spillWriter writerFlusher

	compress bool

	// we keep all the parent values in memory as they're typically shared between multiple items and as spilling multiple times
	// leads to excessive space & processing requirements.  Note however that this does mean that in a scenario where the parent is
	// (largely) unique and is significantly larger than the items, spilling will have a negligible positive impact on actual
	// memory use. (See ScopeValue.Size - we don't include parents in quota.)
	parentsMap map[string]Value
	valFunc    func(av AnnotatedValue) bool

	valIn uint64

	// Encryption key for spill files, resolved lazily (only once a spill actually happens) and cached
	// thereafter so getEncryptionKey is invoked at most once per AnnotatedArray.
	encryptionKeyResolved bool
	getEncryptionKey      func() (*encryption.EaRKey, errors.Error)
	encryptionKey         *encryption.EaRKey
}

func NewAnnotatedArray(acquire func(int) AnnotatedValues, release func(AnnotatedValues),
	shouldSpill func(uint64, uint64) bool,
	trackMemory func(int64) error,
	less func(AnnotatedValue, AnnotatedValue) bool,
	compressSpill bool, getEncryptionKey func() (*encryption.EaRKey, errors.Error)) *AnnotatedArray {

	rv := &AnnotatedArray{
		acquire:          acquire,
		release:          release,
		less:             less,
		shouldSpill:      shouldSpill,
		trackMemory:      trackMemory,
		compress:         compressSpill && logging.LogLevel() != logging.DEBUG,
		parentsMap:       make(map[string]Value),
		getEncryptionKey: getEncryptionKey,
	}
	rv.valFunc = func(av AnnotatedValue) bool {
		p := av.GetAttachment(ATT_PARENT)
		if p == nil {
			return true
		}
		av.RemoveAttachment(ATT_PARENT)
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

// Use this method only to checks whether spill files should be encrypted and to obtain the encryption key information
func (this *AnnotatedArray) resolveEncryptionKey() (*encryption.EaRKey, error) {
	if this.encryptionKeyResolved {
		return this.encryptionKey, nil
	}

	if this.getEncryptionKey == nil {
		return nil, fmt.Errorf("getEncryptionKey() not defined")
	}

	key, err := this.getEncryptionKey()
	if err != nil {
		return nil, err
	}

	this.encryptionKey = key
	this.encryptionKeyResolved = true
	return key, nil
}

func (this *AnnotatedArray) Copy() *AnnotatedArray {
	rv := &AnnotatedArray{
		acquire:               this.acquire,
		release:               this.release,
		less:                  this.less,
		shouldSpill:           this.shouldSpill,
		trackMemory:           this.trackMemory,
		compress:              this.compress,
		parentsMap:            make(map[string]Value),
		encryptionKeyResolved: this.encryptionKeyResolved,
		getEncryptionKey:      this.getEncryptionKey,
		encryptionKey:         this.encryptionKey,
	}
	rv.valFunc = func(av AnnotatedValue) bool {
		p := av.GetAttachment(ATT_PARENT)
		if p == nil {
			return true
		}
		av.RemoveAttachment(ATT_PARENT)
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
			encryptionKey, err := this.resolveEncryptionKey()
			if err != nil {
				return spillError(errors.E_VALUE_SPILL_CREATE, err)
			}

			if encryptionKey != nil {
				logging.Debugf("[%p] need to spill: %v+%v, heapSize: %v encryption keyId: %+q", this, this.memSize, sz,
					this.heapSize, encryptionKey.Id)
			} else {
				logging.Debugf("[%p] need to spill: %v+%v, heapSize: %v", this, this.memSize, sz, this.heapSize)
			}
			err = this.spillToDisk()
			if err != nil {
				return spillError(errors.E_VALUE_SPILL_WRITE, err)
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
				err := this.trackMemory(-int64(v.Size()))
				if err != nil {
					v.Recycle()
					return spillError(errors.E_VALUE_SPILL_WRITE, err)
				}
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
				err := this.trackMemory(-int64(sz))
				if err != nil {
					return spillError(errors.E_VALUE_SPILL_WRITE, err)
				}
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

// spillError keeps an errors.Error (e.g. too many spill files, temp file quota exceeded, encryption key or file create
// failure) as is, so the real reason is reported; only other errors are wrapped with the given code.
func spillError(code errors.ErrorCode, err error) errors.Error {
	if e, ok := err.(errors.Error); ok {
		return e
	}
	return errors.NewValueError(code, err)
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
	var spf *spillFile
	var writer writerFlusher
	var err error
	if this.spillWriter != nil {
		// unsorted: append to the spill file; reading it may have moved the file offset
		spf, writer = this.spill[len(this.spill)-1], this.spillWriter
		_, err = spf.f.Seek(0, io.SeekEnd)
		if err != nil {
			return errors.NewValueError(errors.E_VALUE_SPILL_WRITE, err)
		}
	} else {
		if len(this.spill) > _MAX_SPILL_FILES {
			return errors.NewValueError(errors.E_VALUE_SPILL_MAX_FILES)
		}
		spf, writer, err = this.createSpillFile()
		if err != nil {
			return err
		}
		this.spill = append(this.spill, spf)
		if this.less == nil {
			this.spillWriter = writer
		}
	}
	logging.Debugf("[%p] spilling to %s (#:%v, sz:%v)", this, spf.f.Name(), len(this.mem), this.memSize)

	for i, v := range this.mem {
		err = this.writeSpillValue(spf, writer, v, true)
		if err != nil {
			return err
		}
		this.mem[i] = nil
	}
	err = this.finishSpillFile(spf, writer)
	if err != nil {
		return err
	}

	logging.Debuga(func() string {
		d := util.Since(start)
		return fmt.Sprintf("[%p] spill took: %v memSize: %v spf.sz: %v", this, d, this.memSize, spf.sz)
	})
	this.mem = this.mem[:0]
	return this.mergeSpillFiles()
}

func (this *AnnotatedArray) createSpillFile() (*spillFile, writerFlusher, error) {
	sf, err := util.CreateTemp(_SPILL_FILE_PATTERN)
	if err != nil {
		return nil, nil, errors.NewValueError(errors.E_VALUE_SPILL_CREATE, err)
	}

	encryptionKey, err := this.resolveEncryptionKey()
	if err != nil {
		sf.Close()
		return nil, nil, errors.NewValueError(errors.E_VALUE_SPILL_CREATE, err)
	}

	if encryptionKey == nil {
		logging.Debugf("[%p] created spill file %s (compr:%v)", this, sf.Name(), this.compress)
	} else {
		logging.Debugf("[%p] created spill file %s (compr:%v) keyId: %+q", this, sf.Name(), this.compress, encryptionKey.Id)
	}

	var writer writerFlusher
	if encryptionKey == nil {
		if this.compress {
			writer = zlib.NewWriter(sf)
		} else {
			writer = bufio.NewWriter(sf)
		}
	} else {
		compression := encryption.CBEF_NONE
		if this.compress {
			compression = encryption.CBEF_ZLIB
		}

		ew, err := encryption.NewCBEFWriterSize(sf, encryptionKey, compression, _BUFFER_SIZE)
		if err != nil {
			sf.Close()
			return nil, nil, errors.NewValueError(errors.E_VALUE_SPILL_CREATE, err)
		}
		writer = ew
	}

	spf := &spillFile{f: sf, lessFn: this.less, compress: this.compress, encryptionKey: encryptionKey}
	return spf, writer, nil
}

// Writes the value to the spill file and recycles it. inMem is set for values held in this.mem, whose memory is released
// here; the merge releases the memory of the values it reads back itself.
func (this *AnnotatedArray) writeSpillValue(spf *spillFile, writer writerFlusher, v AnnotatedValue, inMem bool) error {
	vv := v.GetValue()
	if sv, ok := vv.(*ScopeValue); ok {
		if parent := sv.Parent(); parent != nil {
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
				v.SetAttachment(ATT_PARENT, ps)
			}
		}
	}
	s := time.Now()
	err := v.WriteSpill(writer, nil)
	spf.write += time.Now().Sub(s)
	if err != nil {
		return errors.NewValueError(errors.E_VALUE_SPILL_WRITE, err)
	}
	if inMem {
		sz := v.Size()
		if this.trackMemory != nil {
			err := this.trackMemory(-int64(sz))
			if err != nil {
				return err
			}
		}
		this.memSize -= sz
	}
	v.Recycle()
	return nil
}

func (this *AnnotatedArray) finishSpillFile(spf *spillFile, writer writerFlusher) error {
	// an unsorted array keeps appending to the same stream, so a failed flush must not be ignored
	err := writer.Flush()
	if err != nil {
		this.Truncate(nil)
		return errors.NewValueError(errors.E_VALUE_SPILL_WRITE, err)
	}
	sz, err := spf.f.Seek(0, io.SeekEnd)
	if err != nil {
		this.Truncate(nil)
		return errors.NewValueError(errors.E_VALUE_SPILL_SIZE, err)
	}
	// spf.sz is what has been accounted so far (non-zero when appending to the spill file of an unsorted array)
	if !util.UseTemp(spf.f.Name(), sz-spf.sz) {
		this.Truncate(nil)
		return errors.NewTempFileQuotaExceededError()
	}
	spf.sz = sz
	return nil
}

// Sorted arrays only: once there are _MAX_SPILL_FILES spill files, merges the _SPILL_MERGE_FILES smallest into one. The
// spill files are read in sorted order, so their order in this.spill does not matter. As when reading the array, the read
// buffers of the merged files (_BUFFER_SIZE each) are not tracked; the values held are.
func (this *AnnotatedArray) mergeSpillFiles() error {
	if this.less == nil || len(this.spill) < _MAX_SPILL_FILES {
		return nil
	}
	n := min(_SPILL_MERGE_FILES, len(this.spill))
	if n < 2 {
		return nil
	}

	start := util.Now()
	bySize := slices.Clone(this.spill)
	slices.SortStableFunc(bySize, func(a, b *spillFile) int { return cmp.Compare(a.sz, b.sz) })
	merge := bySize[:n]

	spf, writer, err := this.createSpillFile()
	if err != nil {
		return err
	}
	this.spill = append(bySize[n:], spf)

	// The files being merged are no longer in this.spill: close them however the merge ends; on failure the values of the
	// array are discarded (as for any failed spill), as those of the merged files are lost. They are released
	// before the merged file is charged, so the temp space quota is not checked against the peak of a merge, when
	// both are on disk (up to twice the size of the merged files); charging first would fail queries that complete
	// without merging.
	closeMerged := func() {
		for _, m := range merge {
			util.ReleaseTemp(m.f.Name(), m.sz)
			m.f.Close()
			// only set when the merge failed
			if m.current != nil {
				this.releaseMerged(m)
				m.current.Recycle()
				m.current = nil
			}
		}
	}

	for _, m := range merge {
		this.releaseCurrent(m)
		err = m.rewind()
		if err == nil {
			err = this.readMerged(m)
		}
		if err != nil && err != io.EOF {
			closeMerged()
			this.Truncate(nil)
			return spillError(errors.E_VALUE_SPILL_READ, err)
		}
	}

	heap.Init(&merge)
	for merge[0].current != nil {
		m := merge[0]
		err = this.writeSpillValue(spf, writer, m.current, false)
		if err != nil {
			closeMerged()
			this.Truncate(nil)
			return spillError(errors.E_VALUE_SPILL_WRITE, err)
		}
		m.current = nil
		this.releaseMerged(m)
		err = this.readMerged(m)
		if err != nil && err != io.EOF {
			closeMerged()
			this.Truncate(nil)
			return spillError(errors.E_VALUE_SPILL_READ, err)
		}
		heap.Fix(&merge, 0)
	}
	for _, m := range merge {
		spf.write += m.write
		spf.read += m.read
	}
	closeMerged()
	err = this.finishSpillFile(spf, writer)
	if err != nil {
		return err
	}

	logging.Debuga(func() string {
		return fmt.Sprintf("[%p] merged %v spill files to %s (sz:%v) in %v", this, len(merge), spf.f.Name(), spf.sz,
			util.Since(start))
	})
	return nil
}

// Reads the next value of a spill file being merged. The value is tracked by its Size() (like the values in this.mem, and
// unlike the estimate a read tracks), so that releasing it after it is written balances exactly.
func (this *AnnotatedArray) readMerged(m *spillFile) error {
	err := m.nextValue(nil, this.valFunc)
	if err != nil || this.trackMemory == nil {
		return err
	}
	sz := int64(m.current.Size())
	err = this.trackMemory(sz)
	if err != nil {
		return err
	}
	m.mergeTracked = sz
	return nil
}

// A sorted array reads the next value of a spill file before returning the current one, so the value a spill file holds
// when reading stopped early was never returned: release it before the file is reread or closed. An unsorted array returns
// the value a spill file holds, which then belongs to the caller.
func (this *AnnotatedArray) releaseCurrent(m *spillFile) {
	if this.less == nil || m.current == nil {
		return
	}
	if this.trackMemory != nil {
		this.trackMemory(-int64(m.current.Size()))
	}
	m.current.Recycle()
	m.current = nil
}

func (this *AnnotatedArray) releaseMerged(m *spillFile) {
	if m.mergeTracked != 0 {
		this.trackMemory(-m.mergeTracked)
		m.mergeTracked = 0
	}
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
		this.releaseCurrent(this.spill[i])
		err := this.spill[i].rewind()
		if err != nil {
			logging.Debugf("[%p] rewind failed on [%d] %s: %v", this, i, this.spill[i].f.Name(), err)
			return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
		}
		if this.less != nil {
			err = this.spill[i].nextValue(this.trackMemory, this.valFunc)
			if err != nil {
				logging.Debugf("[%p] initial read failed on [%d] %s: %v", this, i, this.spill[i].f.Name(), err)
				return errors.NewValueError(errors.E_VALUE_SPILL_READ, err)
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
		err := this.spill[this.iterator.fileIndex].nextValue(this.trackMemory, this.valFunc)
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
	err := smallest.nextValue(this.trackMemory, this.valFunc)
	if err != io.EOF && err != nil {
		return nil, errors.NewValueError(errors.E_VALUE_SPILL_READ, err), false
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
		this.releaseCurrent(this.spill[i])
		if this.spill[i].f != nil {
			util.ReleaseTemp(this.spill[i].f.Name(), this.spill[i].sz)
			this.spill[i].f.Close()
		}
		this.spill[i].current = nil
	}
	this.spill = nil
	this.spillWriter = nil
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

func (this *AnnotatedArray) Stop() {
	s := this.spill
	for _, sf := range s {
		sf.f.Close()
	}
}
