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
	"sync"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

type NodeEncryptionManager struct {
	keyStore         *nodeKeyStore
	trackedDatatypes map[encryption.KeyDataType]bool
	encryptors       []TrackedEncryptor

	dropLock   sync.Mutex
	keysToDrop []basicKeyInfo
	cond       sync.Cond
}

type basicKeyInfo struct {
	keyID    string
	dataType encryption.KeyDataType
}

func NewNodeEncryptionManager(trackedDataTypes []encryption.KeyDataType,
	trackedEncryptors []TrackedEncryptor) *NodeEncryptionManager {
	mgr := &NodeEncryptionManager{
		keyStore:         newNodeKeyStore(),
		trackedDatatypes: make(map[encryption.KeyDataType]bool, len(trackedDataTypes)),
		encryptors:       trackedEncryptors,
	}

	for _, dt := range trackedDataTypes {
		mgr.trackedDatatypes[dt] = true
	}

	if len(mgr.encryptors) > 0 {
		mgr.initDropKeysWorker()
	}

	logging.Infof("EAR: Initialized node encryption-at-rest manager")

	return mgr
}

func (this *NodeEncryptionManager) initDropKeysWorker() {
	this.dropLock.Lock()
	defer this.dropLock.Unlock()
	this.keysToDrop = make([]basicKeyInfo, 0, _MAX_KEY_DATATYPES)
	this.cond.L = &this.dropLock
	// Start the worker under lock to ensure it has been started before any interaction with encryption manager
	go this.dropKeysWorker()
}

// Performs initialization of the manager with info about the provided key datatypes
// Typically invoked during service startup to initialize the manager with required key data.
func (this *NodeEncryptionManager) PrimeKeys(keyDataTypes []encryption.KeyDataType) errors.Error {
	logging.Infof("EAR: Priming manager with keys for all available key data types")
	this.keyStore.PrimeKeys(keyDataTypes)
	logging.Infof("EAR: Finished priming operation of manager with keys for all available key data types")
	return nil
}

func (this *NodeEncryptionManager) UpdateKeys(dataType cbauth.KeyDataType, newInfo *cbauth.EncrKeysInfo, prime bool) errors.Error {
	return this.keyStore.UpdateKeys(dataType, newInfo, prime)
}

func (this *NodeEncryptionManager) GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return this.keyStore.GetActiveKey(dt)
}

func (this *NodeEncryptionManager) GetKey(dt encryption.KeyDataType, keyID string) (*encryption.EaRKey, errors.Error) {
	return this.keyStore.GetKey(dt, keyID)
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
			"EAR: [data_type=%s] Error refreshing encryption configuration. Failed to fetch configuration from cbauth: %v",
			cbauthTypeToDataType(dt).String(), cbErr)
		return cbErr
	}

	err := this.keyStore.UpdateKeys(dt, newKeys, false)
	if err != nil && err.Code() != errors.E_INVALID_ENCRYPTION_KEY_DATATYPE {
		logging.Errorf("EAR: [data_type=%s] Error refreshing encryption configuration. Failed to update local key store: %v",
			cbauthTypeToDataType(dt).String(), err)
	}

	return nil
}

// Returns IDs of keys being used to encrypt data on disk
// Must returns "" if there is any un-encrypted data on disk
// Must return the active key ID if encryption at rest is enabled for the data type
func (this *NodeEncryptionManager) GetInUseKeysCallback(dt cbauth.KeyDataType) ([]string, error) {
	qdt := cbauthTypeToDataType(dt)
	_, ok := this.trackedDatatypes[qdt]
	if !ok {
		// if no keys are tracked for the provided data type, cbauth requires this method to return empty list instead of nil
		return []string{}, nil
	}

	var keysInUse map[string]bool
	for _, encryptor := range this.encryptors {
		encryptorKeys, err := encryptor.GetInUseKeys(qdt)
		if err != nil {
			logging.Errorf("EAR: [data_type=%s] Error getting in-use keys from encryptor '%s': %v", qdt.String(), encryptor.Name(),
				err)
			return []string{}, err
		}

		if len(encryptorKeys) == 0 {
			continue
		}

		if keysInUse == nil {
			keysInUse = make(map[string]bool, len(encryptorKeys)+1)
		}

		for i := 0; i < len(encryptorKeys); i++ {
			keysInUse[encryptorKeys[i]] = true
		}
	}

	// cbauth requires that the method always return the active key ID only if encryption at rest is enabled for the data type
	activeKey, err := this.keyStore.GetActiveKey(qdt)
	if err != nil {
		return []string{}, err
	}

	if activeKey != nil {
		if keysInUse == nil {
			return []string{activeKey.Id}, nil
		}

		keysInUse[activeKey.Id] = true
	}

	list := make([]string, len(keysInUse))
	i := 0
	for keyId := range keysInUse {
		list[i] = keyId
		i++
	}

	return list, nil

}

