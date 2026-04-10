//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encryption

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/couchbase/query/errors"
)

/*
Buffered reader that reads encrypted data from the underlying reader, de-compresses it (if file is compressed ) and then
decrypts it. Not thread safe.

Read flow:
Reads encrypted data from underlying reader -> decrypts to buffer -> decompresses (if enabled) -> returns plaintext to caller
*/
type CBEFReader struct {
	compression CompressionType

	// Last reader in the read pipeline.
	// Decompression reader wrapping the decryptor if compression is enabled. Otherwise is the decryptor itself.
	outerReader io.Reader
	decryptor   *cbefDecryptor
	closed      bool
}

// getEncryptionKey: Function that returns the encryption key material for a given key ID
func NewCBEFReader(r io.Reader, getEncryptionKey func(keyId string) (*EaRKey, errors.Error)) (*CBEFReader, errors.Error) {
	decryptor, header, err := cbefDecryptorForReader(r, getEncryptionKey, false)
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	compression := CompressionType(header[_CBEF_COMPRESSION_OFFSET])

	cbefr := &CBEFReader{
		decryptor:   decryptor,
		compression: compression,
	}

	// Set up the outer compression reader
	if err := cbefr.setupWrapper(compression); err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	return cbefr, nil
}

// Do not wrap errors returned by Reader methods in a custom type. Callers rely on comparing errors directly (e.g. io.EOF)

func (this *CBEFReader) Read(data []byte) (int, error) {
	if this.closed {
		return 0, fmt.Errorf("Encryption reader is closed")
	}

	return this.outerReader.Read(data)
}

func (this *CBEFReader) Close() error {
	if this.closed {
		return nil
	}

	if or, ok := this.outerReader.(io.Closer); ok {
		if err := or.Close(); err != nil {
			return err
		}
	}

	if err := this.decryptor.Close(); err != nil {
		return err
	}

	this.closed = true
	this.outerReader = nil
	this.decryptor = nil

	return nil
}

func (this *CBEFReader) setupWrapper(compression CompressionType) error {
	var err error
	switch compression {
	case CBEF_ZLIB:
		this.outerReader, err = zlib.NewReader(this.decryptor)
	case CBEF_GZIP:
		this.outerReader, err = gzip.NewReader(this.decryptor)
	case CBEF_NONE:
		this.outerReader = this.decryptor
	default:
		return fmt.Errorf("Unsupported compression type: %d", compression)
	}

	if err != nil {
		return err
	}

	this.compression = compression

	return nil
}

// Buffered reader that reads and then decrypts data from the underlying reader. Not thread safe.
type cbefDecryptor struct {
	closed      bool
	r           io.Reader
	chunkHeader []byte
	nonce       []byte
	AD          []byte
	gcm         cipher.AEAD
	fileOffset  uint64

	// Buffer for decryption operations
	// Used to store the ciphertext read from the underlying reader. The decryption call re-uses this buffer to store the
	// decrypted plaintext. And the plaintext is then read from this buffer
	buffer []byte

	// Current plaintext read position in buffer
	readPos int

	// Length of plaintext in buffer
	plaintextLen int

	chunkOpsOnly bool
}

func newCbefDecryptor(r io.Reader, key []byte, header []byte, chunkOpsOnly bool) (*cbefDecryptor, error) {

	if r == nil {
		return nil, fmt.Errorf("Input reader is nil")
	}

	decryptor := &cbefDecryptor{
		r:            r,
		fileOffset:   _CBEF_HEADER_LENGTH,
		chunkOpsOnly: chunkOpsOnly,
	}

	// Set up AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if aesgcm, err := cipher.NewGCM(block); err != nil {
		return nil, err
	} else {
		decryptor.gcm = aesgcm
	}

	// Setup the AD
	decryptor.AD = make([]byte, _CBEF_AD_LENGTH)
	copy(decryptor.AD[:], header)

	decryptor.nonce = make([]byte, _CBEF_NONCE_LENGTH)
	decryptor.chunkHeader = make([]byte, _CBEF_CHUNK_HEADER_LENGTH)

	return decryptor, nil

}

func (this *cbefDecryptor) Read(data []byte) (int, error) {
	if this.closed {
		return 0, fmt.Errorf("Decryptor is closed")
	}

	if this.chunkOpsOnly {
		return 0, fmt.Errorf("Read is not supported for this decryptor configuration")
	}

	if len(data) == 0 {
		return 0, nil
	}

	read := 0
	for read < len(data) {

		// Check if the buffer has plaintext data to read
		if this.readPos < this.plaintextLen {
			copied := copy(data[read:], this.buffer[this.readPos:this.plaintextLen])
			read += copied
			this.readPos += copied
			continue
		}

		// The buffer has no plaintext data. Decrypt the next encrypted chunk into the buffer
		if err := this.readAndDecrypt(); err != nil {
			return read, err
		}
	}

	return read, nil
}

