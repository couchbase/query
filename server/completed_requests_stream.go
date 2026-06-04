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
format with ZLIB compression enabled, ensuring the payload on disk is encrypted. // The data is ZLIB compressed, as cbcat tool
currently does not support GZIP and only supports ZLIB compression type. When encryption is enabled, the TOC is not appended
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
	go_errors "errors"
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
	"github.com/couchbase/query/encryption/keymgmt"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/ffdc"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/util"
)

const (
	_MSG_PREFIX                                = "CRS: "
	_REQUEST_LOG_STREAM_FILE                   = "local_request_log."
	_REQUEST_LOG_STREAM_ACTIVE_FILE            = "rlstream."
	_REQUEST_LOG_STREAM_METADATA_FILE          = "metadata." + _REQUEST_LOG_STREAM_FILE
	_REQUEST_LOG_STREAM_ACTIVE_METADATA_FILE   = "metadata." + _REQUEST_LOG_STREAM_ACTIVE_FILE
	_STREAM_BUF_SIZE                           = util.KiB * 64
	_ACTIVE_FILES                              = uint64(16)
	_SWEEP_INTERVAL                            = time.Second * 30
	_CRS_CLEANUP_INTERVAL                      = 15 * time.Minute
	_MAX_RAW_SIZE                              = util.MiB * 100   // maximum raw size before closing (size when cached for reading)
	_MIN_RAW_SIZE                              = util.KiB * 256   // minimum raw size before being considered for initial idle flushing
	_MAX_IDLE_1                                = time.Minute * 10 // idle stream files with at least _MIN_RAW_SIZE closed after this interval
	_MAX_IDLE_2                                = time.Minute * 60 // idle stream files closed after this interval
	_STREAM_MAGIC                              = 0x4352534D       // "CRSM"
	_MAX_CACHE                                 = 5                // maximum number of cached files (materialised) for reading
	_RLS_TIMEOUT                               = time.Second * 10 // maximum time to wait writing to the stop channel
	_STREAM_UNSET_KEY_ID                       = "UNSET"          // indicates that the stream file is not encrypted; this is used to avoid unnecessary calls to the encryption provider when encryption is disabled
	_STREAM_REENCRYPT_ARCHIVE_FILE_NAME_PREFIX = "reencrypt_" + _REQUEST_LOG_STREAM_FILE
)

var crsKeyDataType = encryption.KeyDataType{TypeName: encryption.LOG_KEY_DATATYPE}

