//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encryption

import (
	"fmt"

	"github.com/couchbase/query/errors"
)

var KBKDFDeriveKey = func(masterKey []byte, label []byte, context []byte, derivedKey []byte, digest string) ([]byte, error) {
	return nil, errors.NewEncryptionError(errors.E_ENCRYPTION, fmt.Errorf("Key derivation function not initialized"))
}

var AES256GCMEncrypt = func(key, nonce, ad, plaintext, dst []byte, authTagLen int) ([]byte, error) {
	return nil, errors.NewEncryptionError(errors.E_ENCRYPTION, fmt.Errorf("Encryption function not initialized"))
}

var AES256GCMDecrypt = func(key, nonce, ad, ciphertext, dst []byte, authTagLen int) ([]byte, error) {
	return nil, errors.NewEncryptionError(errors.E_ENCRYPTION, fmt.Errorf("Decryption function not initialized"))
}
