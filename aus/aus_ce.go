//  Copyright 2024-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build !enterprise

package aus

import (
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/server"
	"github.com/couchbase/query/value"
)

func InitAus(server *server.Server) {
	// do nothing
}

// system:aus keyspace related functions

func CountAus() (int64, errors.Error) {
	return -1, errors.NewAusNotSupportedError()
}

func FetchAus() (map[string]interface{}, errors.Error) {
	return nil, errors.NewAusNotSupportedError()
}

func SetAus(settings interface{}, distribute bool) (errors.Error, errors.Errors) {
	return errors.NewAusNotSupportedError(), nil
}

// system:aus_settings keyspace related functions

func ScanAusSettings(bucket string, f func(path string) error) errors.Error {
	return errors.NewAusNotSupportedError()
}

func FetchAusSettings(path string) (value.Value, errors.Errors) {
	return nil, errors.Errors{errors.NewAusNotSupportedError()}

}

func MutateAusSettings(op MutateOp, pair value.Pair, queryContext datastore.QueryContext, preserveMutations bool) (
	int, value.Pairs, errors.Errors) {
	return 0, nil, errors.Errors{errors.NewAusNotSupportedError()}
}

// Scope and Collection level AUS document cleanup functions

func DropScope(namespace string, bucket string, scope string, scopeUid string) {
	// do nothing
}

func DropCollection(namespace string, bucket string, scope string, scopeUid string, collection string, collectionUid string) {
	// do nothing
}

// Backup related functions

func BackupAusSettings(namespace string, bucket string, filter func([]string) bool) ([]interface{}, errors.Error) {
	return nil, nil
}
