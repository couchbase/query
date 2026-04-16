//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package ffdc

import (
	"bufio"
	"compress/gzip"
	go_errors "errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
)

type ffdcFile struct {
	lock sync.RWMutex
	name string

	// If the content of this file contains sensitive data
	// Only files with sensitive data will be encrypted when encryption at rest is enabled
	// And only sensitive files will need to be tracked by the encryption key manager
	sensitive bool

	// The key ID that is currently encrypting the sensitive file
	// If the file is not encrypted, this will be set to encryption.UNENCRYPTED_KEY_ID
	currentKeyId string

	// During key drop, a sensitive file may be transformed (i.e re-encrypted, decrypted, encrypted).
	// This is the intended encryption key ID after transformation.
	// It is the key ID (or the unencrypted key ID) that the file should have once transformation completes.
	// This is the key ID that the transformed file is going to encrypted by this will be set to encryption.UNENCRYPTED_KEY_ID
	targetKeyId string
}

func (this *ffdcFile) Name(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.name
}

func (this *ffdcFile) Sensitive() bool {
	// Does not need to be read under lock as it will not be changed after file creation
	return this.sensitive
}

func (this *ffdcFile) CurrentKeyId(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.currentKeyId
}

func (this *ffdcFile) TargetKeyId(lock bool) string {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.targetKeyId
}

func (this *ffdcFile) AllFields(lock bool) (name string, sensitive bool, currentKeyId string, targetKeyId string) {
	if lock {
		this.lock.RLock()
		defer this.lock.RUnlock()
	}
	return this.name, this.sensitive, this.currentKeyId, this.targetKeyId
}

func (this *ffdcFile) setName(name string, lock bool) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.name = name
}

func (this *ffdcFile) setCurrentKeyId(currentKeyId string, lock bool) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.currentKeyId = currentKeyId
}

func (this *ffdcFile) setTargetKeyId(targetKeyId string, lock bool) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.targetKeyId = targetKeyId
}

func (this *ffdcFile) resetAfterDropKeyTransformation(name string, currentKeyId string, targetKeyId string, lock bool) {
	if lock {
		this.lock.Lock()
		defer this.lock.Unlock()
	}
	this.name = name
	this.currentKeyId = currentKeyId
	this.targetKeyId = targetKeyId
}

func (this *ffdcFile) transformFile(targetKeyID string, stagingEncrypted bool,
	transform func(src *os.File, dst *os.File) error, ffdcMgr *ffdcManager) error {

	this.lock.Lock()
	this.setTargetKeyId(targetKeyID, false)
	originalName, sensitive, currentKeyId, _ := this.AllFields(false)
	this.lock.Unlock()

	ffdcPath := path.Join(GetPath(), originalName)
	tmpName := getStagingFileName(originalName, stagingEncrypted)
	tmpPath := path.Join(GetPath(), tmpName)

	wait := 100 * time.Millisecond
	maxAttempts := 5
	var ferr error
	var src *os.File
	var dst *os.File

	for attempt := 0; attempt < maxAttempts; attempt++ {
		src, ferr = os.Open(ffdcPath)
		if ferr != nil {
			if go_errors.Is(ferr, os.ErrNotExist) {
				// Nothing to process
				this.lock.Lock()
				this.setCurrentKeyId(_UNSET_KEY_ID, false)
				this.setTargetKeyId(_UNSET_KEY_ID, false)
				this.lock.Unlock()
				return nil
			}
		} else {
			dst, ferr = os.Create(tmpPath)
			if ferr != nil {
				src.Close()
			}
		}

		if ferr == nil {
			break
		}

		if attempt < maxAttempts-1 {
			time.Sleep(wait)
		}
	}

	if ferr != nil {
		// Initial file handling failed. Return error
		this.setTargetKeyId(_UNSET_KEY_ID, true)
		return ferr
	}

	// A transformation error is fatal and will not be retried
	transformErr := transform(src, dst)
	dst.Sync()
	dst.Close()
	src.Close()

	finalName := strings.TrimPrefix(tmpName, reencryptPrefix)
	var renameErr error
	if transformErr == nil {
		var oldPath string
		var newPath string
		var sameName bool
		if finalName == originalName {
			oldPath = tmpPath
			newPath = ffdcPath
			sameName = true
		} else {
			oldPath = tmpPath
			newPath = path.Join(GetPath(), finalName)
		}

		// Rename the transformed file to its intended ffdc file name
		renameErr = withRetry(maxAttempts, wait, true,
			func() error {
				return os.Rename(oldPath, newPath)
			})

		if renameErr == nil {
			// In certain transformations, the FFDC file name's extension changes, so a simple rename is not enough
			// encrypted -> unencrypted: FFDC file will now have the ".gz" extension
			// unencrypted -> encrypted: FFDC file will no longer have the ".gz" extension
			// In these scenarios, renaming the transformed file to the intended name is not enough. As the original file still
			// remains.  So after a successful rename, the original file must be deleted.
			// If re-encryption scenario, this extension change does not happen and rename is enough (it replaces the original).
			if sameName {
				this.resetAfterDropKeyTransformation(finalName, targetKeyID, _UNSET_KEY_ID, true)
				return nil
			} else {
				// Delete the original file
				var removeErr error
				removeErr = withRetry(maxAttempts, wait, true, func() error {
					return os.Remove(ffdcPath)
				})

				if removeErr != nil {
					if go_errors.Is(removeErr, os.ErrNotExist) {
						removeErr = nil
					} else {
						of := &ffdcFile{
							name:         originalName,
							sensitive:    sensitive,
							currentKeyId: currentKeyId,
							targetKeyId:  _UNSET_KEY_ID,
						}
						ffdcMgr.trackOrphanFile(of)
					}
				}

				this.resetAfterDropKeyTransformation(finalName, targetKeyID, _UNSET_KEY_ID, true)
				return removeErr
			}
		}
	}

	if transformErr != nil || renameErr != nil {
		err := withRetry(maxAttempts, wait, true, func() error {
			return os.Remove(tmpPath)
		})

		if err != nil && !go_errors.Is(err, os.ErrNotExist) {
			logging.Errorf("FFDC: Failed to remove staging file %v after failed transform for %v: %v",
				tmpName, originalName, err)
			of := &ffdcFile{
				name:         tmpName,
				sensitive:    targetKeyID != encryption.UNENCRYPTED_KEY_ID,
				currentKeyId: targetKeyID,
				targetKeyId:  _UNSET_KEY_ID,
			}
			ffdcMgr.trackOrphanFile(of)
		}
	}

	var rv error
	if transformErr != nil {
		rv = fmt.Errorf("Failed to transform file %v: %v", originalName, transformErr)
	} else if renameErr != nil {
		rv = fmt.Errorf("Failed to replace original file %s with the transformed file: %v", originalName, renameErr)
	}

	this.setTargetKeyId(_UNSET_KEY_ID, true)
	return rv
}

