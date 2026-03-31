//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

/*
Completed requests streaming to disk.

The basic idea is to have enough concurrent activity to fully exercise the I/O and constrain the request throughput with that so
that there isn't an ever-growing backlog to manage.  For this reason each servicer writes its completed request to a shared file.
To limit the bottleneck multiple files are active concurrently - the number was chosen empirically - but this does mean the requests
are not streamed in strict completion order.

The files are GZIP streams of the JSON completed_requests output, rather than the RequestLog structures, since it is intended that
they can be examined outside the engine.  GZIP was picked over ZLIB as the gzip command line utility is commonly available.  A table
of contents to aid reading the files is added to the end after the GZIP stream; the GZIP protocol dictates this must be ignored by
processors so doesn't pose a problem for the gzip command however the '-q' option should be used to suppress the warning it will
emit on encountering the metadata/table of contents (TOC).

Individual files are limited in size based on the size of the raw (uncompressed) data being written to them.  This is to control
the space needed when reading the files.

The active files for streaming to are not part of the managed size nor are they read since for maximum performance they are a single
ZIP stream and cannot therefore be read until the stream is closed.  When closed, they're renamed and included in the managed files
list.

Encryption at rest:

If encryption at rest is enabled, the same request JSON stream is written, but the stream file (`rlstream.<id>`) is stored in CBEF
format with GZIP compression enabled, ensuring the payload on disk is encrypted. When encryption is enabled, the TOC is not appended
to the encrypted stream file. Instead, the TOC is written to a separate metadata file (`metadata.rlstream.<id>`). During archival,
the stream and metadata files are renamed to `local_request_log.<num>` and `metadata.local_request_log.<num>` respectively.

The TOC is written to a separate file so it can remain unencrypted. The TOC must be accessible in plaintext/non-encrypted form
because it is required for certain processing when an archive file is read through system:completed_requests_history.

Users can directly choose to read the archive files from command line utilities. Couchbase's cbcat tool will be the preferred tool
to read encrypted archive files, as cbcat decrypts encrypte files in CBEF format. If the TOC were kept unencrypted within the same
archive file as the encrypted payload, decryption using cbcat tool would fail with errors. Storing the TOC in a separate
metadata file avoids this issue while keeping the archive payload fully encrypted.
*/

package server

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"container/list"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const (
	_MSG_PREFIX                              = "CRS: "
	_REQUEST_LOG_STREAM_FILE                 = "local_request_log."
	_REQUEST_LOG_STREAM_ACTIVE_FILE          = "rlstream."
	_REQUEST_LOG_STREAM_METADATA_FILE        = "metadata." + _REQUEST_LOG_STREAM_FILE
	_REQUEST_LOG_STREAM_ACTIVE_METADATA_FILE = "metadata." + _REQUEST_LOG_STREAM_ACTIVE_FILE
	_STREAM_BUF_SIZE                         = util.KiB * 64
	_ACTIVE_FILES                            = uint64(16)
	_SWEEP_INTERVAL                          = time.Second * 30
	_MAX_RAW_SIZE                            = util.MiB * 100   // maximum raw size before closing (size when cached for reading)
	_MIN_RAW_SIZE                            = util.KiB * 256   // minimum raw size before being considered for initial idle flushing
	_MAX_IDLE_1                              = time.Minute * 10 // idle stream files with at least _MIN_RAW_SIZE closed after this interval
	_MAX_IDLE_2                              = time.Minute * 60 // idle stream files closed after this interval
	_STREAM_MAGIC                            = 0x4352534D       // "CRSM"
	_MAX_CACHE                               = 5                // maximum number of cached files (materialised) for reading
	_RLS_TIMEOUT                             = time.Second * 10 // maximum time to wait writing to the stop channel
)

type requestStreamFile struct {
	sync.Mutex
	encrypt bool
	f       *os.File
	w       *bufio.Writer
	z       *gzip.Writer
	ew      *encryption.CBEFWriter
	encoder *json.Encoder
	index   uint64    // for quick reference
	written uint64    // bytes written before compression etc.
	size    int64     // file size set when closing. This will include the size of the metadata file if stream file is encrypted
	mtime   time.Time // time of last write
	offsets []uint64  // entry offsets in uncompressed data stream
}

