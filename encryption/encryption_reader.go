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
func NewCBEFReader(r io.Reader, getEncryptionKey func(keyId string) []byte) (*CBEFReader, errors.Error) {
	if r == nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, fmt.Errorf("input reader is nil"))
	}

	// Read and validate file header
	header := make([]byte, _CBEF_HEADER_LENGTH)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil,
			errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, fmt.Errorf("Failed to read header of file: %v", err))
	}

	if err := validateCBEFHeader(header); err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	keyIDLength := int(header[_CBEF_KEY_ID_LENGTH_OFFSET])
	keyID := string(header[_CBEF_KEY_ID_OFFSET : _CBEF_KEY_ID_OFFSET+keyIDLength])
	salt := header[_CBEF_RANDOM_SALT_OFFSET:]

	key := getEncryptionKey(keyID)
	if key == nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, fmt.Errorf("Key not found for key ID: %s", keyID))
	}

	// Derive the key
	derivedKey, err := cbefDeriveKey(key, salt, len(key))
	if err != nil {
		return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_READER_CREATE, err)
	}

	decryptor, err := newCbefDecryptor(r, derivedKey, header)
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
	header      []byte
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
}

func newCbefDecryptor(r io.Reader, key []byte, header []byte) (*cbefDecryptor, error) {

	if r == nil {
		return nil, fmt.Errorf("Input reader is nil")
	}

	decryptor := &cbefDecryptor{
		r:          r,
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

	this.header = nil
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
