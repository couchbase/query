//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"compress/gzip"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/couchbase/gocbcrypto"
	"github.com/couchbase/query/errors"
)

// CBEF = Couchbase Encrypted File Format [https://github.com/couchbase/platform/blob/master/cbcrypto/EncryptedFileFormat.md]

/*
 	CBEF File Header layout:

	| offset | length | description                    |
	+--------+--------+--------------------------------+
	| 0      | 21     | magic: \0Couchbase Encrypted\0 |
	| 21     | 1      | version                        |
	| 22     | 1      | compression                    |
	| 23     | 1      | key derivation                 |
	| 24     | 3      | unused (should be set to 0)    |
	| 27     | 1      | id len                         |
	| 28     | 36     | id bytes                       |
	| 64     | 16     | salt (uuid)                    |

	Total of 80 bytes

	The file will have a file header followed by multiple encrypted chunks.
*/

/*

	The following information is for version 1 of the CBEF format.


	Chunk:
		Layout:
		<4-byte chunk header> ++ <12-byte nonce> ++ <ciphertext>

	Ciphertext: <encrypted text + 16-byte authentication tag>

	Chunk header:
		Length: 4 bytes
		Content: The length of the [nonce ++ [encrypted text ++ authentication tag]] in Big Endian format

	Associated Data (AD):
		Info: Data that is not encrypted, but is authenticated. AD is supplied to the encryption and decryption calls
		to bind contextual information to the ciphertext. And decryption only succeeds if the correct context is provided.
		Length: 88 bytes
		Layout:
			<80-byte file header> ++ <8-byte offset of chunk header>

			Concatenation of the file header and the file offset of the chunk's header. The offset must be appended as a 64 bit
			integer in Big Endian format

	Nonce/IV:
		Info: Must be unique for every encryption call with the same key to ensure security
		Length: 12 bytes

		The following information is how Query will construct the nonce for each encryption call:
			Layout:
				<4-byte fixed field> ++ <8-byte variable field>

				Fixed field:
				Will be set to 0. And will be incremented by 1 once the variable field has reached its maximum value.

				Variable field:
				Will be a 64 bit integer that will be incremented by 1 for each encryption call.


	CBEF Specification for Key Derivation:
		The key used to encrypt/decrypt the file's contents must be derived using OpenSSL's KBKDF (HMAC+SHA2_256)
		in Counter Mode.

		KBKDF(master key, label, context)

		master key = key material associated with the key ID in the file header
		label = "Couchbase Encrypted File"
		context = "Couchbase Encrypted File/<salt in the file header>"

	Encryption Algorithm: AES-256-GCM
*/

/*
  Encryption/ Decryption operations:

  Encryption call: Encrypt(plaintext, key, nonce, AD) -> ciphertext
  Decryption call: Decrypt(ciphertext, key, nonce, AD) -> plaintext

  When using GCM mode of encryption, encrypting a plaintext byte string produces an encrypted text byte string of the same length.
*/

// Offsets for CBEF header fields
const (
	_CBEF_MAGIC_OFFSET          = 0
	_CBEF_VERSION_OFFSET        = 21
	_CBEF_COMPRESSION_OFFSET    = 22
	_CBEF_KEY_DERIVATION_OFFSET = 23
	_CBEF_UNUSED_FIELD_OFFSET   = 24
	_CBEF_KEY_ID_LENGTH_OFFSET  = 27
	_CBEF_KEY_ID_OFFSET         = 28
	_CBEF_RANDOM_SALT_OFFSET    = 64
)

// Constants related to CBEF
const (
	_CBEF_HEADER_LENGTH = 80

	// Chunk length is stored in 4 bytes in thein the chunk header. This length must include the whole chunk i.e
	// <nonce> ++ <encrypted text> ++ <authentication tag>. So the maximum plaintext payload that can be encrypted per chunk is the
	// max 32 bit integer minus the nonce and authentication tag overhead
	_CBEF_MAX_PLAINTEXT_LIMIT = math.MaxUint32 - _CBEF_AUTHENTICATION_TAG_LENGTH - _CBEF_NONCE_LENGTH

	_CBEF_NONCE_LENGTH              = 12
	_CBEF_NONCE_FIXED_FIELD_LENGTH  = 4
	_CBEF_AUTHENTICATION_TAG_LENGTH = 16
	_CBEF_AD_LENGTH                 = _CBEF_HEADER_LENGTH + 8 // 8 bytes for the file offset
	_CBEF_CHUNK_HEADER_LENGTH       = 4
	_CBEF_DEFAULT_PLAINTEXT_LIMIT   = 64 * KiB
	_CBEF_VERSION                   = 1
	_CBEF_KEY_DERIVATION            = 1
	_CBEF_HASH_FUNCTION             = "SHA2-256"
)