func (this *requestStreamFile) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"[%02d] count=%v %v\"", this.index, len(this.offsets), this.mtime.Format("15:04:05.000"))), nil
}

func (this *requestStreamFile) isClosed() bool {
	return this.f == nil
}

// this intercepts the JSON encoder output on its way to the ZIP stream so we have access to the number of bytes produced
func (this *requestStreamFile) Write(p []byte) (int, error) {

	var n int
	var err error

	if this.ew != nil {
		n, err = this.ew.Write(p)
	} else {
		n, err = this.z.Write(p)
	}

	if err == nil {
		this.written += uint64(n)
	}
	return n, err
}

func (this *requestStreamFile) encode(v interface{}) error {
	var err error
	if this.f == nil {
		if err = this.create(); err != nil {
			return err
		}
	}
	if this.encoder != nil {
		off := this.written
		err = this.encoder.Encode(v)
		if err == nil {
			this.offsets = append(this.offsets, off)
			if this.written >= _MAX_RAW_SIZE {
				this.close()
			} else {
				this.mtime = time.Now()
			}
		}
	}
	return err
}

func (this *requestStreamFile) create() error {
	var err error
	this.f, err = os.Create(requestLogStreamActiveFileName(this.index))
	if err != nil {
		return err
	}

	this.encrypt = requestLog.stream.encrypt
	if this.encrypt {
		this.ew, err = encryption.NewCBEFWriterSize(this.f, requestLog.stream.key,
			encryption.CBEF_GZIP, _STREAM_BUF_SIZE)
		if err != nil {
			return err
		}
	} else {
		this.w = bufio.NewWriterSize(this.f, _STREAM_BUF_SIZE)
		this.z = gzip.NewWriter(this.w)
	}

	this.encoder = json.NewEncoder(this)
	this.written = 0
	this.offsets = nil
	this.size = 0
	this.mtime = time.Time{}
	return nil
}

func (this *requestStreamFile) close() {
	this.encoder = nil
	if this.z != nil {
		this.z.Close()
		this.z = nil
	}
	if this.w != nil {
		this.w.Flush()
		this.w = nil
	}
	if this.ew != nil {
		this.ew.Flush()
		this.ew.Close()
		this.ew = nil
	}

	// Create the TOC
	if this.f != nil {
		buf := make([]byte, 0, 16+len(this.offsets)*8)
		for i := range this.offsets {
			buf = binary.BigEndian.AppendUint64(buf, this.offsets[i])
		}
		buf = binary.BigEndian.AppendUint32(buf, _STREAM_MAGIC)
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(this.offsets)))
		buf = binary.BigEndian.AppendUint64(buf, this.written)

		if !this.encrypt {
			// write trailer after ZIP stream
			// non-ZIP bytes are ignored by command-line utilities accessing the file directly
			this.f.Write(buf)
		} else {
			// Write TOC to a separate metadata file
			meta, _ := os.Create(requestMetadataActiveFileName(this.index))
			meta.Write(buf)
			msz, _ := meta.Seek(0, io.SeekEnd)
			this.size += int64(msz)
			meta.Close()
		}

		// Calculate the stream file size
		fSz, _ := this.f.Seek(0, os.SEEK_END)
		this.size += int64(fSz)
		this.f.Close()
		this.f = nil
	}
}

func requestLogStreamActiveFileName(num uint64) string {
	return fmt.Sprintf("%s/%s%d", ffdc.GetPath(), _REQUEST_LOG_STREAM_ACTIVE_FILE, num)
}

func requestLogStreamFileName(num uint64) string {
	return fmt.Sprintf("%s/%s%d", ffdc.GetPath(), _REQUEST_LOG_STREAM_FILE, num)
}

func requestMetadataActiveFileName(num uint64) string {
	return fmt.Sprintf("%s/%s%d", ffdc.GetPath(), _REQUEST_LOG_STREAM_ACTIVE_METADATA_FILE, num)
}

func requestMetadataFileName(num uint64) string {
	return fmt.Sprintf("%s/%s%d", ffdc.GetPath(), _REQUEST_LOG_STREAM_METADATA_FILE, num)
}

