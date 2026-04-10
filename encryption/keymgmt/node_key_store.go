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
	"bytes"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

var _MAX_CB_BUCKETS = 30
var _MAX_KEY_DATATYPES = _MAX_CB_BUCKETS + 2 // 30 buckets + 1 for 'logs' + 1 for 'other'

type nodeKeyStore struct {
	// Since keyID of cbauth keys is unique across all key data types, maintain a single map of keyID to key material.
	encrKeysMaterial map[string]*encryption.EaRKey

	// Stores all the key-related information for each key data type
	encrKeysInfo map[encryption.KeyDataType]*encryption.EncrKeysInfo
	lock         sync.RWMutex
}

func newNodeKeyStore() *nodeKeyStore {
	return &nodeKeyStore{}
}

// Attempts to load the key store with key information of all the provided key data types.
// If priming fails for any datatype, it does not mean permanent absence of its key info from the store.
// Any missing entries will be loaded later through future refresh callback triggers or lazy prime on first access of the key.
func (this *nodeKeyStore) PrimeKeys(keyDataTypes []encryption.KeyDataType) errors.Error {
	this.lock.Lock()
	if this.encrKeysMaterial == nil {
		this.encrKeysMaterial = make(map[string]*encryption.EaRKey, _MAX_KEY_DATATYPES)
	}

	if this.encrKeysInfo == nil {
		// Can potentially expand beyond _MAX_KEY_DATATYPES, but start with this initial capacity
		this.encrKeysInfo = make(map[encryption.KeyDataType]*encryption.EncrKeysInfo, _MAX_KEY_DATATYPES)
	}
	this.lock.Unlock()

	for _, dt := range keyDataTypes {
		this.lock.RLock()
		entry, ok := this.encrKeysInfo[dt]
		needsPrime := !ok || entry == nil // Only prime if another path has not already loaded key info for this data type
		this.lock.RUnlock()

		if !needsPrime {
			continue
		}

		cdt := dataTypeToCbauthType(dt)
		this.primeKey(cdt)
	}

	return nil
}

func (this *nodeKeyStore) UpdateKeys(dataType cbauth.KeyDataType, newInfo *cbauth.EncrKeysInfo, prime bool) errors.Error {
	if newInfo == nil {
		return nil
	}

	dt, err := validateKeyDataType(dataType)
	if err != nil {
		return err
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if this.encrKeysMaterial == nil {
		this.encrKeysMaterial = make(map[string]*encryption.EaRKey, _MAX_KEY_DATATYPES)
	}

	if this.encrKeysInfo == nil {
		// Can potentially expand beyond _MAX_KEY_DATATYPES, but start with this initial capacity
		this.encrKeysInfo = make(map[encryption.KeyDataType]*encryption.EncrKeysInfo, _MAX_KEY_DATATYPES)
	}

	// Update the manager if no config exists or the new config differs from the existing config for this type
	currInfo, exists := this.encrKeysInfo[dt]
	var changed bool

	if prime {
		// Update only if there is no entry yet for this data type.
		// If another path (like refresh callback trigger, or a prime attempt by GetActiveKey, etc)
		// already updated it just before this, skip updating in order to prevent replacing newer data with older fetched data.
		if !exists || currInfo == nil {
			changed = true
		} else {
			return nil
		}

	} else if !exists || currInfo == nil {
		changed = true
	} else {
		if currInfo.ActiveKeyId != newInfo.ActiveKeyId {
			changed = true
		} else if len(currInfo.Keys) != len(newInfo.Keys) {
			changed = true
		} else if len(currInfo.UnavailableKeyIds) != len(newInfo.UnavailableKeyIds) {
			changed = true
		} else if !slices.Equal(currInfo.UnavailableKeyIds, newInfo.UnavailableKeyIds) {
			changed = true
		} else {
			for _, n := range newInfo.Keys {
				found := false
				for _, c := range currInfo.Keys {
					if n.Id == c.Id && bytes.Equal(n.Key, c.Key) && n.Cipher == c.Cipher {
						found = true
						break
					}
				}

				if !found {
					changed = true
					break
				}
			}
		}
	}

	if !changed {
		return nil
	}

	if currInfo != nil {
		for _, uk := range currInfo.UnavailableKeyIds {
			delete(this.encrKeysMaterial, uk)
		}

		for _, ck := range currInfo.Keys {
			delete(this.encrKeysMaterial, ck.Id)
		}

	}

	info := &encryption.EncrKeysInfo{
		ActiveKeyId: newInfo.ActiveKeyId,
	}

	if len(newInfo.UnavailableKeyIds) > 0 {
		info.UnavailableKeyIds = make([]string, len(newInfo.UnavailableKeyIds))
		copy(info.UnavailableKeyIds, newInfo.UnavailableKeyIds)
	}

	if len(newInfo.Keys) > 0 {
		info.Keys = make([]*encryption.EaRKey, len(newInfo.Keys))

		for i, k := range newInfo.Keys {
			newKey := &encryption.EaRKey{
				Id:     k.Id,
				Cipher: k.Cipher,
			}

			// Deep copy key material
			newKey.Key = make([]byte, len(k.Key))
			copy(newKey.Key, k.Key)

			info.Keys[i] = newKey

			// Update key material map.
			// Store a copy of the key material pointer to allow updates to EncrKeysInfo.Keys[i] after we update this map
			this.encrKeysMaterial[k.Id] = newKey
		}
	}

	this.encrKeysInfo[dt] = info

	logging.Infof("EAR: [data_type=%s] New encryption-at-rest configuration received. Configuration updated to: %s", dt.String(),
		info.String())

	return nil
}

func (this *nodeKeyStore) GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return this.getKeyHelper(dt, true, "")
}