func (this *cbefDecryptor) ReadChunk() ([]byte, error) {
	if this.closed {
		return nil, fmt.Errorf("Decryptor is closed")
	}

	if !this.chunkOpsOnly {
		return nil, fmt.Errorf("ReadChunk is not supported for this decryptor configuration")
	}

	err := this.readAndDecrypt()
	if err != nil {
		return nil, err
	}

	return this.buffer[:this.plaintextLen], nil
}

// Reads the next chunk from the underlying reader and decrypts it into the buffer.
// Expects the the reader's position to be at the start of the chunk to decrypt
func (this *cbefDecryptor) readAndDecrypt() error {
	if this.closed {
		return fmt.Errorf("Decryptor is closed")
	}

	// Read the chunk header
	if _, err := io.ReadFull(this.r, this.chunkHeader); err != nil {
		return err
	}
	chunkSize := binary.BigEndian.Uint32(this.chunkHeader)

	// Read the nonce
	if _, err := io.ReadFull(this.r, this.nonce); err != nil {
		return err
	}

	// Check that the chunk size contains at least the nonce length before calculating the ciphertext length
	if chunkSize < _CBEF_NONCE_LENGTH {
		return fmt.Errorf("Invalid chunk size specified in chunk header: %d", chunkSize)
	}
	ciphertextLen := int(chunkSize - _CBEF_NONCE_LENGTH)

	// Check if the buffer is large enough to hold the ciphertext. If it is not, allocate a new buffer
	if cap(this.buffer) < ciphertextLen {
		this.buffer = make([]byte, ciphertextLen)
	} else {
		// If the buffer is large enough, resize it appropriately
		// The subsequent Read() on the underlying reader relies on the slice length to know how many bytes to read
		this.buffer = this.buffer[:ciphertextLen]
	}

	// Read the ciphertext
	if _, err := io.ReadFull(this.r, this.buffer); err != nil {
		return err
	}

	// Generate the AD
	binary.BigEndian.PutUint64(this.AD[_CBEF_HEADER_LENGTH:], this.fileOffset)

	// Decrypt
	plaintext, err := this.gcm.Open(this.buffer[:0], this.nonce, this.buffer, this.AD)
	if err != nil {
		return fmt.Errorf("Failed to decrypt ciphertext: %w", err)
	}

	// Update the file offset
	// chunkSize already includes the nonce length, so we only add chunk header + chunkSize
	this.fileOffset += uint64(_CBEF_CHUNK_HEADER_LENGTH + chunkSize)

	// Update buffer variables
	this.buffer = plaintext
	this.readPos = 0
	this.plaintextLen = len(plaintext)

	return nil

}

func (this *cbefDecryptor) Close() error {
	if this.closed {
		return nil
	}

	this.closed = true
	this.AD = nil
	this.nonce = nil
	this.buffer = nil
	this.chunkHeader = nil
	return nil
}

func validateCBEFHeader(header []byte) error {
	if len(header) != _CBEF_HEADER_LENGTH {
		return fmt.Errorf("Invalid header length: %d", len(header))
	}

	if !bytes.Equal(header[:_CBEF_VERSION_OFFSET], _CBEF_MAGIC) {
		return fmt.Errorf("Invalid magic: %v", header[:_CBEF_VERSION_OFFSET])
	}

	if header[_CBEF_VERSION_OFFSET] != _CBEF_VERSION {
		return fmt.Errorf("Invalid version")
	}

	compression := CompressionType(header[_CBEF_COMPRESSION_OFFSET])
	if compression != CBEF_NONE && compression != CBEF_GZIP && compression != CBEF_ZLIB {
		return fmt.Errorf("Unsupported compression type: %s", compression.String())
	}

	keyDerivation := int(header[_CBEF_KEY_DERIVATION_OFFSET])
	if keyDerivation != _CBEF_KEY_DERIVATION {
		return fmt.Errorf("Unsupported key derivation value: %d", keyDerivation)
	}

	// check that the unused field is in fact unused
	for i := _CBEF_UNUSED_FIELD_OFFSET; i < _CBEF_KEY_ID_LENGTH_OFFSET; i++ {
		if header[i] != 0 {
			return fmt.Errorf("Field meant to be unused has invalid value")
		}
	}

	keyIDLength := int(header[_CBEF_KEY_ID_LENGTH_OFFSET])
	if keyIDLength <= 0 || keyIDLength > (_CBEF_RANDOM_SALT_OFFSET-_CBEF_KEY_ID_OFFSET) {
		return fmt.Errorf("Invalid keyID length: %d", keyIDLength)
	}

	return nil
}

/*
This function expects that the reader's position is at the start of the CBEF file header. Returns true if the file header
contains the CBEF magic. This function will advance the reader's position. Callers should rewind the reader to the start of the
reader after calling this function
*/
func IsCBEFReader(r io.Reader) bool {
	if r == nil {
		return false
	}

	magic := make([]byte, _CBEF_VERSION_OFFSET)
	if _, err := io.ReadFull(r, magic); err != nil {
		return false
	}
	return bytes.Equal(magic, _CBEF_MAGIC)
}