// Queues the keys to be dropped in the encryption manager and returns immediately.
// Any required actions (e.g., file re-encryption for key drop) are performed asynchronously, as cbauth requires the callback
// to return without delay.
func (this *NodeEncryptionManager) DropKeysCallback(dt cbauth.KeyDataType, KeyIdsToDrop []string) {
	qdt := cbauthTypeToDataType(dt)
	_, ok := this.trackedDatatypes[qdt]
	if !ok {
		logging.Warnf("EAR: [data_type=%s] Received request to drop key ids %s for a key data type that Query does not"+
			" subscribe to. Ignoring the request.", qdt.String(), KeyIdsToDrop)
		return
	}

	logging.Infof("EAR: [data_type=%s] Received request to drop key ids %v", qdt.String(), KeyIdsToDrop)

	if len(this.encryptors) == 0 {
		logging.Infof("EAR: [data_type=%s] No encryptors registered with the manager. Ignoring the request as no action required.",
			qdt.String())
		return
	}

	this.dropLock.Lock()
	// Perform de-duplication
	for _, keyId := range KeyIdsToDrop {
		found := false
		for _, k := range this.keysToDrop {
			if keyId == k.keyID && qdt == k.dataType {
				found = true
				break
			}
		}

		if !found {
			this.keysToDrop = append(this.keysToDrop, basicKeyInfo{
				keyID:    keyId,
				dataType: qdt,
			})
		}
	}

	this.cond.Broadcast()
	this.dropLock.Unlock()

}

func (this *NodeEncryptionManager) SynchronizeKeyFilesCallback(dt cbauth.KeyDataType) error {
	// Query has no requirement for this as of now
	return nil
}

func (this *NodeEncryptionManager) dropKeysWorker() {
	locked := false
	defer func() {
		recover()
		if locked {
			this.dropLock.Unlock()
		}
		go this.dropKeysWorker()
	}()

	this.dropLock.Lock()
	locked = true
	for {
		if len(this.keysToDrop) == 0 {
			this.cond.Wait()
		} else {
			key := this.keysToDrop[0]
			this.dropLock.Unlock()
			locked = false

			var dropErr errors.Error
			for _, encryptor := range this.encryptors {
				dropErr = encryptor.DropKey(key.dataType, key.keyID)
				if dropErr != nil {
					break
				}
			}

			this.dropLock.Lock()
			locked = true
			copy(this.keysToDrop, this.keysToDrop[1:])
			this.keysToDrop = this.keysToDrop[:len(this.keysToDrop)-1]
			this.dropLock.Unlock()
			locked = false

			if dropErr == nil {
				logging.Infof("EAR: [data_type=%s] Successfully dropped key %s", key.dataType.String(), key.keyID)
			} else {
				logging.Errorf("EAR: [data_type=%s] Error dropping key %s: %v", key.dataType.String(), key.keyID, dropErr)
			}
			cdt := dataTypeToCbauthType(key.dataType)

			// Notify cbauth of completion of key drop
			// If the drop was un-successful, cbauth will retry again after some time
			cerr := cbauth.KeysDropComplete(cdt, dropErr)
			if cerr != nil {
				logging.Errorf("EAR: [data_type=%s] Error notifying cbauth of completion of key drop for key %s: %v",
					key.dataType.String(), key.keyID, cerr)
			}

			this.dropLock.Lock()
			locked = true
		}
	}
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