func (this *nodeKeyStore) GetKey(dt encryption.KeyDataType, keyID string) (*encryption.EaRKey, errors.Error) {
	// Fast path to find key material
	// Ideally this should always work, since the key material map is populated when keys are updated
	this.lock.RLock()
	// Return a copy of the key material pointer to allow configuration updates for this data type after returning it to the caller,
	// without impacting the returned key material.
	key, ok := this.encrKeysMaterial[keyID]
	this.lock.RUnlock()
	if ok {
		return key, nil
	}

	return this.getKeyHelper(dt, false, keyID)
}

// Pass the keyID to find a specific key. If looking for the active key, the input keyID can be empty
func (this *nodeKeyStore) getKeyHelper(dt encryption.KeyDataType, findActiveKey bool, keyID string) (*encryption.EaRKey, errors.Error) {
	// Check if key info is present for the datatype
	this.lock.RLock()
	keyInfo, ok := this.encrKeysInfo[dt]
	needsPrime := !ok || keyInfo == nil
	this.lock.RUnlock()

	// Try priming here only when this data type still has no cached entry. This is a backup path for rare cases where
	// startup priming failed or the refresh callback that updates the entry, is delayed
	if needsPrime {
		err := this.primeKey(dataTypeToCbauthType(dt))
		if err != nil {
			return nil, err
		}
	}

	this.lock.RLock()
	defer this.lock.RUnlock()

	if needsPrime {
		// After priming, check the cached entry again
		keyInfo, ok = this.encrKeysInfo[dt]
		if !ok || keyInfo == nil {
			return nil, nil
		}
	}

	if findActiveKey {
		// If ActiveKeyId is empty, encryption at rest is not enabled for this data type
		if keyInfo == nil || keyInfo.ActiveKeyId == encryption.UNENCRYPTED_KEY_ID {
			return nil, nil
		}

		keyID = keyInfo.ActiveKeyId
	}

	// Return a copy of the key material pointer to allow configuration updates for this data type after returning it to the caller,
	// without impacting the returned key material.
	key, ok := this.encrKeysMaterial[keyID]
	if ok {
		return key, nil
	}

	// Go the long route to find the key material
	// Ideally this should not happen, since the key material map is populated when keys are updated
	for _, k := range keyInfo.Keys {
		if k.Id == keyID {
			return k, nil
		}
	}

	return nil, errors.NewEncryptionError(errors.E_ENCRYPTION_KEY_INFO_NOT_FOUND, nil, keyID, dt.String())
}

func (this *nodeKeyStore) primeKey(dt cbauth.KeyDataType) errors.Error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keys, cbErr := cbauth.GetEncryptionKeysBlocking(ctx, dt)
	if cbErr != nil {
		return errors.NewEncryptionError(errors.E_ENCRYPTION_PRIME, cbErr, cbauthTypeToDataType(dt).String())
	}

	err := this.UpdateKeys(dt, keys, true)
	if err != nil {
		t := cbauthTypeToDataType(dt)
		logging.Errorf("EAR: [data_type=%s] Error priming encryption-at-rest configuration: %v", t.String(), err)
		return errors.NewEncryptionError(errors.E_ENCRYPTION_PRIME, err, t.String())
	}

	return nil
}