/*
If the file header is a valid CBEF header, returns whether the file is in CBEF format and if it is, the keyID from the header
This function expects that the reader's position is at the start of the CBEF file header. And will advance the reader's position.
Callers should rewind the reader to the start of the reader after calling this function
*/
func GetKeyIdFromCBEF(r io.Reader) (bool, string) {
	// Read and validate file header
	header := make([]byte, _CBEF_HEADER_LENGTH)
	if _, err := io.ReadFull(r, header); err != nil {
		return false, ""
	}

	if err := validateCBEFHeader(header); err != nil {
		return false, ""
	}

	keyIDLength := int(header[_CBEF_KEY_ID_LENGTH_OFFSET])
	keyID := string(header[_CBEF_KEY_ID_OFFSET : _CBEF_KEY_ID_OFFSET+keyIDLength])
	return true, keyID
}

/*
CBEFCursor allows reading and seeking within encrypted CBEF files where the file header indicates no compression.
It relies on callers supplying valid start offsets for the encrypted chunks when seeking. This is because
seeking to an offset that is not the start of a chunk will cause subsequent reads/decryption to fail
*/
type CBEFCursor struct {
	r io.ReadSeeker
	*CBEFReader
}

func NewCBEFCursor(r io.ReadSeeker, getKey func(keyId string) (*EaRKey, errors.Error)) (*CBEFCursor, error) {
	cbefReader, err := NewCBEFReader(r, getKey)
	if err != nil {
		return nil, err
	}

	if cbefReader.compression != CBEF_NONE {
		return nil, fmt.Errorf("CBEFCursor does not support compression type: %s", cbefReader.compression.String())
	}

	return &CBEFCursor{
		r:          r,
		CBEFReader: cbefReader,
	}, nil

}

func (this *CBEFCursor) Seek(offset int64, whence int) (int64, error) {
	newOffset, err := this.r.Seek(offset, whence)
	if err != nil {
		return newOffset, err
	}

	// Update the file offset in the decryptor to reflect the new offset in the underlying reader
	this.decryptor.fileOffset = uint64(newOffset)
	this.decryptor.readPos = 0
	this.decryptor.plaintextLen = 0

	return newOffset, nil
}

// Reader that reads and decrypts data from the underlying reader. Not thread safe.
// The reader does not perform any decompression on the data read. If the data read needs to be decompressed before being returned,
// the caller is responsible for decompressing the data after reading the decrypted bytes from the reader.
type cbefDirectReader struct {
	decryptor   *cbefDecryptor
	compression CompressionType
	closed      bool
}

func newCBEFDirectReader(r io.Reader, getEncryptionKey func(keyId string) (*EaRKey, errors.Error)) (*cbefDirectReader,
	errors.Error) {
	decryptor, header, err := cbefDecryptorForReader(r, getEncryptionKey, true)
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	compression := CompressionType(header[_CBEF_COMPRESSION_OFFSET])

	return &cbefDirectReader{
		compression: compression,
		decryptor:   decryptor,
	}, nil
}

func (this *cbefDirectReader) ReadChunk() ([]byte, error) {
	if this.closed {
		return nil, fmt.Errorf("Encryption reader is closed")
	}

	return this.decryptor.ReadChunk()
}

func (this *cbefDirectReader) Close() error {
	if this.closed {
		return nil
	}

	return this.decryptor.Close()
}

// Reads the header from the CBEF reader and creates a CBEF decryptor. Returns the decryptor and file header
func cbefDecryptorForReader(r io.Reader, getEncryptionKey func(keyId string) (*EaRKey, errors.Error), directReader bool) (*cbefDecryptor, []byte,
	error) {
	if r == nil {
		return nil, nil, fmt.Errorf("input reader is nil")
	}

	// Read and validate file header
	header := make([]byte, _CBEF_HEADER_LENGTH)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, nil, fmt.Errorf("Failed to read header of file: %v", err)
	}

	if err := validateCBEFHeader(header); err != nil {
		return nil, nil, err
	}

	keyIDLength := int(header[_CBEF_KEY_ID_LENGTH_OFFSET])
	keyID := string(header[_CBEF_KEY_ID_OFFSET : _CBEF_KEY_ID_OFFSET+keyIDLength])
	salt := header[_CBEF_RANDOM_SALT_OFFSET:]

	key, kerr := getEncryptionKey(keyID)
	if kerr != nil {
		return nil, nil, kerr
	} else if key == nil {
		return nil, nil, fmt.Errorf("Key not found for key ID: %s", keyID)
	}

	// Derive the key
	derivedKey, err := cbefDeriveKey(key.Key, salt, len(key.Key))
	if err != nil {
		return nil, nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	decryptor, err := newCbefDecryptor(r, derivedKey, header, directReader)
	if err != nil {
		return nil, nil, err
	}

	return decryptor, header, nil
}