type requestStreamFile struct {
	sync.Mutex
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

	keyID              string
	encryptionProvider encryption.EncryptionProvider
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
	if this.encryptionProvider == nil {
		return errors.NewEncryptionError(errors.E_NO_ENCRYPTION_MANAGER, nil)
	}

	encryptionKey, err1 := this.encryptionProvider.GetActiveKey(crsKeyDataType)
	if err1 != nil {
		return err1
	}

	var err error
	this.f, err = os.Create(requestLogStreamActiveFileName(this.index))
	if err != nil {
		return err
	}

	if encryptionKey != nil {
		this.ew, err = encryption.NewCBEFWriterSize(this.f, encryptionKey, encryption.CBEF_ZLIB, _STREAM_BUF_SIZE)
		if err != nil {
			// If any error occurs during active stream file creation, truncate and set this.f to nil
			// The trigger to create the stream file is requestStreamFile.f being nil.
			// So, if an error occurs during creation, setting this.f to nil allows the next request to trigger a new attempt at
			//  creating the stream file.
			// We do not need to delete the file on disk as the next attempt to create the file, will not raise an error on
			// os.Create(), but will rather just truncate the existing file
			this.f.Truncate(0)
			this.f.Close()
			this.f = nil
			this.ew = nil
			return err
		}
		this.keyID = encryptionKey.Id
	} else {
		this.keyID = encryption.UNENCRYPTED_KEY_ID
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

		if this.keyID == encryption.UNENCRYPTED_KEY_ID {
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

// Pass a unique UUID to embed in the staging file name so that a new transformation attempt on the same archive file does not
// reuse the exact same path and get accidentally deleted by the background orphan file cleanup routine
func requestLogStreamTransformFileName(num uint64, id string) string {
	return fmt.Sprintf("%s/%s%d.%s", ffdc.GetPath(), _STREAM_REENCRYPT_ARCHIVE_FILE_NAME_PREFIX, num, id)
}

func logFilePath(fileName string) string {
	return fmt.Sprintf("%s/%s", ffdc.GetPath(), fileName)
}

// Will acquire the requestLogStream's orphanLock for tracking files that failed to be removed. Will unlock after
func removeArchiveFiles(num uint64, stream *requestLogStream) {
	removeCRSFile(requestLogStreamFileName(num), stream)
	os.Remove(requestMetadataFileName(num))
}

// Will acquire the requestLogStream's orphanLock for tracking files that failed to be removed. Will unlock after

func removeActiveFiles(num uint64, stream *requestLogStream) {
	removeCRSFile(requestLogStreamActiveFileName(num), stream)
	os.Remove(requestMetadataActiveFileName(num))
}

// Use this to delete any sensitive files
// Will acquire the requestLogStream's orphanLock for tracking files that failed to be removed. Will unlock after
func removeCRSFile(path string, stream *requestLogStream) {
	err := os.Remove(path)
	if err == nil || go_errors.Is(err, os.ErrNotExist) {
		return
	}

	err = os.Truncate(path, 0)
	if err == nil || go_errors.Is(err, os.ErrNotExist) {
		return
	}

	f, err := os.Open(path)
	keyid := encryption.UNENCRYPTED_KEY_ID
	if err == nil {
		encrypted, id := encryption.GetKeyIdFromCBEF(f)
		if encrypted {
			keyid = id
		}
		f.Close()
	}

	stream.orphanLock.Lock()
	if stream.orphanFiles == nil {
		stream.orphanFiles = make([]crsOrphanFile, 0, 8)
	}

	stream.orphanFiles = append(stream.orphanFiles, crsOrphanFile{keyID: keyid, path: path})
	logging.Warnf(_MSG_PREFIX+"Failed to remove file %v encrypted with key id %+q: %v. Tracking as orphan file.", path, keyid,
		err)
	stream.orphanLock.Unlock()
}

// info about a managed file (not an active stream file)
type fileInfo struct {
	num              uint64 // This should not be changed once set
	size             uint64
	count            int // may be -1 if unknown
	currKeyID        string
	targetKeyID      string
	fileReadingCount int           // number of active readers of the file on disk
	operation        archiveFileOp // The maintenance operation being performed on this file
	lock             sync.RWMutex
}

func newBaseFileInfo(num uint64, size uint64, count int, currentKeyID string) *fileInfo {
	fi := &fileInfo{num: num, size: size, count: count, currKeyID: _STREAM_UNSET_KEY_ID, targetKeyID: _STREAM_UNSET_KEY_ID}

	if currentKeyID != _STREAM_UNSET_KEY_ID {
		fi.currKeyID = currentKeyID
	}
	return fi
}

func (this *fileInfo) getSize(lock bool) uint64 {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.size
}

func (this *fileInfo) getCount(lock bool) int {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.count
}

func (this *fileInfo) getCurrKeyID(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.currKeyID
}

func (this *fileInfo) getTargetKeyID(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.targetKeyID
}

func (this *fileInfo) getFileReadingCount(lock bool) int {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.fileReadingCount
}

func (this *fileInfo) setCount(lock bool, count int) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.count = count
}

func (this *fileInfo) setTargetKeyID(lock bool, targetKeyID string) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.targetKeyID = targetKeyID
}

func (this *fileInfo) setSize(lock bool, size uint64) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.size = size
}

func (this *fileInfo) resetAfterTransform(lock bool, newCurrKeyID string) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.targetKeyID = _STREAM_UNSET_KEY_ID
	this.currKeyID = newCurrKeyID
}

func (this *fileInfo) getCurrentKeyID(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.currKeyID
}

func (this *fileInfo) activeMaintenanceOperation() archiveFileOp {
	this.lock.RLock()
	op := this.operation
	this.lock.RUnlock()
	return op
}

func (this *fileInfo) beginLoad() bool {
	var beginLoad bool
	this.lock.Lock()
	if this.operation == _ARCHIVE_NONE || this.operation == _ARCHIVE_TRANSFORMING {
		this.fileReadingCount++
		beginLoad = true
	}
	this.lock.Unlock()
	return beginLoad
}

func (this *fileInfo) endLoad() {
	this.lock.Lock()
	this.fileReadingCount--
	this.lock.Unlock()
}

func (this *fileInfo) beginTransform() bool {
	var beginTransform bool
	this.lock.Lock()
	if this.operation == _ARCHIVE_NONE {
		this.operation = _ARCHIVE_TRANSFORMING
		beginTransform = true
	}
	this.lock.Unlock()
	return beginTransform
}

func (this *fileInfo) endTransform() {
	this.lock.Lock()
	if this.operation == _ARCHIVE_TRANSFORMING {
		this.operation = _ARCHIVE_NONE
	}
	this.lock.Unlock()
}

func (this *fileInfo) beginDelete() bool {
	var startDelete bool
	this.lock.Lock()
	if this.operation == _ARCHIVE_NONE && this.fileReadingCount == 0 {
		this.operation = _ARCHIVE_DELETING
		startDelete = true
	}
	this.lock.Unlock()
	return startDelete
}

