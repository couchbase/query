//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package encryption

import (
	"encoding/json"
	"fmt"

	"github.com/couchbase/query/errors"
)

// Avoid external dependencies in the top-level encryption package

var (
	BUCKET_KEY_DATATYPE = "service_bucket"
	LOG_KEY_DATATYPE    = "log"
	OTHER_KEY_DATATYPE  = "other"
	UNENCRYPTED_KEY_ID  = ""
)

type KeyDataType struct {
	TypeName   string `json:"typeName"`
	BucketUUID string `json:"bucketUUID"` // UUID is used only when Type is "bucket"
}

func (dt KeyDataType) String() string {
	if dt.BucketUUID != "" {
		return fmt.Sprintf("%s(%s)", dt.TypeName, dt.BucketUUID)
	}
	return dt.TypeName
}

type EncrKeysInfo struct {
	ActiveKeyId       string    `json:"activeKeyId"`
	Keys              []*EaRKey `json:"keys"`
	UnavailableKeyIds []string  `json:"unavailableKeyIds"`
}

func (info *EncrKeysInfo) String() string {
	b, _ := json.Marshal(info)
	return string(b)
}

type EaRKey struct {
	Id     string `json:"id"`
	Cipher string `json:"cipher"`
	Key    []byte `json:"-"` // do not marshal sensitive key material
}

func (k *EaRKey) String() string {
	b, _ := json.Marshal(k)
	return string(b)
}

type EncryptionProvider interface {
	GetActiveKey(dt KeyDataType) (*EaRKey, errors.Error)
	GetKey(dt KeyDataType, keyID string) (*EaRKey, errors.Error)
}
