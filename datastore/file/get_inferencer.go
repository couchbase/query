// Copyright 2016-Present Couchbase, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL-Couchbase.txt.  As of the Change Date specified in that
// file, in accordance with the Business Source License, use of this software
// will be governed by the Apache License, Version 2.0, included in the file
// licenses/APL2.txt.
//
// The enterprise edition has access to couchbase/query-ee, which
// includes schema inferencing. This file is only built in with
// the enterprise edition.

package file

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	infer "github.com/couchbase/query/inferencer"
)

func GetDefaultInferencer(store datastore.Datastore) (datastore.Inferencer, errors.Error) {
	return infer.NewDefaultSchemaInferencer(store)
}
