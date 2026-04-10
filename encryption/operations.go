//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encryption

import (
	"io"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

const (
	_REENCRYPTION_READ_BUFFER_SIZE = 4 * util.KiB
)

// Decrypts the source CBEF file's content and writes the decrypted content to the destination writer
// Caller is responsible for closing source reader and destination writer
func DecryptCBEFFile(src io.Reader, dst io.Writer, getSrcEncryptionKey func(keyId string) (*EaRKey, errors.Error)) error {
	// Use NewCBEFReader instead of cbefDirectReader to read the source file
	// NewCBEFReader returns decrypted and decompressed data (when applicable)
	// while cbefDirectReader returns decrypted chunk payloads as-is, which may still be compressed
	// By using NewCBEFReader, it ensures callers can stream true plaintext to the destination writer and apply any custom write
	// handling
	er, err := NewCBEFReader(src, getSrcEncryptionKey)
	if err != nil {
		return err
	}

	buf := make([]byte, _REENCRYPTION_READ_BUFFER_SIZE)
	for {
		n, rerr := er.Read(buf)
		if rerr != nil && rerr != io.EOF {
			return rerr
		}

		_, werr := dst.Write(buf[:n])

		if werr != nil {
			return werr
		}

		if rerr == io.EOF {
			break
		}
	}

	er.Close()
	return nil
}

// Re-encrypts the source CBEF file's content and writes the re-encrypted content to the destination writer
// Re-encrypts the source file chunk-by-chunk while preserving the original chunk boundaries and compression configuration.
// Each chunk is decrypted from the source, then re-encrypted with the new key and written to the destination
// This function does not close the source reader or destination writer.
func ReEncryptCBEFFile(src io.Reader, dst io.Writer, getSrcEncryptionKey func(keyId string) (*EaRKey, errors.Error), dstKey *EaRKey) error {
	er, err := newCBEFDirectReader(src, getSrcEncryptionKey)
	if err != nil {
		return err
	}

	ew, err := newCBEFDirectWriter(dst, dstKey, er.compression)
	if err != nil {
		return err
	}

	// Read each chunk from the source reader, re-encrypt it and write to the destination writer
	// If the data is compressed, we do not need to decompress and recompress it. Since re-encryption preserves the compression type,
	// the compressed content remains the same. Hence, simply encrypt the compressed content with the new key.
	for {
		chunk, rerr := er.ReadChunk()
		if rerr != nil && rerr != io.EOF {
			return rerr
		}

		werr := ew.WriteChunk(chunk)
		if werr != nil {
			return werr
		}

		if rerr == io.EOF {
			break
		}
	}

	er.Close()
	ew.Close()
	return nil
}

// Encrypts the source file's content and writes the encrypted content to the destination writer in CBEF format
// Specify the compression type to compress the content before encryption
// Caller is responsible for closing source reader and destination writer
func EncryptFileAsCBEF(src io.Reader, dst io.Writer, dstKey *EaRKey, compression CompressionType, encryptionBufferSize int) error {
	ew, err := NewCBEFWriterSize(dst, dstKey, compression, encryptionBufferSize)
	if err != nil {
		return err
	}

	buf := make([]byte, _REENCRYPTION_READ_BUFFER_SIZE)
	for {
		n, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF {
			return rerr
		}

		_, werr := ew.Write(buf[:n])

		if werr != nil {
			return werr
		}

		if rerr == io.EOF {
			break
		}
	}

	ew.Close()
	return nil
}
