//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package keymgmt

import (
	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
)

func NewEncryptionManager(trackedDataTypes []encryption.KeyDataType, trackedEncryptors []TrackedEncryptor) EncryptionManager {
	return &NoopEncryptionManager{}
}

type NoopEncryptionManager struct{}

func (this *NoopEncryptionManager) GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return nil, nil
}

func (this *NoopEncryptionManager) GetKey(dt encryption.KeyDataType, keyID string) (*encryption.EaRKey, errors.Error) {
	return nil, nil
}

func (this *NoopEncryptionManager) PrimeKeys(keyDataTypes []encryption.KeyDataType) errors.Error {
	return nil
}

func (this *NoopEncryptionManager) UpdateKeys(dataType cbauth.KeyDataType, newInfo *cbauth.EncrKeysInfo, prime bool) errors.Error {
	return nil
}

func (this *NoopEncryptionManager) RegisterCbauthEncryptionCallbacks() {
}

func (this *NoopEncryptionManager) GetInUseKeysCallback(dt cbauth.KeyDataType) ([]string, error) {
	return []string{}, nil
}

func (this *NoopEncryptionManager) DropKeysCallback(dt cbauth.KeyDataType, KeyIdsToDrop []string) {
}

func (this *NoopEncryptionManager) SynchronizeKeyFilesCallback(dt cbauth.KeyDataType) error {
	return nil
}

func (this *NoopEncryptionManager) RefreshKeysCallback(dt cbauth.KeyDataType) error {
	return nil
}
