//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package openssl

import (
	"github.com/couchbase/gocbcrypto"
	"github.com/couchbase/query/encryption"
)

func kBKDFDeriveKey(masterKey []byte, label []byte, context []byte, derivedKey []byte, digest string) ([]byte, error) {
	return gocbcrypto.OpenSSLKBKDFDeriveKey(masterKey, label, context, derivedKey, digest, "")
}

func Init() {
	encryption.KBKDFDeriveKey = kBKDFDeriveKey
}
