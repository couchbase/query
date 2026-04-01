//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package keymgmt

import (
	"fmt"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

type NodeEncryptionManager struct {
	keyStore *nodeKeyStore
}

func NewNodeEncryptionManager() *NodeEncryptionManager {
	return &NodeEncryptionManager{
		keyStore: newNodeKeyStore(),
	}
}

// Performs initialization of the manager with info about the provided key datatypes
// Typically invoked during service startup to initialize the manager with required key data.
func (this *NodeEncryptionManager) PrimeKeys(keyDataTypes []encryption.KeyDataType) errors.Error {
	logging.Infof("Priming encryption-at-rest manager with keys for all available key data types")
	this.keyStore.PrimeKeys(keyDataTypes)
	logging.Infof("Finished priming operation of encryption-at-rest manager with keys for all available key data types")
	return nil
}

func (this *NodeEncryptionManager) UpdateKeys(dataType cbauth.KeyDataType, newInfo *cbauth.EncrKeysInfo, prime bool) errors.Error {
	return this.keyStore.UpdateKeys(dataType, newInfo, prime)
}

func (this *NodeEncryptionManager) GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return this.keyStore.GetActiveKey(dt)
}

func (this *NodeEncryptionManager) RegisterCbauthEncryptionCallbacks() {
	cbauth.RegisterEncryptionKeysCallbacks(this.RefreshKeysCallback, this.GetInUseKeysCallback, this.DropKeysCallback,
		this.SynchronizeKeyFilesCallback)
}

func (this *NodeEncryptionManager) RefreshKeysCallback(dt cbauth.KeyDataType) error {
	// Key info will always be present in cbauth when RefreshKeysCallback is called.
	// Thus call cbauth.GetEncryptionKeys() instead of GetEncryptionKeysBlocking().
	newKeys, cbErr := cbauth.GetEncryptionKeys(dt)

	// In RefreshKeysCallback, any error returned by GetEncryptionKeys() is a hard error including cbauth.ErrKeysNotAvailable.
	// This is because key info should always be available when RefreshKeysCallback is called by cbauth.
	if cbErr != nil {
		logging.Errorf(
			"Error refreshing encryption-at-rest configuration for key data type %s. Failed to fetch configuration from cbauth: %v",
			cbauthTypeToDataType(dt).String(), cbErr)
		return cbErr
	}

	err := this.keyStore.UpdateKeys(dt, newKeys, false)
	if err != nil && err.Code() != errors.E_INVALID_ENCRYPTION_KEY_DATATYPE {
		logging.Errorf(
			"Error refreshing encryption-at-rest configuration for key data type %s. Failed to update local encryption manager: %v",
			cbauthTypeToDataType(dt).String(), err)
	}

	return nil
}

func (this *NodeEncryptionManager) GetInUseKeysCallback(dt cbauth.KeyDataType) ([]string, error) {
	// EAR TODO
	// If no keys are in use, cbauth expects an empty slice to be returned, not a nil value
	return []string{}, nil
}

func (this *NodeEncryptionManager) DropKeysCallback(dt cbauth.KeyDataType, KeyIdsToDrop []string) {
	// EAR TODO
}

func (this *NodeEncryptionManager) SynchronizeKeyFilesCallback(dt cbauth.KeyDataType) error {
	// Query has no requirement for this as of now
	return nil
}

func validateKeyDataType(dt cbauth.KeyDataType) (encryption.KeyDataType, errors.Error) {
	kdt := encryption.KeyDataType{
		TypeName:   dt.TypeName,
		BucketUUID: dt.BucketUUID,
	}

	if dt.BucketUUID != "" && dt.TypeName != encryption.BUCKET_KEY_DATATYPE {
		return encryption.KeyDataType{}, errors.NewEncryptionError(errors.E_INVALID_ENCRYPTION_KEY_DATATYPE,
			fmt.Errorf("bucketUUID is only valid when typeName is service_bucket"), kdt.String())
	}

	if dt.TypeName == encryption.BUCKET_KEY_DATATYPE && dt.BucketUUID == "" {
		return encryption.KeyDataType{}, errors.NewEncryptionError(errors.E_INVALID_ENCRYPTION_KEY_DATATYPE,
			fmt.Errorf("bucketUUID is required when typeName is service_bucket"), kdt.String())
	}

	switch dt.TypeName {
	case encryption.BUCKET_KEY_DATATYPE:
		if dt.BucketUUID == "" {
			return encryption.KeyDataType{}, errors.NewEncryptionError(errors.E_INVALID_ENCRYPTION_KEY_DATATYPE,
				fmt.Errorf("bucketUUID is required when typeName is service_bucket"), kdt.String())
		}
	case encryption.LOG_KEY_DATATYPE, encryption.OTHER_KEY_DATATYPE:
	default:
		return encryption.KeyDataType{}, errors.NewEncryptionError(errors.E_INVALID_ENCRYPTION_KEY_DATATYPE,
			fmt.Errorf("unsupported key data type"), kdt.String())
	}

	return kdt, nil
}

func dataTypeToCbauthType(dt encryption.KeyDataType) cbauth.KeyDataType {
	return cbauth.KeyDataType{
		TypeName:   dt.TypeName,
		BucketUUID: dt.BucketUUID,
	}
}

func cbauthTypeToDataType(dt cbauth.KeyDataType) encryption.KeyDataType {
	return encryption.KeyDataType{
		TypeName:   dt.TypeName,
		BucketUUID: dt.BucketUUID,
	}
}