func removeArchiveFiles(num uint64) {
	os.Remove(requestLogStreamFileName(num))
	os.Remove(requestMetadataFileName(num))
}

func removeActiveFiles(num uint64) {
	os.Remove(requestLogStreamActiveFileName(num))
	os.Remove(requestMetadataActiveFileName(num))
}

// info about a managed file (not an active stream file)
type fileInfo struct {
	num       uint64
	size      uint64
	count     int // may be -1 if unknown
	encrypted bool
}

type requestLogStream struct {
	sync.Mutex

	stop chan bool

	configSize uint64     // target size to remain below
	size       uint64     // maintained sum of all file sizes
	files      *list.List // fileInfo
	filesLock  sync.Mutex

	cache readCache

	// writing
	active       uint64 // used to determine if active; for atomic operations
	streamFiles  []*requestStreamFile
	streamCount  uint64 // stats
	streamErrors uint64 // stats
	fileNum      uint64 // last known/generated file number

	encrypt bool
	key     *encryption.EaRKey
}

func (this *requestLogStream) String() string {
	return fmt.Sprintf("{active:%v,stop:%d,configSize:%v,files:%p,streamFiles:%d}",
		this.active, len(this.stop), this.configSize, this.files, len(this.streamFiles))
}

func (this *requestLogStream) stopCapture() bool {
	this.Lock()
	if this.active == 0 || this.stop == nil {
		logging.Infof(_MSG_PREFIX + "Not active.")
		this.Unlock()
		return false
	}
	atomic.StoreUint64(&this.active, 0)
	select {
	case this.stop <- true:
	case <-time.After(_RLS_TIMEOUT):
		logging.Errorf(_MSG_PREFIX+"Timeout writing to stop channel. %v", this.String())
		logging.DumpAllStacks(logging.DEBUG, "")
		this.Unlock()
		return false
	}
	this.filesLock.Lock()
	for i := range this.streamFiles {
		if file := this.streamFiles[i]; file != nil {
			file.Lock()
			file.close()
			if file.size > 0 {
				this.archive(file, false)
			} else {
				file.Unlock()
			}
		}
	}
	this.filesLock.Unlock()
	close(this.stop)
	this.stop = nil
	this.streamFiles = nil
	this.configSize = 0
	logging.Debugf("Stopped: %v", this)
	this.Unlock()
	return true
}

func (this *requestLogStream) startCapture(newSize uint64) bool {
	this.Lock()
	if this.active != 0 {
		logging.Errorf(_MSG_PREFIX + "Already active.")
		this.Unlock()
		return false
	}
	this.configSize = newSize
	this.loadFiles()
	this.stop = make(chan bool, 1)
	go this.spaceManagementProc(this.stop)
	this.streamFiles = make([]*requestStreamFile, _ACTIVE_FILES)
	for i := range this.streamFiles {
		this.streamFiles[i] = &requestStreamFile{index: uint64(i)}
	}
	this.streamCount = 0
	this.streamErrors = 0
	atomic.StoreUint64(&this.active, 1)
	logging.Debugf("Started: %v", this)
	this.Unlock()
	return true
}

