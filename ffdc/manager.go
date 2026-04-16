//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ffdc

import (
	go_errors "errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

var ffdcMgr = createFFDCManager()

type ffdcManager struct {
	sync.RWMutex
	encryptionProvider encryption.EncryptionProvider

	// List of files that are deemed as orphans due to errors
	// Orphan FFDC files are considered in key tracking to keep key-in-use reporting correct even when file operations fail
	orphanFiles []*ffdcFile
}

func createFFDCManager() *ffdcManager {
	return &ffdcManager{
		orphanFiles: make([]*ffdcFile, 0),
	}
}

func (this *ffdcManager) getActiveKey() (*encryption.EaRKey, errors.Error) {
	if this.encryptionProvider == nil {
		return nil, errors.NewEncryptionError(errors.E_NO_ENCRYPTION_MANAGER, nil)
	}
	return this.encryptionProvider.GetActiveKey(encryption.KeyDataType{TypeName: encryption.LOG_KEY_DATATYPE})
}

func (this *ffdcManager) getKey(keyId string) (*encryption.EaRKey, errors.Error) {
	if this.encryptionProvider == nil {
		return nil, errors.NewEncryptionError(errors.E_NO_ENCRYPTION_MANAGER, nil)
	}
	return this.encryptionProvider.GetKey(encryption.KeyDataType{TypeName: encryption.LOG_KEY_DATATYPE}, keyId)
}

func (this *ffdcManager) InitEncryptionProvider(encProvider encryption.EncryptionProvider) {
	this.encryptionProvider = encProvider
}

func (this *ffdcManager) Name() string {
	return "FFDC"
}

// Returns the the list of all key IDs of all sensntive FFDC files.
// Includes the special unencrypted key ID if there are sensitive files that are not encrypted.
func (this *ffdcManager) GetInUseKeys(dt encryption.KeyDataType) ([]string, error) {
	if dt.TypeName != encryption.LOG_KEY_DATATYPE {
		return []string{}, nil
	}

	keys := make(map[string]bool, 12)
	for _, r := range reasons {
		r.RLock()
		for _, occ := range r.occurrences {
			occ.filesLock.RLock()
			for _, f := range occ.files {
				if !f.Sensitive() {
					continue
				}

				f.lock.RLock()
				currKeyId := f.CurrentKeyId(false)
				if currKeyId != _UNSET_KEY_ID {
					keys[currKeyId] = true
				}

				targetKeyId := f.TargetKeyId(false)
				if targetKeyId != _UNSET_KEY_ID {
					keys[targetKeyId] = true
				}
				f.lock.RUnlock()

			}
			occ.filesLock.RUnlock()
		}
		r.RUnlock()
	}

	this.RLock()
	for _, f := range this.orphanFiles {
		f.lock.RLock()
		currKeyId := f.CurrentKeyId(false)
		if currKeyId != _UNSET_KEY_ID {
			keys[currKeyId] = true
		}

		targetKeyId := f.TargetKeyId(false)
		if targetKeyId != _UNSET_KEY_ID {
			keys[targetKeyId] = true
		}
		f.lock.RUnlock()
	}
	this.RUnlock()

	keysInUse := make([]string, 0, len(keys))
	for k := range keys {
		keysInUse = append(keysInUse, k)
	}
	return keysInUse, nil
}

// Dropping a key requires that no sensitive data on-disk remains encrypted with it.
// If the special "unencrypted" key is to be dropped, un-encrypted data should no longer be on disk.
// FFDC files using the key to be dropped are appropriately transformed (i.e re-encrypted, encrypted, or decrypted)
// to eliminate the key's usage.
// This method is successful only when there are no FFDC files using the key to be dropped
func (this *ffdcManager) DropKey(dt encryption.KeyDataType, keyIdToDrop string) error {
	if dt.TypeName != encryption.LOG_KEY_DATATYPE {
		return nil
	}

	var dropErr error
	var dropErrCount int
	maxAttempts := 2

	// Retry full reason processing if the key is still found in occurrences.
	for attempt := 0; attempt <= maxAttempts; attempt++ {

		// An error during drop is fatal and we should not try to retry the whole drop process.
		// Only retry if there still are files encrypted with the key to drop
		if dropErr != nil {
			return fmt.Errorf("Drop key failed for %v FFDC reasons. Last error: %v", dropErrCount, dropErr)
		}

		if attempt == maxAttempts {
			return fmt.Errorf("Key is still encrypting FFDC file(s)")
		}

		for _, r := range reasons {
			err := r.processForKeyDrop(keyIdToDrop, this)
			if err != nil {
				// Track error and continue processing other reasons even if there is an error in one reason.
				// This is to maximize the number of files that get re-encrypted in this attempt.
				dropErrCount++
				dropErr = err
				continue
			}
		}

		// Check if key is still found in any occurrence files before retrying
		success := true
		for _, r := range reasons {
			r.RLock()
			for _, occ := range r.occurrences {
				occ.filesLock.RLock()
				for _, f := range occ.files {
					if !f.Sensitive() {
						continue
					}

					if f.CurrentKeyId(true) == keyIdToDrop {
						success = false
						break
					}
				}
				occ.filesLock.RUnlock()
			}
			r.RUnlock()
		}

		if success {
			break
		}
	}

	// Cleanup orphan files and retry if the key is still found in orphan files
	for attempt := 0; attempt <= maxAttempts; attempt++ {
		if attempt == maxAttempts {
			return fmt.Errorf("Key is still encrypting orphan file(s)")

		}

		this.cleanupOrphanFiles()

		success := true
		this.RLock()
		for _, of := range this.orphanFiles {
			if of.CurrentKeyId(true) == keyIdToDrop || of.TargetKeyId(true) == keyIdToDrop {
				success = false
				break
			}
		}
		this.RUnlock()

		if success {
			break
		}
	}

	return nil
}

func (this *ffdcManager) trackOrphanFile(file *ffdcFile) {
	this.Lock()
	this.orphanFiles = append(this.orphanFiles, file)
	this.Unlock()
}

func (this *ffdcManager) trackOrphanFileFromName(fileName string) {
	ffdc, queryGenerated, err := genFfdcFile(fileName)
	if err != nil {
		of := &ffdcFile{
			name:         fileName,
			currentKeyId: _UNSET_KEY_ID,
			targetKeyId:  _UNSET_KEY_ID,
		}
		this.trackOrphanFile(of)
		return
	}

	if !queryGenerated {
		return
	}

	if ffdc != nil {
		this.trackOrphanFile(ffdc)
		return
	}
}

func (this *ffdcManager) cleanupOrphanFiles() {
	this.Lock()
	for i := 0; i < len(this.orphanFiles); {
		name := this.orphanFiles[i].Name(true)
		err := os.Remove(path.Join(GetPath(), name))
		if err == nil || go_errors.Is(err, os.ErrNotExist) {
			logging.Infof("FFDC: Orphaned file removed: %v", name)
			copy(this.orphanFiles[i:], this.orphanFiles[i+1:])
			this.orphanFiles = this.orphanFiles[:len(this.orphanFiles)-1]
		} else {
			i++
		}
	}
	this.Unlock()
}
