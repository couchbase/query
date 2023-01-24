// Copyright 2018-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// The enterprise edition has access to couchbase/query-ee, which
// includes update statistics. This file is only built in with
// the enterprise edition.

//go:build enterprise
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
