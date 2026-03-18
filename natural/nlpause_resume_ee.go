//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.
//
//go:build enterprise

package natural

import (
	"github.com/couchbase/query-ee/dictionary"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
)

func hasQueryMetadataForNLChat(create bool, requestId, createReason string, waitOnCreate bool) (bool, errors.Error) {
	return dictionary.HasQueryMetadata(create, requestId, createReason, waitOnCreate)
}

func ScanQueryMetadataForNLChat(preScan func(datastore.Keyspace) errors.Error,
	handler func(string, datastore.Keyspace) errors.Error,
	postScan func(datastore.Keyspace) errors.Error) errors.Error {

	hasquerymetadata, err := hasQueryMetadataForNLChat(false, "", "", false)
	if err != nil {
		return err
	}
	if !hasquerymetadata {
		return nil
	}
	return datastore.ScanSystemCollection(dictionary.QUERY_METADATA_BUCKET, CHAT_DOC_PREFIX, preScan, handler, postScan)
}