func (this *fileInfo) endDelete() {
	this.lock.Lock()
	if this.operation == _ARCHIVE_DELETING {
		this.operation = _ARCHIVE_NONE
	}
	this.lock.Unlock()
}

type archiveFileOp int

const (
	_ARCHIVE_NONE archiveFileOp = iota
	_ARCHIVE_DELETING
	_ARCHIVE_TRANSFORMING
)

type crsOrphanFile struct {
	keyID string
	path  string
}

type requestLogStream struct {
	sync.Mutex

	bootstrapped bool

	stop chan bool

	configSize uint64 // target size to remain below
	size       uint64 // maintained sum of all file sizes

	// In memory list of archive files on disk. Will be the source of truth of archive files for the purpose of key management
	// and file operations
	files     *list.List // entries are of type *fileInfo
	filesLock sync.Mutex

	cache readCache

	// writing
	active uint64 // used to determine if active; for atomic operations

	// In memory list of active files on disk. Will be the source of truth of active files for the purpose of key management and
	// file operations
	streamFiles  []*requestStreamFile
	streamCount  uint64 // stats
	streamErrors uint64 // stats
	fileNum      uint64 // last known/generated file number

	encryptionProvider encryption.EncryptionProvider

	// List of files that are deemed as orphans due to errors
	// Orphan files are considered in key tracking to keep key-in-use reporting correct even when file operations fail
	orphanFiles []crsOrphanFile
	orphanLock  sync.RWMutex
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

	if len(this.streamFiles) == 0 {
		this.streamFiles = make([]*requestStreamFile, _ACTIVE_FILES)
		for i := range this.streamFiles {
			this.streamFiles[i] = &requestStreamFile{index: uint64(i), keyID: _STREAM_UNSET_KEY_ID,
				encryptionProvider: this.encryptionProvider}
		}
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
	if this.bootstrapped { // Expects requestLogStream to be locked
		return
	}

	active := atomic.LoadUint64(&this.active) == 0
	this.filesLock.Lock()
	this.files = list.New()
	sz := uint64(0)
	var acts []uint64
	var actsz []uint64
	// list the stream files
	d, err := os.Open(ffdc.GetPath())
	if err == nil {
		fi := make([]*fileInfo, 0, 128)
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
							removeArchiveFiles(num, this)
							continue
						}
						encrypted, keyId := isEncrypted(f)
						f.Close()

						if encrypted {
							// Include size of metadata file
							metadataFile := requestMetadataFileName(num)
							if stat, err := os.Stat(metadataFile); err == nil {
								fsz += uint64(stat.Size())
							} else {
								logging.Warnf(_MSG_PREFIX+"Failed to open metadata file during loading %v (%v). Skipping file.",
									metadataFile, err)
								removeArchiveFiles(num, this)
								continue
							}
						}

						sz += fsz
						ffdcFi := newBaseFileInfo(num, fsz, -1, keyId)
						fi = append(fi, ffdcFi)
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
					} else if strings.HasPrefix(ents[i].Name(), _STREAM_REENCRYPT_ARCHIVE_FILE_NAME_PREFIX) {
						removeCRSFile(logFilePath(ents[i].Name()), this)
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
				removeActiveFiles(acts[i], this)
				continue
			}

			encrypted, keyId := isEncrypted(f)

			// Check if there is valid TOC/metadata for the stream file
			var metadataFile *os.File // The file that contains the TOC/metadata
			if encrypted {
				m, err := os.Open(requestMetadataActiveFileName(acts[i]))
				if err != nil {
					f.Close()
					logging.Warnf(_MSG_PREFIX+"Failed to find/open past active metadata file %v during loading. Skipping file.",
						acts[i])
					removeActiveFiles(acts[i], this)
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
			}
			f.Close()
			metadataFile.Close()

			if !validMagic {
				logging.Warnf(_MSG_PREFIX+"No valid metadata found for past active file %v during loading. Skipping file.", acts[i])
				removeActiveFiles(acts[i], this)
				continue
			}

			if e := os.Rename(activeFile, requestLogStreamFileName(num)); e != nil {
				logging.Warnf(_MSG_PREFIX+"Failed to rename past active file %v to archive file %v. Skipping file.", acts[i], num)
				removeActiveFiles(acts[i], this)
				continue
			}

			if encrypted {
				if e := os.Rename(requestMetadataActiveFileName(acts[i]), requestMetadataFileName(num)); e != nil {
					logging.Warnf(_MSG_PREFIX+
						"Failed to rename past metadata active file %v to metadata archive file %v (%v). Skipping file.",
						acts[i], num, e)
					removeActiveFiles(acts[i], this)
					continue
				}
			}

			sz += actsz[i]
			ffdcFi := newBaseFileInfo(num, actsz[i]+metadataSize, -1, keyId)
			fi = append(fi, ffdcFi)

		}
		if len(fi) > 0 {
			sort.Slice(fi, func(i int, j int) bool {
				return fi[i].num < fi[j].num
			})
			for i := range fi {
				this.files.PushBack(fi[i])
			}
		}

		this.bootstrapped = true
	} else {
		logging.Errorf(_MSG_PREFIX+"Failed to bootstrap managed file information. Could not open directory %v: %v",
			ffdc.GetPath(), err)
	}
	if atomic.LoadUint64(&this.active) == 0 {
		atomic.StoreUint64(&this.size, sz)
		if e := this.files.Back(); e != nil {
			atomic.StoreUint64(&this.fileNum, e.Value.(*fileInfo).num)
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
func (this *requestLogStream) archive(file *requestStreamFile, lockFilesList bool) {
	archive := newBaseFileInfo(atomic.AddUint64(&this.fileNum, 1), uint64(file.size), int(len(file.offsets)), _STREAM_UNSET_KEY_ID)
	var renameErr error
	renameErr = os.Rename(requestLogStreamActiveFileName(file.index), requestLogStreamFileName(archive.num))
	if renameErr != nil {
		logging.Warnf(_MSG_PREFIX+"Failed to rename active file %v to archive file %v (%v).", file.index, archive.num, renameErr)
		file.Unlock()
		return
	}

	// The file's keyID will either be a keyID or the unencrypted value
	if file.keyID != encryption.UNENCRYPTED_KEY_ID {
		if e := os.Rename(requestMetadataActiveFileName(file.index), requestMetadataFileName(archive.num)); e != nil {
			logging.Warnf(_MSG_PREFIX+"Failed to rename metadata active file %v to metadata archive file %v (%v)", file.index,
				archive.num, e)
		}
	}

	file.size = 0
	file.mtime = time.Time{}

	archive.currKeyID = file.keyID

	// Reset the keyID since there is no physical file associated with the file object anymore
	file.keyID = _STREAM_UNSET_KEY_ID
	file.Unlock()

	if lockFilesList {
		this.filesLock.Lock()
	}
	atomic.AddUint64(&this.size, archive.size)
	this.files.PushBack(archive)
	if lockFilesList {
		this.filesLock.Unlock()
	}
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
						itfi := it.Value.(*fileInfo)
						if _, ok := fi[itfi.num]; !ok {
							archiveSz := itfi.getSize(true)
							released += archiveSz
							if atomic.LoadUint64(&this.size) >= archiveSz {
								atomic.AddUint64(&this.size, ^(archiveSz - 1))
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
						itfi := it.Value.(*fileInfo)

						if _, ok := fi[itfi.num]; ok {

							if !itfi.beginDelete() {
								continue
							}

							archiveSz := itfi.getSize(true)
							released += archiveSz
							if atomic.LoadUint64(&this.size) >= archiveSz {
								atomic.AddUint64(&this.size, ^(archiveSz - 1))
							} else {
								atomic.StoreUint64(&this.size, 0)
							}
							this.files.Remove(it)
							removeArchiveFiles(itfi.num, this)
							itfi.endDelete()
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
		itfi := it.Value.(*fileInfo)

		itfi.lock.RLock()
		currKeyID := itfi.getCurrKeyID(false)
		count := itfi.getCount(false)
		itfi.lock.RUnlock()

		if count == -1 {
			if !itfi.beginLoad() {
				continue
			}

			var f *os.File
			var e error

			// attempt to update the file information

			if currKeyID == encryption.UNENCRYPTED_KEY_ID {
				f, e = os.Open(requestLogStreamFileName(itfi.num))
			} else {
				f, e = os.Open(requestMetadataFileName(itfi.num))
			}

			if e == nil {
				if _, e = f.Seek(-16, os.SEEK_END); e == nil {
					buf := make([]byte, 8)
					_, e = f.Read(buf)
					if e == nil && binary.BigEndian.Uint32(buf) == _STREAM_MAGIC {
						count = int(binary.BigEndian.Uint32(buf[4:]))
						itfi.setCount(true, count)
					}
				}
				f.Close()
			}

			itfi.endLoad()
		}
		if count != -1 {
			res = append(res, itfi.num)
			res = append(res, uint64(count))
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

	// Pinned entries are kept resident in the cache and are not eligible for eviction until they are un-pinned.
	// This is used while an archive file is being transformed so that reads can keep serving data from the cached copy
	// until the operation finishes.
	// During transformation, the transformed data is written to a temporary file, which is then renamed to replace
	// the original file.
	pinned bool
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
			this.cache.MoveToFront(it) // MRU
			break
		}
		ce = nil
	}
	this.Unlock()
	return ce
}

// Adds entry if not already present in cache.
// If the entry already exists in the cache and 'pin' is set to true, will pin the existing entry to the cache
func (this *readCache) add(num uint64, ce *readCacheEntry, pin bool) *readCacheEntry {
	this.Lock()

	if this.cache == nil {
		this.cache = list.New()
	}

	var found bool
	var entry *readCacheEntry
	for it := this.cache.Front(); it != nil; it = it.Next() {
		entry = it.Value.(*readCacheEntry)
		if entry.num == num {
			if pin {
				entry.Lock()
				entry.pinned = true
				entry.Unlock()
			}
			this.cache.MoveToFront(it) // MRU
			found = true
			break
		}
	}

	if !found {
		for this.cache.Len() >= _MAX_CACHE {
			allPinned := true
			for it := this.cache.Back(); it != nil; it = it.Prev() {
				entry = it.Value.(*readCacheEntry)

				entry.Lock()
				if entry.pinned {
					entry.Unlock()
					continue
				}
				entry.Unlock()

				this.cache.Remove(it)
				allPinned = false
				break
			}

			if allPinned {
				break
			}
		}

		ce.num = num
		ce.pinned = pin
		this.cache.PushFront(ce)
		entry = ce
	}

	this.Unlock()
	return entry
}

func (this *readCache) setPinForEntryIfPresent(num uint64, pin bool) bool {
	this.Lock()

	if this.cache == nil {
		this.cache = list.New()
	}

	var present bool
	for it := this.cache.Front(); it != nil; it = it.Next() {
		entry := it.Value.(*readCacheEntry)
		if entry.num == num {
			entry.Lock()
			entry.pinned = pin
			present = true
			entry.Unlock()
			break
		}
	}
	this.Unlock()
	return present
}

func (this *requestLogStream) load(num uint64, pin bool) (*readCacheEntry, error) {
	ce := this.cache.get(num)
	if ce == nil {

		this.filesLock.Lock()
		var canLoad bool
		var encrypted bool
		for it := this.files.Front(); it != nil; it = it.Next() {
			itfi := it.Value.(*fileInfo)

			if itfi.num == num {
				if itfi.beginLoad() {
					defer itfi.endLoad()
					canLoad = true
					encrypted = itfi.getCurrentKeyID(true) != encryption.UNENCRYPTED_KEY_ID
				}
				break
			}
		}

		this.filesLock.Unlock()

		if !canLoad {
			return nil, nil
		}

		streamFile, err := os.Open(requestLogStreamFileName(num))
		if err != nil {
			return nil, err
		}

		defer streamFile.Close()

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
			if this.encryptionProvider == nil {
				return nil, errors.NewEncryptionError(errors.E_NO_ENCRYPTION_MANAGER, nil)
			}

			er, err := encryption.NewCBEFReader(streamFile, func(keyId string) (*encryption.EaRKey, errors.Error) {
				return this.encryptionProvider.GetKey(crsKeyDataType, keyId)
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

		ce = &readCacheEntry{}
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

		ce.pinned = pin
		ce = this.cache.add(num, ce, pin)
	}
	return ce, nil
}

// Encryption at rest related methods

func (this *requestLogStream) InitEncryptionProvider(encProvider encryption.EncryptionProvider) {
	this.Lock()
	this.encryptionProvider = encProvider
	// Load existing files. This is needed to track encrypted files. Irrespective of whether completed request streaming is enabled
	// or not
	requestLog.stream.loadFiles()
	this.Unlock()
}

func (this *requestLogStream) Name() string {
	return "Completed Request Stream"
}

func (this *requestLogStream) GetInUseKeys(dt encryption.KeyDataType) ([]string, error) {
	if dt.TypeName != crsKeyDataType.TypeName {
		return []string{}, nil
	}

	keys := make(map[string]bool, 12)

	// Get the keys used by the active stream files
	this.filesLock.Lock()
	for _, active := range this.streamFiles {
		active.Lock()
		if active.keyID != _STREAM_UNSET_KEY_ID {
			keys[active.keyID] = true
		}
		active.Unlock()
	}
	this.filesLock.Unlock()

	// Get the keys used by the archived files
	this.filesLock.Lock()
	if this.files != nil {
		for archive := this.files.Front(); archive != nil; archive = archive.Next() {
			a := archive.Value.(*fileInfo)
			a.lock.RLock()
			currKeyId := a.getCurrKeyID(false)
			if currKeyId != _STREAM_UNSET_KEY_ID {
				keys[currKeyId] = true
			}

			targetKeyId := a.getTargetKeyID(false)
			if targetKeyId != _STREAM_UNSET_KEY_ID {
				keys[targetKeyId] = true
			}
			a.lock.RUnlock()
		}
	}
	this.filesLock.Unlock()

	// Get the keys being used by orphan files
	this.orphanLock.RLock()
	for _, o := range this.orphanFiles {
		if o.keyID != _STREAM_UNSET_KEY_ID {
			keys[o.keyID] = true
		}
	}
	this.orphanLock.RUnlock()

	keysInUse := make([]string, len(keys))
	i := 0
	for k := range keys {
		keysInUse[i] = k
		i++
	}

	return keysInUse, nil
}

func (this *requestLogStream) DropKey(dt encryption.KeyDataType, keyIdToDrop string) error {
	if dt.TypeName != crsKeyDataType.TypeName {
		return nil
	}

	// Archive all active files using the key to be dropped
	this.Lock()
	for _, active := range this.streamFiles {
		active.Lock()
		if active.keyID == keyIdToDrop {
			active.close()
			if active.size > 0 {
				// archive() will unlock the active file
				this.archive(active, true)
				continue
			}
		}
		active.Unlock()
	}
	this.Unlock()

	// Identify archive files using the key being dropped. And appropriately transform them to use the current active key
	snapshot := make([]*fileInfo, 0, 12)
	this.filesLock.Lock()
	if this.files != nil {
		for archive := this.files.Front(); archive != nil; archive = archive.Next() {
			a := archive.Value.(*fileInfo)
			if a.getCurrKeyID(true) == keyIdToDrop {
				snapshot = append(snapshot, a)
			}
		}
	}
	this.filesLock.Unlock()

	for _, archive := range snapshot {
		if !archive.beginTransform() {
			continue
		}

		postOp := func() {
			archive.endTransform()
			// Unpin cache entry after transformation is done
			this.cache.setPinForEntryIfPresent(archive.num, false)
		}

		// While the on-disk file is being transformed and renamed, readers can continue to serve data from the cached entry
		// for this file. Loading and [inning the cache entry ensures the entry remains available in the cache for reads and
		// protected from cache eviction, until transformation is complete. Once transformation is done, the entry is un-pinned
		// from the cache
		present := this.cache.setPinForEntryIfPresent(archive.num, true)
		if !present {
			_, err := this.load(archive.num, true)
			if err != nil {
				postOp()
				return fmt.Errorf("Error loading archive file %s: %v", requestLogStreamFileName(archive.num), err)
			}
		}

		if this.encryptionProvider == nil {
			postOp()
			return errors.NewEncryptionError(errors.E_NO_ENCRYPTION_MANAGER, nil)
		}

		activeKey, err := this.encryptionProvider.GetActiveKey(crsKeyDataType)
		if err != nil {
			postOp()
			return err
		}

		transformErr := archive.transformForKeyDrop(keyIdToDrop, activeKey, this.encryptionProvider, this)
		postOp()

		if transformErr != nil {
			return fmt.Errorf("Transformation of archive file %v failed with error: %v", requestLogStreamFileName(archive.num),
				transformErr)
		}
	}

	// Check if any files are still using the key to be dropped
	this.filesLock.Lock()
	for _, active := range this.streamFiles {
		active.Lock()
		if active.keyID == keyIdToDrop {
			active.Unlock()
			this.filesLock.Unlock()
			return fmt.Errorf("Key is still in use by active file %v", requestLogStreamActiveFileName(active.index))
		}
		active.Unlock()
	}

	for archive := this.files.Front(); archive != nil; archive = archive.Next() {
		a := archive.Value.(*fileInfo)
		// No need to check target key ID since the transformation procedure was just completed earlier and the target key ID
		// would be unset by said procedure
		if a.getCurrKeyID(true) == keyIdToDrop {
			this.filesLock.Unlock()
			return fmt.Errorf("Key is still in use by archived file %v", requestLogStreamFileName(a.num))
		}
	}
	this.filesLock.Unlock()

	// Perform an orphan file cleanup
	this.cleanupOrphanFiles()

	this.orphanLock.RLock()
	for _, o := range this.orphanFiles {
		if o.keyID == keyIdToDrop {
			this.orphanLock.RUnlock()
			return fmt.Errorf("Key is still in use by orphan file %v", o.path)
		}
	}
	this.orphanLock.RUnlock()

	return nil
}

func (this *requestLogStream) ActiveKeyRotated(dt encryption.KeyDataType) {
	if dt.TypeName != crsKeyDataType.TypeName {
		return
	}

	// When the active key is rotated, the rotated active key should not be used to encrypt new data
	// Close and archive all active stream files so that new writes create new active files that use the new active key
	// This prevents any further data from being written with the old key
	this.Lock()
	for _, active := range this.streamFiles {
		active.Lock()
		if active.keyID != _STREAM_UNSET_KEY_ID {
			active.close()
			if active.size > 0 {
				// archive() will unlock the active file
				this.archive(active, true)
				continue
			}
		}
		active.Unlock()
	}
	this.Unlock()
}

func (this *requestLogStream) cleanupOrphanFiles() {
	this.orphanLock.Lock()
	for i := 0; i < len(this.orphanFiles); i++ {
		o := this.orphanFiles[i]
		err := os.Remove(o.path)
		if err != nil && !os.IsNotExist(err) {
			err = os.Truncate(o.path, 0)
		}

		if err != nil && !os.IsNotExist(err) {
			logging.Warnf(_MSG_PREFIX+"Failed to remove orphan file %v: %v", o.path, err)
			continue
		}

		logging.Infof(_MSG_PREFIX+"Removed orphan file %v", o.path)
		copy(this.orphanFiles[i:], this.orphanFiles[i+1:])
		this.orphanFiles = this.orphanFiles[:len(this.orphanFiles)-1]
	}
	this.orphanLock.Unlock()
}

func (this *requestLogStream) periodicCRSCleanup() {
	ticker := time.NewTicker(_SWEEP_INTERVAL)
	defer func() {
		ticker.Stop()
		// cannot panic and die
		err := recover()
		logging.Debugf(_MSG_PREFIX+"Periodic cleanup routine failed with error: %v. Restarting.", err)
		go this.periodicCRSCleanup()
	}()

	for range ticker.C {
		this.cleanupOrphanFiles()
	}
}

func (fi *fileInfo) encryptUnencryptedFile(activeKey *encryption.EaRKey, encPath string) error {
	origPath := requestLogStreamFileName(fi.num)
	metaPath := requestMetadataFileName(fi.num)

	f, err := os.Open(origPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encrypt all the request data i.e the compressed data upto the TOC
	buf := bufio.NewReaderSize(f, _STREAM_BUF_SIZE)
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return err
	}
	defer gz.Close()
	gz.Multistream(false) // we have trailing data that gzip should not attempt to interpret

	encFile, err := os.Create(encPath)
	if err != nil {
		return err
	}
	defer encFile.Close()

	err = encryption.EncryptFileAsCBEF(gz, encFile, activeKey, encryption.CBEF_ZLIB, _STREAM_BUF_SIZE)
	if err != nil {
		return err
	}

	// Write the TOC to the metadata file
	// Read last 16 bytes for the trailer of the TOC
	_, err = f.Seek(-16, io.SeekEnd)
	if err != nil {
		return err
	}

	trailer := make([]byte, 16)
	if _, err := io.ReadFull(f, trailer); err != nil {
		return err
	}
	count := binary.BigEndian.Uint32(trailer[4:8]) // Number of JSON request entries in the file

	// The TOC consists of the trailer (16 bytes) and the offset table (8 bytes per JSON request entry)
	tocLen := int64(16 + 8*count)

	_, err = f.Seek(-tocLen, io.SeekEnd)
	if err != nil {
		return err
	}

	toc := make([]byte, tocLen)
	_, err = io.ReadFull(f, toc)
	if err != nil {
		return err
	}

	metaFile, err := os.Create(metaPath)
	if err != nil {
		return err
	}
	defer metaFile.Close()

	_, err = metaFile.Write(toc)
	if err != nil {
		return err
	}

	// Do not swap here. Just return success.
	return nil
}

func (fi *fileInfo) reencryptEncryptedFile(encProvider encryption.EncryptionProvider, activeKey *encryption.EaRKey, encPath string) error {

	origPath := requestLogStreamFileName(fi.num)

	src, err := os.Open(origPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(encPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	err = encryption.ReEncryptCBEFFile(src, dst,
		func(keyID string) (*encryption.EaRKey, errors.Error) {
			return encProvider.GetKey(
				crsKeyDataType,
				keyID,
			)
		},
		activeKey,
	)

	if err != nil {
		return err
	}

	return nil
}

func (fi *fileInfo) decryptEncryptedFile(encProvider encryption.EncryptionProvider, decPath string) error {
	origPath := requestLogStreamFileName(fi.num)
	metaPath := requestMetadataFileName(fi.num)

	src, err := os.Open(origPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(decPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	bw := bufio.NewWriterSize(dst, _STREAM_BUF_SIZE)
	gw := gzip.NewWriter(bw)

	err = encryption.DecryptCBEFFile(src, gw, func(keyID string) (*encryption.EaRKey, errors.Error) {
		return encProvider.GetKey(
			crsKeyDataType,
			keyID,
		)
	})
	gw.Close()
	bw.Flush()

	// Write TOC to a new metadata file
	metaFile, err := os.Open(metaPath)
	if err != nil {
		// If the metadata file for does not exist for some reason, just continue with success.
		// There is no need to block key drop in this case
		if go_errors.Is(err, os.ErrNotExist) {
			return nil

		}
		return err
	}
	defer metaFile.Close()

	_, err = io.Copy(dst, metaFile)
	if err != nil {
		return err
	}

	return nil
}

func (this *fileInfo) transformForKeyDrop(keyIdToDrop string, activeKey *encryption.EaRKey,
	encProvider encryption.EncryptionProvider, stream *requestLogStream) error {

	if (activeKey == nil && keyIdToDrop == encryption.UNENCRYPTED_KEY_ID) || (activeKey != nil && keyIdToDrop == activeKey.Id) {
		return fmt.Errorf("Attempt to drop active key")
	}

	targetKeyID := _STREAM_UNSET_KEY_ID

	transformID, _ := util.UUIDV4()
	origPath := requestLogStreamFileName(this.num)
	transformPath := requestLogStreamTransformFileName(this.num, transformID)

	// Step 1: create transformed file
	var transformErr error
	// unencrypted -> encrypted (active key)
	if keyIdToDrop == encryption.UNENCRYPTED_KEY_ID {
		targetKeyID = activeKey.Id
		this.setTargetKeyID(true, targetKeyID)
		transformErr = this.encryptUnencryptedFile(activeKey, transformPath)
	} else if activeKey == nil { // encrypted -> unencrypted
		targetKeyID = encryption.UNENCRYPTED_KEY_ID
		this.setTargetKeyID(true, targetKeyID)
		transformErr = this.decryptEncryptedFile(encProvider, transformPath)
	} else { // encrypted (old key) -> encrypted (active key)
		targetKeyID = activeKey.Id
		this.setTargetKeyID(true, targetKeyID)
		transformErr = this.reencryptEncryptedFile(encProvider, activeKey, transformPath)
	}

	cleanup := func() {
		// Delete the transform file
		removeCRSFile(transformPath, stream)
		this.setTargetKeyID(true, _STREAM_UNSET_KEY_ID)
	}

	if transformErr != nil {
		cleanup()
		return transformErr
	}

	// Step 2: Swap the transformed file with the original file
	// Check if readers are drained before swapping. Do not infinitely wait here, since we do not want to block key drop entirely
	// due to blocked readers
	retries := 5
	for i := 0; i <= retries; i++ {
		if i == retries {
			cleanup()
			return fmt.Errorf(
				"Failed to replace the original file with the transformed file as original file is in use by active readers." +
					"Retries exhausted.")
		}

		if this.getFileReadingCount(true) == 0 {
			break
		}
		time.Sleep(time.Second * 5)
	}

	// Perform swap
	err := os.Rename(transformPath, origPath)
	if err != nil {
		cleanup()
		return err
	}

	// Step 3: transformation-specific post-swap processing
	if activeKey == nil { // encrypted -> unencrypted
		// Delete the metadata file since unencrypted files do not have separate metadata files
		metadataPath := requestMetadataFileName(this.num)
		// It is alright if remove fails at the metadata file has no sensitive info
		os.Truncate(metadataPath, 0)
		os.Remove(metadataPath)
	}

	// Step 4: transformation-specific new size computation
	// It is okay if getting file size stats fails, it is a non fatal error. It is a best effort operation
	if activeKey == nil { // encrypted -> unencrypted
		stat, err := os.Stat(origPath)
		if err == nil {
			this.setSize(true, uint64(stat.Size()))
		}
	} else if keyIdToDrop == encryption.UNENCRYPTED_KEY_ID { // unencrypted -> encrypted (active key)
		sz := 0
		stat, err := os.Stat(origPath)
		if err == nil {
			sz += int(stat.Size())

			metasz, err := os.Stat(requestMetadataFileName(this.num))
			if err == nil {
				sz += int(metasz.Size())
			}

			this.setSize(true, uint64(sz))
		}

	}

	// Step 5: Reset file information
	this.resetAfterTransform(true, targetKeyID)
	return nil
}

// External API

func InitRequestStream() keymgmt.TrackedEncryptor {
	go requestLog.stream.periodicCRSCleanup()
	return &requestLog.stream
}

// returns a _flat_ (for memory efficiency) array of file number & record count pairs
// this is used to produce the system namespace "index" for the streamed history
func RequestsFileStreamFileInfo() []uint64 {
	return requestLog.stream.entryCounts()
}

func RequestsFileStreamRead(fileNum uint64, skip uint64, count uint64, user string, fn func(map[string]interface{}) bool) error {
	ce, err := requestLog.stream.load(fileNum, false)
	if err != nil {
		return err
	}

	if ce == nil {
		return nil
	}

	max := uint64(len(ce.offsets))
	if skip >= max {
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

func isEncrypted(file *os.File) (bool, string) {
	encrypted, keyID := encryption.GetKeyIdFromCBEF(file)
	file.Seek(0, io.SeekStart)
	return encrypted, keyID
}