func (this *ffdcFile) transformForKeyDrop(keyIdToDrop string, activeKey *encryption.EaRKey, ffdcMgr *ffdcManager) error {
	// unencrypted -> encrypted (active key)
	if keyIdToDrop == encryption.UNENCRYPTED_KEY_ID {
		return this.transformFile(activeKey.Id, true, func(src *os.File, dst *os.File) error {
			br := bufio.NewReader(src)
			gr, err := gzip.NewReader(br)
			if err != nil {
				return err
			}

			err = encryption.EncryptFileAsCBEF(gr, dst, activeKey, encryption.CBEF_ZLIB, _ENCRYPTION_BUFFER_SIZE)
			gr.Close()

			return err
		}, ffdcMgr)
	}

	// encrypted -> unencrypted
	if activeKey == nil {
		return this.transformFile(encryption.UNENCRYPTED_KEY_ID, false, func(src *os.File, dst *os.File) error {
			keyId := this.CurrentKeyId(true)
			key, err := ffdcMgr.getKey(keyId)
			if err != nil {
				return err
			}

			zip := gzip.NewWriter(dst)
			bw := bufio.NewWriter(zip)

			derr := encryption.DecryptCBEFFile(src, bw, func(keyID string) (*encryption.EaRKey, errors.Error) {
				return key, nil
			})

			bw.Flush()
			zip.Close()

			return derr
		}, ffdcMgr)
	}

	// encrypted (old key) -> encrypted (active key)
	return this.transformFile(activeKey.Id, true, func(src *os.File, dst *os.File) error {
		keyId := this.CurrentKeyId(true)
		key, err := ffdcMgr.getKey(keyId)
		if err != nil {
			return err
		}
		return encryption.ReEncryptCBEFFile(src, dst, func(keyID string) (*encryption.EaRKey, errors.Error) {
			return key, nil
		}, activeKey)
	}, ffdcMgr)
}

// Returns the ffdcFile object and a boolean indicating whether the file was generated by the service
func genFfdcFile(fileName string) (*ffdcFile, bool, error) {
	ffdc := &ffdcFile{
		name:        fileName,
		sensitive:   checkFileSensitive(fileName),
		targetKeyId: _UNSET_KEY_ID,
	}

	// Check if file is encrypted and get keyID if it is
	if strings.HasSuffix(fileName, unencryptedFileExtension) {
		ffdc.currentKeyId = encryption.UNENCRYPTED_KEY_ID
		return ffdc, true, nil
	}

	f, err := os.Open(path.Join(GetPath(), fileName))
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	encrypted, keyID := encryption.GetKeyIdFromCBEF(f)

	// If a file without ".gz" extension is not encrypted, then it was not a file created by the service
	if !encrypted {
		return nil, false, nil
	}

	ffdc.currentKeyId = keyID

	return ffdc, true, nil

}

func getStagingFileName(fileName string, isStagingEncrypted bool) string {
	fileName = strings.TrimSuffix(fileName[len(fileNamePrefix)+1:], unencryptedFileExtension)

	var sb strings.Builder
	sb.WriteString(reencryptFileNamePrefix)
	sb.WriteString("_")
	sb.WriteString(fileName)

	if isStagingEncrypted {
		return sb.String()
	} else {
		sb.WriteString(unencryptedFileExtension)
		return sb.String()
	}
}

// Extracts action from file name in order to determine if the file has sensitive data
// Supports files with query_ffdc_* and reencrypt_ffdc_* names
// Returns false if invalid action/file name
func checkFileSensitive(fileName string) bool {
	var prefix string
	if strings.HasPrefix(fileName, reencryptFileNamePrefix) {
		prefix = reencryptFileNamePrefix
	} else if strings.HasPrefix(fileName, fileNamePrefix) {
		prefix = fileNamePrefix
	} else {
		return false
	}

	parts := strings.Split(fileName[len(prefix)+1:], "_")
	if len(parts) < 4 {
		return false
	}

	action := parts[1]
	if ok := sensitiveActions[action]; ok {
		return true
	}

	return false

}

func withRetry(maxAttempts int, wait time.Duration, backoff bool, op func() error) error {
	var err error
	for attempts := 0; attempts < maxAttempts; attempts++ {
		err = op()
		if err == nil {
			return nil
		}
		if attempts < maxAttempts-1 {
			time.Sleep(wait)
			if backoff {
				wait *= 2
			}
		}
	}
	return err
}