var _CBEF_MAGIC = []byte("\x00Couchbase Encrypted\x00")
var _CBEF_KBKDF_CONTEXT_PREFIX = []byte("Couchbase Encrypted File/")
var _CBEF_KBKDF_LABEL = []byte("Couchbase Encrypted File")

type CompressionType int

const (
	CBEF_NONE CompressionType = iota
	_CBEF_SNAPPY
	CBEF_ZLIB
	CBEF_GZIP
	_CBEF_ZSTD
	_CBEF_BZIP2
)

func (this CompressionType) String() string {
	switch this {
	case CBEF_NONE:
		return "no compression"
	case _CBEF_SNAPPY:
		return "snappy"
	case CBEF_ZLIB:
		return "zlib"
	case CBEF_GZIP:
		return "gzip"
	case _CBEF_ZSTD:
		return "zstd"
	case _CBEF_BZIP2:
		return "bzsip2"
	default:
		return "undefined compression algorithm"
	}
}

/*
Buffered writer that compresses (if specified), encrypts and writes data to the underlying writer. Not thread safe.

Write flow:

Caller writes plaintext -> Plaintext is compressed (if enabled) -> (optionally compressed) plaintext is buffered ->
buffered plaintext is encrypted when buffer is flushed/full -> encrypted data written to underlying writer
*/
type CBEFWriter struct {
	compression CompressionType

	// First writer in the write pipeline.
	// Is a compression writer wrapping the encryptor if compression is enabled. Otherwise is the encryptor itself
	outerWriter io.Writer

	encryptor *cbefEncryptor
	closed    bool
}

// Creates a new CBEFWriter with a default buffer size
func NewCBEFWriter(w io.Writer, keyID string, key []byte, compression CompressionType) (*CBEFWriter, errors.Error) {
	return NewCBEFWriterSize(w, keyID, key, compression, _CBEF_DEFAULT_PLAINTEXT_LIMIT)
}

// Creates a new CBEFWriter with the specified buffer size (bytes)
func NewCBEFWriterSize(w io.Writer, keyID string, key []byte, compression CompressionType, bufferSize int) (
	*CBEFWriter, errors.Error) {

	if w == nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, fmt.Errorf("Input writer is nil"))
	}

	if len(keyID) == 0 || len(keyID) > (_CBEF_RANDOM_SALT_OFFSET-_CBEF_KEY_ID_OFFSET) {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, fmt.Errorf("Invalid keyID length: %d", len(keyID)))
	}

	if bufferSize <= 0 {
		bufferSize = _CBEF_DEFAULT_PLAINTEXT_LIMIT
	}

	// Create file header
	header := make([]byte, _CBEF_HEADER_LENGTH)
	copy(header[_CBEF_MAGIC_OFFSET:_CBEF_VERSION_OFFSET], _CBEF_MAGIC)
	header[_CBEF_VERSION_OFFSET] = _CBEF_VERSION
	header[_CBEF_COMPRESSION_OFFSET] = byte(compression)
	header[_CBEF_KEY_DERIVATION_OFFSET] = byte(_CBEF_KEY_DERIVATION)
	// "unused" field in the header is already set to 0 as values in a byte array are 0 by default
	header[_CBEF_KEY_ID_LENGTH_OFFSET] = byte(len(keyID))
	copy(header[_CBEF_KEY_ID_OFFSET:_CBEF_RANDOM_SALT_OFFSET], keyID)

	// Generate a random salt
	_, err := rand.Read(header[_CBEF_RANDOM_SALT_OFFSET:_CBEF_HEADER_LENGTH])
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, err)
	}

	// Dervive a new key using KBKDF
	derivedKey, err := cbefDeriveKey(key, header[_CBEF_RANDOM_SALT_OFFSET:_CBEF_HEADER_LENGTH], len(key))
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, err)
	}

	// Write header
	_, err = w.Write(header)
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, err)
	}

	encryptor, err := newCbefEncryptor(w, derivedKey, header, bufferSize)
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, err)
	}

	cbefw := &CBEFWriter{
		encryptor:   encryptor,
		compression: compression,
	}

	// Setup outer compression writer
	err = cbefw.setupOuterWriter(compression)
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_WRITER_CREATE, err)
	}

	return cbefw, nil
}