// populates the managed files information
func (this *requestLogStream) loadFiles() {
	active := atomic.LoadUint64(&this.active) == 0
	this.filesLock.Lock()
	this.files = list.New()
	sz := uint64(0)
	var acts []uint64
	var actsz []uint64
	// list the stream files
	d, err := os.Open(ffdc.GetPath())
	if err == nil {
		fi := make([]fileInfo, 0, 128)
		for {
			ents, err := d.ReadDir(10)
			if err == nil {
				for i := range ents {
					if ents[i].IsDir() {
						continue
					}

					if strings.HasPrefix(ents[i].Name(), _REQUEST_LOG_STREAM_FILE) {
						numStr := ents[i].Name()[len(_REQUEST_LOG_STREAM_FILE):]
						num, err := strconv.ParseUint(numStr, 10, 64)
						if err != nil {
							continue
						}
						fsz := uint64(0)
						if info, err := ents[i].Info(); err == nil {
							fsz = uint64(info.Size())
						}

						f, err := os.Open(requestLogStreamFileName(num))
						if err != nil {
							logging.Warnf(_MSG_PREFIX+"Failed to open file during loading %v (%v). Skipping file.",
								ents[i].Name(), err)
							removeArchiveFiles(num)
							continue
						}
						encrypted := isEncrypted(f)
						f.Close()

						if encrypted {
							// Include size of metadata file
							metadataFile := requestMetadataFileName(num)
							if stat, err := os.Stat(metadataFile); err == nil {
								fsz += uint64(stat.Size())
							} else {
								logging.Warnf(_MSG_PREFIX+"Failed to open metadata file during loading %v (%v). Skipping file.",
									metadataFile, err)
								removeArchiveFiles(num)
								continue
							}
						}

						sz += fsz
						fi = append(fi, fileInfo{num: num, size: fsz, count: -1, encrypted: encrypted})
					} else if !active && strings.HasPrefix(ents[i].Name(), _REQUEST_LOG_STREAM_ACTIVE_FILE) {
						numStr := ents[i].Name()[len(_REQUEST_LOG_STREAM_ACTIVE_FILE):]
						num, err := strconv.ParseUint(numStr, 10, 64)
						if err != nil {
							continue
						}
						acts = append(acts, num)
						fsz := uint64(0)
						if info, err := ents[i].Info(); err == nil {
							fsz = uint64(info.Size())
						}
						actsz = append(actsz, fsz)
					}
				}
			}
			if err != nil || len(ents) < 10 {
				break
			}
		}
		d.Close()
		// rename old active files (if any) and include them in the file info list
		for i := range acts {
			num := atomic.AddUint64(&this.fileNum, 1)

			activeFile := requestLogStreamActiveFileName(acts[i])
			f, err := os.Open(activeFile)
			if err != nil {
				logging.Warnf(_MSG_PREFIX+"Failed to find/open past active file %v file during loading: %v. Skipping file.",
					acts[i], err)
				removeActiveFiles(acts[i])
				continue
			}

			encrypted := isEncrypted(f)

			// Check if there is valid TOC/metadata for the stream file
			var metadataFile *os.File // The file that contains the TOC/metadata
			if encrypted {
				m, err := os.Open(requestMetadataActiveFileName(acts[i]))
				if err != nil {
					f.Close()
					logging.Warnf(_MSG_PREFIX+"Failed to find/open past active metadata file %v during loading. Skipping file.",
						acts[i])
					removeActiveFiles(acts[i])
					continue
				}
				metadataFile = m
			} else {
				metadataFile = f
			}

			// Check the validity of the TOC
			_, err = metadataFile.Seek(-16, io.SeekEnd)
			var validMagic bool
			if err == nil {
				buf := make([]byte, 4)
				_, err = io.ReadFull(metadataFile, buf)
				if err == nil {
					if binary.BigEndian.Uint32(buf) == _STREAM_MAGIC {
						validMagic = true
					}
				}
			}

			var metadataSize uint64
			if encrypted {
				if stat, err := metadataFile.Stat(); err == nil {
					metadataSize = uint64(stat.Size())
				}
				metadataFile.Close()
			}
			f.Close()

			if !validMagic {
				logging.Warnf(_MSG_PREFIX+"No valid metadata found for past active file %v during loading. Skipping file.", acts[i])
				removeActiveFiles(acts[i])
				continue
			}

			if e := os.Rename(activeFile, requestLogStreamFileName(num)); e != nil {
				logging.Warnf(_MSG_PREFIX+"Failed to rename past active file %v to archive file %v. Skipping file.", acts[i], num)
				removeActiveFiles(acts[i])
				continue
			}

			if encrypted {
				if e := os.Rename(requestMetadataActiveFileName(acts[i]), requestMetadataFileName(num)); e != nil {
					logging.Warnf(_MSG_PREFIX+
						"Failed to rename past metadata active file %v to metadata archive file %v (%v). Skipping file.",
						acts[i], num, e)
					removeActiveFiles(acts[i])
					continue
				}
			}

			sz += actsz[i]
			fi = append(fi, fileInfo{num: num, size: actsz[i] + metadataSize, count: -1, encrypted: encrypted})

		}
		if len(fi) > 0 {
			sort.Slice(fi, func(i int, j int) bool {
				return fi[i].num < fi[j].num
			})
			for i := range fi {
				this.files.PushBack(fi[i])
			}
		}
	}
	if atomic.LoadUint64(&this.active) == 0 {
		atomic.StoreUint64(&this.size, sz)
		if e := this.files.Back(); e != nil {
			atomic.StoreUint64(&this.fileNum, e.Value.(fileInfo).num)
		}
	}
	this.filesLock.Unlock()
}

