//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encryption_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"testing"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/encryption/openssl"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
)

func init() {
	// Key derivation (KBKDF) is provided by the openssl package. It must be initialized before any
	// CBEFWriter/CBEFReader/CBEFCursor is created, since key derivation is used to derive the per-file key.
	openssl.Init()
}

func testKey(t *testing.T) *encryption.EaRKey {
	t.Helper()

	keyMaterial := make([]byte, 32)
	if _, err := rand.Read(keyMaterial); err != nil {
		t.Fatalf("Failed to generate key material: %v", err)
	}

	uuid, err := util.UUIDV4()
	if err != nil {
		t.Fatalf("Failed to generate key ID: %v", err)
	}

	return &encryption.EaRKey{
		Id:     uuid,
		Cipher: encryption.AES_256_GCM_CIPHER,
		Key:    keyMaterial,
	}
}

func keyGetter(key *encryption.EaRKey) func(string) (*encryption.EaRKey, errors.Error) {
	return func(keyId string) (*encryption.EaRKey, errors.Error) {
		if keyId != key.Id {
			return nil, errors.NewEncryptionError(errors.E_ENCRYPTION, fmt.Errorf("unknown key ID: %s", keyId))
		}
		return key, nil
	}
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()

	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	return data
}

func write(t *testing.T, w io.Writer, data []byte) {
	t.Helper()

	_, err := w.Write(data)
	if err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}
}

// Reads all data from r, issuing Read() calls with varying input buffer sizes each time
func readVaryingSizes(t *testing.T, r io.Reader) []byte {
	t.Helper()

	readSizes := []int{1, 3, 17, 100, 512, 1023, 1024, 1025, 4096, 5000}

	var result []byte
	for i := 0; ; i++ {
		buf := make([]byte, readSizes[i%len(readSizes)])
		n, err := r.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Failed to read decrypted data: %v", err)
		}
	}
	return result
}

func TestCBEFWriterReaderRoundTrip(t *testing.T) {
	var tests = []struct {
		name        string
		compression encryption.CompressionType
	}{
		{"no compression", encryption.CBEF_NONE},
		{"zlib compression", encryption.CBEF_ZLIB},
		{"gzip compression", encryption.CBEF_GZIP},
	}

	const bufferSize = 1024

	// Three writes exercising the three ways a Write() call can relate to the buffer size: more than a full
	// buffer, less than a full buffer, and exactly a full buffer.
	moreThanBuffer := bufferSize + 500
	lessThanBuffer := bufferSize - 300
	equalToBuffer := bufferSize

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := testKey(t)
			data := randomBytes(t, moreThanBuffer+lessThanBuffer+equalToBuffer)

			piece1 := data[:moreThanBuffer]
			piece2 := data[moreThanBuffer : moreThanBuffer+lessThanBuffer]
			piece3 := data[moreThanBuffer+lessThanBuffer:]

			var encrypted bytes.Buffer

			writer, err := encryption.NewCBEFWriterSize(&encrypted, key, test.compression, bufferSize)
			if err != nil {
				t.Fatalf("Failed to create CBEFWriter: %v", err)
			}

			write(t, writer, piece1)
			write(t, writer, piece2)
			write(t, writer, piece3)

			if err := writer.Close(); err != nil {
				t.Fatalf("Failed to close CBEFWriter: %v", err)
			}

			reader, err := encryption.NewCBEFReader(bytes.NewReader(encrypted.Bytes()), keyGetter(key))
			if err != nil {
				t.Fatalf("Failed to create CBEFReader: %v", err)
			}
			defer reader.Close()

			decrypted := readVaryingSizes(t, reader)

			if !bytes.Equal(decrypted, data) {
				t.Errorf("Decrypted data does not match original data (original len %d, decrypted len %d)",
					len(data), len(decrypted))
			}
		})
	}
}

func TestCBEFCursor(t *testing.T) {
	key := testKey(t)

	const bufferSize = 1024

	// Each entry is written and flushed as its own chunk so that the recorded offsets align with chunk
	// boundaries, which CBEFCursor requires when seeking.
	chunks := [][]byte{
		randomBytes(t, 100),
		randomBytes(t, bufferSize),
		randomBytes(t, 1),
		randomBytes(t, bufferSize-1),
		randomBytes(t, 777),
	}

	var encrypted bytes.Buffer

	writer, err := encryption.NewCBEFWriterSize(&encrypted, key, encryption.CBEF_NONE, bufferSize)
	if err != nil {
		t.Fatalf("Failed to create CBEFWriter: %v", err)
	}

	offsets := make([]int64, len(chunks))
	for i, chunk := range chunks {
		offsets[i] = int64(encrypted.Len())

		if _, err := writer.Write(chunk); err != nil {
			t.Fatalf("Failed to write chunk %d: %v", i, err)
		}
		if err := writer.Flush(); err != nil {
			t.Fatalf("Failed to flush chunk %d: %v", i, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close CBEFWriter: %v", err)
	}

	cursor, cursorErr := encryption.NewCBEFCursor(bytes.NewReader(encrypted.Bytes()), keyGetter(key))
	if cursorErr != nil {
		t.Fatalf("Failed to create CBEFCursor: %v", cursorErr)
	}
	defer cursor.Close()

	// Seek to and read each chunk out of order to verify random access works correctly.
	order := []int{2, 0, 4, 1, 3}
	for _, i := range order {
		if _, err := cursor.Seek(offsets[i], io.SeekStart); err != nil {
			t.Fatalf("Failed to seek to chunk %d at offset %d: %v", i, offsets[i], err)
		}

		chunk := make([]byte, len(chunks[i]))
		if _, err := io.ReadFull(cursor, chunk); err != nil {
			t.Fatalf("Failed to read chunk %d: %v", i, err)
		}

		if !bytes.Equal(chunk, chunks[i]) {
			t.Errorf("Chunk %d does not match original data after seeking to offset %d", i, offsets[i])
		}
	}
}
