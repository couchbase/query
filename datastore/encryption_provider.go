//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package datastore

import (
	"github.com/couchbase/query/encryption"
	"github.com/couchbase/query/errors"
)

var NoopEncryptionProviderInstance = &NoopEncryptionProvider{}

type NoopEncryptionProvider struct{}

func (NoopEncryptionProvider) GetActiveKey(dt encryption.KeyDataType) (*encryption.EaRKey, errors.Error) {
	return nil, nil
}

func (NoopEncryptionProvider) GetKey(dt encryption.KeyDataType, keyID string) (*encryption.EaRKey, errors.Error) {
	return nil, nil
}