// Do not wrap errors returned by Writer methods in a custom type. Callers rely on comparing errors directly (e.g. io.EOF)

func (this *CBEFWriter) Write(data []byte) (int, error) {
	if this.closed {
		return 0, fmt.Errorf("Encryption writer is closed")
	}

	return this.outerWriter.Write(data)
}

// Flush encrypts and writes any buffered data to the uderlying writer
func (this *CBEFWriter) Flush() error {
	if this.closed {
		return fmt.Errorf("Encryption writer is closed")
	}

	// If there is an outer writer - flush it
	if this.outerWriter != this.encryptor {
		if wf, ok := this.outerWriter.(interface{ Flush() error }); ok {
			if err := wf.Flush(); err != nil {
				return fmt.Errorf("Failed to flush encryption writer: %v", err)
			}
		}
	}

	err := this.encryptor.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush encryption writer: %v", err)
	}

	return nil

}

// Closes the writer after flushing
func (this *CBEFWriter) Close() error {
	if this.closed {
		return nil
	}

	// If there is an outer writer - close it
	if this.outerWriter != this.encryptor {
		if err := this.Flush(); err != nil {
			return err
		}

		if ow, ok := this.outerWriter.(io.Closer); ok {
			if err := ow.Close(); err != nil {
				return err
			}
		}
	}

	if err := this.encryptor.Close(); err != nil {
		return err
	}

	this.closed = true
	this.outerWriter = nil
	this.encryptor = nil

	return nil
}

func (this *CBEFWriter) setupOuterWriter(compression CompressionType) error {
	switch compression {
	case CBEF_ZLIB:
		this.outerWriter = zlib.NewWriter(this.encryptor)
	case CBEF_GZIP:
		this.outerWriter = gzip.NewWriter(this.encryptor)
	case CBEF_NONE:
		this.outerWriter = this.encryptor
	default:
		return fmt.Errorf("Unsupported compression type: %s", compression.String())
	}

	this.compression = compression

	return nil
}

// Buffered writer that encrypts and writes data to the underlying writer. Not thread safe.
type cbefEncryptor struct {
	closed      bool
	w           io.Writer
	header      []byte
	AD          []byte
	chunkHeader []byte
	gcm         cipher.AEAD
	fileOffset  uint64

	nonce []byte
	// Fixed field of nonce
	nonceFixedFieldCounter uint32 // Fixed field of nonce
	// Variable field of nonce
	nonceCounter uint64

	// Buffer for encryption operations
	// Stores plaintext up to plainTextLimit. When the accumulated plaintext is encrypted, the encryption call re-uses this buffer
	// to store the ciphertext
	buffer []byte

	// Maximum amount of plaintext that can be stored in the buffer before encryption is performed
	plaintextLimit int

	// End of accumulated plaintext in buffer and start position for next write
	writePos int
}

func newCbefEncryptor(w io.Writer, key []byte, header []byte, bufferSize int) (*cbefEncryptor, error) {
	if w == nil {
		return nil, fmt.Errorf("Input writer is nil")
	}

	if bufferSize <= 0 || bufferSize > _CBEF_MAX_PLAINTEXT_LIMIT {
		return nil, fmt.Errorf("Invalid buffer size: %d", bufferSize)
	}

	encryptor := &cbefEncryptor{
		w:          w,
		header:     header,
		fileOffset: _CBEF_HEADER_LENGTH,
	}

	// Set up AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if aesgcm, err := cipher.NewGCM(block); err != nil {
		return nil, err
	} else {
		encryptor.gcm = aesgcm
	}

	// Setup the AD
	encryptor.AD = make([]byte, _CBEF_AD_LENGTH)
	copy(encryptor.AD[:], header)

	// Setup the nonce
	encryptor.nonce = make([]byte, _CBEF_NONCE_LENGTH)

	// Setup the buffer
	encryptor.buffer = make([]byte, bufferSize+encryptor.gcm.Overhead())
	encryptor.plaintextLimit = bufferSize

	// Setup the chunk header
	encryptor.chunkHeader = make([]byte, _CBEF_CHUNK_HEADER_LENGTH)

	return encryptor, nil
}

