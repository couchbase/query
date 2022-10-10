// Copyright (c) 2018 Couchbase, Inc.
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
// The enterprise edition has access to couchbase/query-ee, which
// includes update statistics. This file is only built in with
// the enterprise edition.

// +build enterprise

package couchbase

import (
	ustat "github.com/couchbase/query-ee/updstat"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func GetDefaultStatUpdater(store datastore.Datastore) (datastore.StatUpdater, errors.Error) {
	return ustat.NewDefaultStatUpdater(store)
}
