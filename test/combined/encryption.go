//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/couchbase/query/logging"
)

// Whether encryption at rest has been enabled or disabled for the currently running iteration, and any issues
// encountered while doing so (which do not fail the test, but are reported by e-mail).
type EncryptionAtRest struct {
	iteration    uint64 // the iteration this state applies to, so failures can be matched up with/reported alongside it
	enabled      bool   // whether this iteration should run (and did successfully start running) with encryption at rest
	keyId        int    // the encryption-at-rest key id in use for this iteration, when enabled
	failed       bool   // whether some step of initialisation/disablement failed
	failureNotes []string
	reported     bool // whether the failure (if any) has already been included in a notification e-mail
}

var Encryption *EncryptionAtRest

func (this *EncryptionAtRest) recordFailure(msg string) {
	this.failed = true
	this.failureNotes = append(this.failureNotes, msg)
	logging.Warnf("Encryption at rest: %s", msg)
}

// Determines, from the "encryptionAtRest" configuration element, whether this iteration should run with encryption at
// rest enabled and, if so, locates (creating it if necessary) the encryption key to use.
//
// "encryptionAtRest": { "every": N } means every Nth iteration is run with encryption at rest enabled.
func setupEncryptionAtRest(config map[string]interface{}, iter uint64) *EncryptionAtRest {
	enc := &EncryptionAtRest{iteration: iter}

	ear, ok := config["encryptionAtRest"].(map[string]interface{})
	if !ok {
		return enc
	}
	every, ok := ear["every"].(float64)
	if !ok || every <= 0 {
		return enc
	}
	if iter%uint64(every) != 0 {
		return enc
	}

	enc.enabled = true

	id, err := getEncryptionKeyId()
	if err != nil {
		enc.recordFailure(fmt.Sprintf("Failed to retrieve encryption-at-rest keys: %v", err))
		enc.enabled = false
		return enc
	}

	if id == -1 {
		logging.Infof("Encryption at rest: no existing key found; creating one.")
		if err := createEncryptionKey(); err != nil {
			enc.recordFailure(fmt.Sprintf("Failed to create encryption-at-rest key: %v", err))
			enc.enabled = false
			return enc
		}
		id, err = getEncryptionKeyId()
		if err != nil {
			enc.recordFailure(fmt.Sprintf("Failed to retrieve encryption-at-rest key id after creation: %v", err))
			enc.enabled = false
			return enc
		}
		if id == -1 {
			enc.recordFailure("No encryption-at-rest key id found after creating a key.")
			enc.enabled = false
			return enc
		}
	}

	enc.keyId = id
	logging.Infof("Encryption at rest: enabled for this iteration using key id %d.", enc.keyId)
	return enc
}

// Retrieves the id of the first configured encryption-at-rest key, or -1 if there is none.
func getEncryptionKeyId() (int, error) {
	resp, err := doNodeGet("/settings/encryptionKeys/")
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return -1, fmt.Errorf("%v: %s", resp.Status, string(b))
	}
	var keys []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return -1, err
	}
	if len(keys) == 0 {
		return -1, nil
	}
	if id, ok := keys[0]["id"].(float64); ok {
		return int(id), nil
	}
	return -1, nil
}

func createEncryptionKey() error {
	body := map[string]interface{}{
		"name": "key-1",
		"type": "cb-server-managed-aes-key-256",
		"data": map[string]interface{}{
			"canBeCached":  true,
			"autoRotation": false,
		},
		"usage": []string{"bucket-encryption", "other-encryption"},
	}
	_, _, err := doNodePostJSON("/settings/encryptionKeys/", body)
	return err
}

// Enables encryption at rest, using the given key id, for the given bucket.  A keyId of -1 disables it.
func setBucketEncryption(bucket string, keyId int) error {
	target := fmt.Sprintf("/pools/default/buckets/%s", url.PathEscape(bucket))
	_, _, err := doNodePost(target, map[string]interface{}{"encryptionAtRestKeyId": keyId}, false)
	return err
}

func setOtherEncryption(keyId int) error {
	body := map[string]interface{}{
		"other": map[string]interface{}{
			"encryptionMethod": "encryptionKey",
			"encryptionKeyId":  keyId,
		},
	}
	_, _, err := doNodePostJSON("/settings/security/encryptionAtRest", body)
	return err
}

func disableOtherEncryption() error {
	_, _, err := doNodePost("/settings/security/encryptionAtRest",
		map[string]interface{}{"other.encryptionMethod": "disabled"}, false)
	return err
}

// Applies (or removes) encryption at rest for the buckets created/found during this iteration's Database.create().
// "bucketNew" indicates, for each bucket touched this iteration, whether it was newly created (true) or an existing
// bucket from a previous run was (re)used (false).  Failures here never fail the test; they are recorded against the
// current Encryption state and reported by e-mail once the iteration completes.
func applyEncryptionAtRest(bucketNew map[string]bool) {
	if Encryption == nil || len(bucketNew) == 0 {
		return
	}

	// give the cluster a little breathing space between bucket creation and encryption (re)configuration, and before
	// data is imported into the buckets
	time.Sleep(5 * time.Second)

	if Encryption.enabled {
		for bucket := range bucketNew {
			if err := setBucketEncryption(bucket, Encryption.keyId); err != nil {
				Encryption.recordFailure(fmt.Sprintf("Failed to enable encryption at rest for bucket `%s`: %v", bucket, err))
			}
		}
		if err := setOtherEncryption(Encryption.keyId); err != nil {
			Encryption.recordFailure(fmt.Sprintf("Failed to enable encryption at rest for \"other\": %v", err))
		}
	} else {
		// only buckets carried over from a previous run (i.e. not newly created this iteration) can already be
		// encrypted and so are the only ones that need explicit disablement
		for bucket, isNew := range bucketNew {
			if isNew {
				continue
			}
			if err := setBucketEncryption(bucket, -1); err != nil {
				Encryption.recordFailure(fmt.Sprintf("Failed to disable encryption at rest for bucket `%s`: %v", bucket, err))
			}
		}
		// only relevant if the cluster itself was carried over from a previous run
		if ClusterReused {
			if err := disableOtherEncryption(); err != nil {
				Encryption.recordFailure(fmt.Sprintf("Failed to disable encryption at rest for \"other\": %v", err))
			}
		}
	}
}

// Builds the e-mail content describing the current Encryption state's failure, prefixed with the iteration number it
// applies to.  Returns nil if there is nothing (new) to report.
func encryptionFailureContent() []interface{} {
	if Encryption == nil || !Encryption.failed || Encryption.reported {
		return nil
	}
	content := make([]interface{}, 0, len(Encryption.failureNotes)+1)
	if Encryption.enabled {
		content = append(content, fmt.Sprintf("Iteration %d: Encryption at rest initialisation failed for one or more "+
			"steps; the iteration ran without full encryption at rest coverage.", Encryption.iteration))
	} else {
		content = append(content, fmt.Sprintf("Iteration %d: Encryption at rest disablement failed for one or more "+
			"steps; the iteration may have run with encryption at rest still enabled where disablement was expected.",
			Encryption.iteration))
	}
	for _, n := range Encryption.failureNotes {
		content = append(content, n)
	}
	return content
}

// Sends a standalone e-mail notification if the current Encryption state recorded a failure that hasn't already been
// reported (e.g. folded into a run-failure e-mail for the same iteration).  Intended to be called once an iteration
// has run to completion so failures are reported even when the iteration otherwise succeeded.
func checkEncryptionNotification() {
	content := encryptionFailureContent()
	if content == nil {
		return
	}
	notify(content...)
	Encryption.reported = true
}