func (this *requestLogStream) encode(i interface{}) {
	if this.streamFiles == nil || atomic.LoadUint64(&this.active) == 0 {
		return
	}
	num := atomic.AddUint64(&this.streamCount, 1) % uint64(len(this.streamFiles))
	file := this.streamFiles[num]
	if file == nil || atomic.LoadUint64(&this.active) == 0 {
		return
	}
	file.Lock()
	err := file.encode(i)
	if err != nil {
		logging.Errorf(_MSG_PREFIX+"Encoding failed: %v", err)
		atomic.AddUint64(&this.streamErrors, 1)
	}
	if file.isClosed() && file.size > 0 {
		this.archive(file, true)
	} else {
		file.Unlock()
	}
}

// expects the file to be locked and will unlock when done
func (this *requestLogStream) archive(file *requestStreamFile, lockFilesList bool) uint64 {
	var fi fileInfo

	fi.num = atomic.AddUint64(&this.fileNum, 1)
	fi.size = uint64(file.size)
	fi.count = int(len(file.offsets))

	file.size = 0
	file.mtime = time.Time{}

	if e := os.Rename(requestLogStreamActiveFileName(file.index), requestLogStreamFileName(fi.num)); e != nil {
		logging.Warnf(_MSG_PREFIX+"Failed to rename active file %v to archive file %v (%v)", file.index, fi.num, e)
	}

	if this.encrypt {
		if e := os.Rename(requestMetadataActiveFileName(file.index), requestMetadataFileName(fi.num)); e != nil {
			logging.Warnf(_MSG_PREFIX+"Failed to rename metadata active file %v to metadata archive file %v (%v)", file.index,
				fi.num, e)
		}
	}

	file.Unlock()

	if lockFilesList {
		this.filesLock.Lock()
	}
	atomic.AddUint64(&this.size, fi.size)
	this.files.PushBack(fi)
	if lockFilesList {
		this.filesLock.Unlock()
	}
	return fi.num
}

