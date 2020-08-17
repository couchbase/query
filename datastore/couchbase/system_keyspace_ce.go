// Copyright (c) 2020 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you
// may not use this file except in compliance with the License. You
// may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Currently, the community edition does not have access to update
// statistics, so this stub returns an error.

// +build !enterprise

package couchbase

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func (s *store) CreateSystemCBOStats() errors.Error {
	return nil
}

func (s *store) HasSystemCBOStats() (bool, errors.Error) {
	return false, nil
}

func (s *store) GetSystemCBOStats() (datastore.Keyspace, errors.Error) {
	return nil, nil
}
