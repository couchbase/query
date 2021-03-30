//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package resolver

import (
	"fmt"
	"strings"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/datastore/couchbase"
	"github.com/couchbase/query/datastore/file"
	"github.com/couchbase/query/datastore/mock"
	"github.com/couchbase/query/errors"
)

func NewDatastore(uri string) (datastore.Datastore, errors.Error) {
	if strings.HasPrefix(uri, ".") || strings.HasPrefix(uri, "/") {
		return file.NewDatastore(uri)
	}

	if strings.HasPrefix(uri, "http:") {
		return couchbase.NewDatastore(uri)
	}

	if strings.HasPrefix(uri, "dir:") {
		return file.NewDatastore(uri[4:])
	}

	if strings.HasPrefix(uri, "file:") {
		return file.NewDatastore(uri[5:])
	}

	if strings.HasPrefix(uri, "mock:") {
		return mock.NewDatastore(uri)
	}

	return nil, errors.NewError(nil, fmt.Sprintf("Invalid datastore uri: %s", uri))
}