// background routine handling space management
func (this *requestLogStream) spaceManagementProc(stop chan bool) {
	logging.Infof(_MSG_PREFIX+"[%p] Space management started.", stop)
	defer func() {
		e := recover()
		if e != nil {
			logging.Stackf(logging.WARN, _MSG_PREFIX+"Panic in space management: %v", e)
			var ok bool
			select {
			case _, ok = <-stop: // detect a closed channel
				ok = false // if it was successful reading it means we should be stopping and therefore not restarting here
			default:
				ok = true
			}
			if ok {
				go this.spaceManagementProc(stop) // restart after a panic
			}
		}
	}()
	ticker := time.NewTicker(_SWEEP_INTERVAL)
	stopping := false
	for !stopping {
		select {
		case <-ticker.C:
		case <-stop:
			stopping = true
		}
		if this.files == nil || atomic.LoadUint64(&this.active) == 0 {
			continue
		}

		// check for and close idle files
		mark := time.Now()
		for i := 0; i < len(this.streamFiles); i++ {
			file := this.streamFiles[i]
			file.Lock()
			if !file.isClosed() && !file.mtime.IsZero() {
				idle := mark.Sub(file.mtime)
				if (idle > _MAX_IDLE_1 && file.written > _MIN_RAW_SIZE) || idle > _MAX_IDLE_2 {
					file.close()
					if file.size > 0 {
						this.archive(file, true)
						file = nil
					}
				}
			}
			if file != nil {
				file.Unlock()
			}
		}

		d, err := os.Open(ffdc.GetPath())
		if err == nil {
			fi := make(map[uint64]bool)
			for {
				ents, err := d.ReadDir(10)
				if err == nil {
					for i := range ents {
						if !ents[i].IsDir() && strings.HasPrefix(ents[i].Name(), _REQUEST_LOG_STREAM_FILE) {
							numStr := ents[i].Name()[len(_REQUEST_LOG_STREAM_FILE):]
							num, err := strconv.ParseUint(numStr, 10, 64)
							if err != nil {
								continue
							}
							fi[num] = true
						}
					}
				}
				if err != nil || len(ents) < 10 {
					break
				}
			}
			d.Close()
			if atomic.LoadUint64(&this.active) != 0 {
				released := uint64(0)
				this.filesLock.Lock()
				// remove file records for files that don't exist (external removals)
				if this.files.Len() > 0 {
					var nit *list.Element
					for it := this.files.Front(); it != nil; it = nit {
						nit = it.Next()
						itfi := it.Value.(fileInfo)
						if _, ok := fi[itfi.num]; !ok {
							released += itfi.size
							if atomic.LoadUint64(&this.size) >= itfi.size {
								atomic.AddUint64(&this.size, ^(itfi.size - 1))
							} else {
								atomic.StoreUint64(&this.size, 0)
							}
							this.files.Remove(it)
						}
					}
				}
				// remove files to bring size back down to configured maximum
				if atomic.LoadUint64(&this.size) > this.configSize {
					var nit *list.Element
					for it := this.files.Front(); it != nil && atomic.LoadUint64(&this.size) > this.configSize; it = nit {
						nit = it.Next()
						itfi := it.Value.(fileInfo)
						if _, ok := fi[itfi.num]; ok {
							released += itfi.size
							if atomic.LoadUint64(&this.size) >= itfi.size {
								atomic.AddUint64(&this.size, ^(itfi.size - 1))
							} else {
								atomic.StoreUint64(&this.size, 0)
							}
							this.files.Remove(it)
							os.Remove(requestLogStreamFileName(itfi.num))
							// If the stream file is not enrypted, the metadata file is not present. so this would be a no-op
							os.Remove(requestMetadataFileName(itfi.num))
						}
					}
				}
				this.filesLock.Unlock()
				if released > 0 {
					logging.Infof(_MSG_PREFIX+"Space management freed %v", ffdc.Human(released))
				}
			}
		}
	}
	logging.Infof(_MSG_PREFIX+"[%p] Space management stopped.", stop)
}

// reading

func (this *requestLogStream) entryCounts() []uint64 {
	if this.files == nil {
		this.loadFiles()
		if this.files == nil {
			return nil
		}
	}
	res := make([]uint64, 0, 128)
	this.filesLock.Lock()
	for it := this.files.Front(); it != nil; it = it.Next() {
		itfi := it.Value.(fileInfo)
		if itfi.count == -1 {

			var f *os.File
			var e error

			// attempt to update the file information
			if !itfi.encrypted {
				f, e = os.Open(requestLogStreamFileName(itfi.num))
			} else {
				f, e = os.Open(requestMetadataFileName(itfi.num))
			}

			if e == nil {
				if _, e = f.Seek(-16, os.SEEK_END); e == nil {
					buf := make([]byte, 8)
					_, e = f.Read(buf)
					if e == nil && binary.BigEndian.Uint32(buf) == _STREAM_MAGIC {
						itfi.count = int(binary.BigEndian.Uint32(buf[4:]))
						it.Value = itfi
					}
				}
				f.Close()
			}
		}
		if itfi.count != -1 {
			res = append(res, itfi.num)
			res = append(res, uint64(itfi.count))
		}
	}
	this.filesLock.Unlock()
	return res
}

type readCacheEntry struct {
	sync.Mutex
	num     uint64
	raw     []byte   // when read in
	offsets []uint64 // when read in
}

func (this *readCacheEntry) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"file=%d size=%d count=%d\"", this.num, len(this.raw), len(this.offsets))), nil
}

func (this *readCacheEntry) read(rec uint64) interface{} {
	if rec >= uint64(len(this.offsets)) {
		return nil
	}
	off := this.offsets[rec]
	if off >= uint64(len(this.raw)) {
		return nil
	}
	r := bytes.NewReader(this.raw[off:])
	d := json.NewDecoder(r)
	if !d.More() {
		return nil
	}
	var ent interface{}
	err := d.Decode(&ent)
	if err != nil {
		return nil
	}
	return ent
}

