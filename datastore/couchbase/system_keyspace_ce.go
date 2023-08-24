// Copyright 2020-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

//go:build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func (s *store) CreateSystemCBOStats(requestId string) errors.Error {
	return nil
}

func (s *store) DropSystemCBOStats() errors.Error {
	return nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	return false, nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) GetSystemCollection(bucketName string) (datastore.Keyspace, errors.Error) {
	return nil, nil
}

func (s *store) CheckSystemCollection(bucketName, requestId string) errors.Error {
	return nil
}
