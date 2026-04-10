//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package keymgmt

import (
	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
)

type EncryptionManager interface {
	GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error)
	GetKey(dt encryption.KeyDataType, keyID string) (*encryption.EaRKey, errors.Error)
	PrimeKeys(keyDataTypes []encryption.KeyDataType) errors.Error
	UpdateKeys(dataType cbauth.KeyDataType, newInfo *cbauth.EncrKeysInfo, prime bool) errors.Error
	RegisterCbauthEncryptionCallbacks()
	GetInUseKeysCallback(dt cbauth.KeyDataType) ([]string, error)
	DropKeysCallback(dt cbauth.KeyDataType, KeyIdsToDrop []string)
	SynchronizeKeyFilesCallback(dt cbauth.KeyDataType) error
	RefreshKeysCallback(dt cbauth.KeyDataType) error
}

type TrackedEncryptor interface {
	GetInUseKeys(dt encryption.KeyDataType) ([]string, errors.Error)
	DropKey(dt encryption.KeyDataType, keyId string) errors.Error
	Name() string
}