type readCache struct {
	sync.Mutex
	cache *list.List
}

func (this *readCache) MarshalJSON() ([]byte, error) {
	var a []interface{}
	if this.cache != nil {
		a = make([]interface{}, 0, this.cache.Len())
		for it := this.cache.Front(); it != nil; it = it.Next() {
			a = append(a, it.Value.(*readCacheEntry))
		}
	}
	return json.Marshal(a)
}

func (this *readCache) get(num uint64) *readCacheEntry {
	this.Lock()
	var ce *readCacheEntry
	if this.cache == nil {
		this.cache = list.New()
	}
	for it := this.cache.Front(); it != nil; it = it.Next() {
		ce = it.Value.(*readCacheEntry)
		if ce.num == num {
			ce.Lock()
			this.cache.MoveToFront(it) // MRU
			break
		}
		ce = nil
	}
	this.Unlock()
	return ce
}

func (this *readCache) add(num uint64) *readCacheEntry {
	this.Lock()
	for this.cache.Len() >= _MAX_CACHE {
		this.cache.Remove(this.cache.Back())
	}
	ce := &readCacheEntry{num: num}
	this.cache.PushFront(ce)
	ce.Lock()
	this.Unlock()
	return ce
}

func (this *requestLogStream) load(num uint64) (*readCacheEntry, error) {
	ce := this.cache.get(num)
	if ce == nil {
		streamFile, err := os.Open(requestLogStreamFileName(num))
		if err != nil {
			return nil, err
		}

		defer streamFile.Close()
		encrypted := isEncrypted(streamFile)

		// Read and validate the metadata/ TOC
		var metadataFile io.ReadSeekCloser
		if encrypted {
			metadataFile, err = os.Open(requestMetadataFileName(num))
			if err != nil {
				return nil, err
			}
			defer metadataFile.Close()
		} else {
			metadataFile = streamFile
		}

		if _, err := metadataFile.Seek(-16, io.SeekEnd); err != nil {
			return nil, err
		}

		buf := make([]byte, 16)
		if _, err := metadataFile.Read(buf); err != nil || binary.BigEndian.Uint32(buf) != _STREAM_MAGIC {
			if err == nil {
				err = io.EOF
			}
			return nil, err
		}

		if !encrypted {
			if _, err := metadataFile.Seek(0, io.SeekStart); err != nil {
				return nil, err
			}
		}

		// Setup reader to read the stream file
		var r io.Reader
		if encrypted {
			er, err := encryption.NewCBEFReader(streamFile, func(keyId string) *encryption.EaRKey {
				return requestLog.stream.key
			})
			if err != nil {
				return nil, err
			}
			r = er
			defer er.Close()
		} else {
			b := bufio.NewReaderSize(streamFile, _STREAM_BUF_SIZE)
			z, err := gzip.NewReader(b)
			if err != nil {
				return nil, err
			}
			z.Multistream(false) // we have trailing data that gzip should not attempt to interpret
			r = z
			defer z.Close()
		}

		raw := make([]byte, binary.BigEndian.Uint64(buf[8:]))
		start := 0
		for start < len(raw) {
			n, err := r.Read(raw[start:])
			if n > 0 {
				start += n
			} else if err != nil && err != io.EOF {
				raw = nil
				if err == nil {
					err = io.EOF
				}
				return nil, err
			} else {
				break
			}
		}
		raw = raw[:start]
		ce = this.cache.add(num)
		ce.raw = raw
		ce.offsets = make([]uint64, binary.BigEndian.Uint32(buf[4:]))
		off := make([]byte, 8)
		// have to seek to the start of the offset table as despite bufio.Reader implementing io.ByteReader, zlib.Reader doesn't
		// always leave the reader correctly positioned.  Hopefully most of the time this is a no-op as the seek position is the
		// current position
		pos := int64(len(ce.offsets)+2) * 8
		if _, err := metadataFile.Seek(-pos, os.SEEK_END); err != nil {
			return nil, err
		}

		for i := range ce.offsets {
			_, err := metadataFile.Read(off)
			if err != nil {
				return nil, err
			}
			ce.offsets[i] = binary.BigEndian.Uint64(off)
		}
	}
	return ce, nil
}

// External API