func (this *cbefEncryptor) Write(data []byte) (int, error) {
	if this.closed {
		return 0, fmt.Errorf("Encryptor is closed")
	}

	if len(data) == 0 {
		return 0, nil
	}

	written := 0
	dataPos := 0

	for dataPos < len(data) {

		// Number of free bytes available in the buffer to write to
		available := this.plaintextLimit - this.writePos

		// Number of bytes to copy from input data to buffer
		copied := len(data) - dataPos
		if copied > available {
			copied = available
		}

		// Copy the data to the buffer
		if copied > 0 {
			// Can safely assume that the buffer is large enough to hold the data to copy
			copy(this.buffer[this.writePos:], data[dataPos:dataPos+copied])
			this.writePos += copied
			dataPos += copied
			written += copied
		}

		// Buffer is full, encrypt and write it
		if this.writePos == this.plaintextLimit {
			if err := this.encryptAndWrite(); err != nil {
				return written, err
			}
		} else if this.writePos > this.plaintextLimit {
			// This should ideally not happen
			return written, fmt.Errorf("Unexpected internal buffer overflow")
		}
	}

	return written, nil
}

// Flush encrypts and writes any buffered data to the uderlying writer
func (this *cbefEncryptor) Flush() error {
	if this.closed {
		return nil
	}

	if this.writePos > 0 {
		if err := this.encryptAndWrite(); err != nil {
			return err
		}
	}

	return nil
}

// Closes the encryptor after flushing
func (this *cbefEncryptor) Close() error {
	if this.closed {
		return nil
	}

	err := this.Flush()

	this.closed = true
	this.header = nil
	this.AD = nil
	this.nonce = nil
	this.buffer = nil
	this.chunkHeader = nil

	return err
}

// Encrypts data in the buffer and writes it to the underlying writer
func (this *cbefEncryptor) encryptAndWrite() error {

	if this.closed {
		return fmt.Errorf("Encryptor is closed")
	}

	if this.writePos == 0 {
		return nil
	}

	// If the variable field of the nonce has reached its max value, increment the fixed field counter.
	// This is to ensure nonce uniqueness
	if this.nonceCounter == math.MaxUint64 {
		if this.nonceFixedFieldCounter == math.MaxUint32 {
			return fmt.Errorf("Nonce has overflowed.")
		}

		this.nonceFixedFieldCounter++
		binary.BigEndian.PutUint32(this.nonce[:_CBEF_NONCE_FIXED_FIELD_LENGTH], this.nonceFixedFieldCounter)
		this.nonceCounter = 0
	}

	binary.BigEndian.PutUint64(this.nonce[_CBEF_NONCE_FIXED_FIELD_LENGTH:], this.nonceCounter)

	// Increment the nonce counter for the next encryption call
	this.nonceCounter++

	// Create the AD for this encryption call
	binary.BigEndian.PutUint64(this.AD[_CBEF_HEADER_LENGTH:], this.fileOffset)

	// Encrypt the data
	ciphertext := this.gcm.Seal(this.buffer[:0], this.nonce, this.buffer[:this.writePos], this.AD)

	// Reset buffer variables
	this.writePos = 0

	// Create and write the chunk header
	binary.BigEndian.PutUint32(this.chunkHeader, uint32(_CBEF_NONCE_LENGTH+len(ciphertext)))

	// Write the chunk
	written := 0
	if n, err := this.w.Write(this.chunkHeader); err != nil {
		this.fileOffset += uint64(n)
		return err
	} else {
		written += n
	}

	if n, err := this.w.Write(this.nonce); err != nil {
		this.fileOffset += uint64(written + n)
		return err
	} else {
		written += n
	}

	if n, err := this.w.Write(ciphertext); err != nil {
		this.fileOffset += uint64(written + n)
		return err
	} else {
		written += n
	}

	// Update the file offset
	this.fileOffset += uint64(written)

	return nil
}

// Derives a new key using KBKDF in accordance with the CBEF specification for key derivation
func cbefDeriveKey(key []byte, salt []byte, derivedKeyLen int) ([]byte, error) {
	derivedKey := make([]byte, derivedKeyLen)

	kdfCtx := make([]byte, len(_CBEF_KBKDF_CONTEXT_PREFIX)+len(salt))
	copy(kdfCtx[:len(_CBEF_KBKDF_CONTEXT_PREFIX)], _CBEF_KBKDF_CONTEXT_PREFIX)
	copy(kdfCtx[len(_CBEF_KBKDF_CONTEXT_PREFIX):], salt)

	derivedKey, err := gocbcrypto.OpenSSLKBKDFDeriveKey(key, _CBEF_KBKDF_LABEL, kdfCtx, derivedKey, _CBEF_HASH_FUNCTION, "")
	if err != nil {
		return nil, fmt.Errorf("Failed to derive key using KBKDF: %v", err)
	}

	return derivedKey, nil
}