// returns a _flat_ (for memory efficiency) array of file number & record count pairs
// this is used to produce the system namespace "index" for the streamed history
func RequestsFileStreamFileInfo() []uint64 {
	return requestLog.stream.entryCounts()
}

func RequestsFileStreamRead(fileNum uint64, skip uint64, count uint64, user string, fn func(map[string]interface{}) bool) error {
	ce, err := requestLog.stream.load(fileNum)
	if err != nil {
		return err
	}
	max := uint64(len(ce.offsets))
	if skip >= max {
		ce.Unlock()
		return nil
	}
	if count == 0 {
		count = max
	}
	if user == "" {
		for i := skip; i < max && count > 0; i++ {
			if v := ce.read(i); v != nil {
				if m, ok := v.(map[string]interface{}); ok {
					if !fn(m) {
						ce.Unlock()
						return nil
					}
					count--
				}
			}
		}
	} else {
		for i := uint64(0); i < max && count > 0; i++ {
			if v := ce.read(i); v != nil {
				if m, ok := v.(map[string]interface{}); ok {
					if m["users"] == user {
						if skip == 0 {
							if !fn(m) {
								ce.Unlock()
								return nil
							}
							count--
						} else {
							skip--
						}
					}
				}
			}
		}
	}
	ce.Unlock()
	return nil
}

func RequestsFileStreamStats(stats map[string]interface{}) {
	if stats != nil {
		requestLog.stream.Lock()
		if requestLog.stream.active != 0 {
			m := make(map[string]interface{}, 8)
			a := make([]interface{}, 0, len(requestLog.stream.streamFiles))
			for i := range requestLog.stream.streamFiles {
				if !requestLog.stream.streamFiles[i].isClosed() {
					a = append(a, requestLog.stream.streamFiles[i])
				}
			}
			m["active"] = a
			m["count"] = requestLog.stream.streamCount
			m["errors"] = requestLog.stream.streamErrors
			m["size"] = requestLog.stream.size
			m["config_size"] = requestLog.stream.configSize
			m["file_num"] = requestLog.stream.fileNum
			m["files"] = requestLog.stream.files.Len()
			m["cache"] = &requestLog.stream.cache
			requestLog.stream.Unlock()

			bytes, err := json.Marshal(m)
			if err == nil {
				var v interface{}
				if json.Unmarshal(bytes, &v) == nil {
					stats["request_log_stream"] = v
				}
			}
		} else {
			requestLog.stream.Unlock()
		}
	}
}

func RequestsFileStreamSize() uint64 {
	// size is in MiB
	return (requestLog.stream.configSize / util.MiB)
}

func RequestsSetFileStreamSize(size int64) {
	if size <= 0 {
		if requestLog.stream.stopCapture() {
			logging.Infof(_MSG_PREFIX+"Stopped. (count: %v errors: %v)", requestLog.stream.streamCount,
				requestLog.stream.streamErrors)
		}
		RequestsRemoveHandler("stream")
	} else {
		// size is in MiB
		if requestLog.stream.startCapture(uint64(size * util.MiB)) {
			RequestsAddHandler(streamToFile, "stream")
			logging.Infof(_MSG_PREFIX+"Started. Retaining approx. %v", ffdc.Human(RequestsFileStreamSize()*util.MiB))
		} else {
			requestLog.stream.stopCapture()
			if requestLog.stream.startCapture(uint64(size * util.MiB)) {
				logging.Infof(_MSG_PREFIX+"Restarted. Retaining approx. %v", ffdc.Human(RequestsFileStreamSize()*util.MiB))
			}
		}
	}
}

// hook into standard processing
func streamToFile(e *RequestLogEntry) {
	if atomic.LoadUint64(&requestLog.stream.active) != 0 {
		// servicer is responsible for the formatting
		// this is relatively expensive so if delegated we'd need an equivalent number (or more) background routines to keep up
		// (and they'd contend for CPU time etc.)
		requestLog.stream.encode(e.Format(true, false, util.GetDurationStyle()))
	}
}

func isEncrypted(file *os.File) bool {
	var encrypted bool
	if encryption.IsCBEFReader(file) {
		encrypted = true
	}

	file.Seek(0, io.SeekStart)
	return encrypted
}
