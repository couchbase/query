//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extparams

import (
	"fmt"
	"maps"
	"reflect"
	"strconv"
)

const (
	_collectionCatalog           = "catalog"
	_collectionCatalogType       = "catalogType"
	_collectionCredentialId      = "credentialId"
	_collectionRevison           = "rev"
	_collectionFormat            = "format"
	_collectionNamespace         = "namespace"
	_collectionTableName         = "tableName"
	_collectionSnapshotId        = "snapshotId"
	_collectionSnapshotTimestamp = "snapshotTimestamp"
	_collectionParallelScans     = "parallelScans"
	_collectionUid               = "uid"
	_collectionBucket            = "bucket"
	_collectionScope             = "scope"
	_collectionName              = "name"
)

var collectionParamsTypes = map[string]any{_collectionRevison: "", _collectionFormat: "", _collectionNamespace: "",
	_collectionTableName: "", _collectionSnapshotId: "", _collectionSnapshotTimestamp: "", _collectionParallelScans: 1,
	_collectionCatalog: "", _collectionCatalogType: "", _collectionCredentialId: "",
	_collectionUid: "", _collectionBucket: "", _collectionScope: "", _collectionName: ""}

// Valid parameters for each catalog type
var collectionMandatoryTypeParams = map[string][]string{
	CatalogTypeIceberg: {_collectionCatalog, _collectionCatalogType, _collectionCredentialId, _collectionNamespace, _collectionTableName},
}

var collectionOptinalTypeParams = map[string][]string{
	CatalogTypeIceberg: {_collectionRevison, _collectionSnapshotId, _collectionSnapshotTimestamp, _collectionParallelScans,
		_collectionFormat, _collectionUid, _collectionBucket, _collectionScope, _collectionName},
}

func validateCollection(params map[string]any) map[string]*ExternalParamsError {
	rv := make(map[string]*ExternalParamsError, len(params))
	typeParams := make(map[string]any)
	typeOptinalParams := make(map[string]any)
	catalogType := validateCatalogParm(params, _collectionCatalogType, rv)
	if catalogType == "" {
		return rv
	}

	for _, s := range collectionMandatoryTypeParams[catalogType] {
		typeParams[s] = collectionParamsTypes[s]
	}
	for _, s := range collectionOptinalTypeParams[catalogType] {
		typeOptinalParams[s] = collectionParamsTypes[s]
	}

	for k, v := range params {
		tv, exists := typeParams[k]
		if !exists {
			optv, opexists := typeOptinalParams[k]
			if !opexists {
				rv[k] = &ExternalParamsError{"unsupported", "Parameter not supported."}
				continue
			}
			tv = optv
		}
		if reflect.TypeOf(v) != reflect.TypeOf(tv) {
			msg := fmt.Sprintf("Parameter value type: '%s'. Expected type: '%s'", getValType(v), getValType(tv))
			rv[k] = &ExternalParamsError{msg, "Parameter value type not matched."}
		}
	}
	for k, _ := range typeParams {
		if _, exists := params[k]; !exists {
			rv[k] = &ExternalParamsError{"not specified", "Parameter must be specified."}
		}
	}

	return rv
}

func SetExternalCollectionInfo(bucket, scope, collection, catalog, catalogType, credential string, params map[string]any) map[string]any {
	nparams := maps.Clone(params)
	nparams[_collectionBucket] = bucket
	nparams[_collectionScope] = scope
	nparams[_collectionCatalog] = catalog
	nparams[_collectionCatalogType] = catalogType
	nparams[_collectionCredentialId] = credential
	nparams[_collectionName] = collection
	return nparams
}

func GetExternalCollectionCatalog(params map[string]any) (catalog string) {
	nparams := maps.Clone(params)
	if v, ok := nparams[_collectionCatalog]; ok {
		if s, sok := v.(string); sok {
			catalog = s
		}
	}
	return
}

func GetCollectionObj(params map[string]any) (map[string]any, error) {
	var nparams map[string]any

	if err := GetAny(params, &nparams); err != nil {
		return nil, err
	}

	rv := validateCollection(nparams)

	if len(rv) > 0 {
		m := make(map[string]string)
		for k, v := range rv {
			m[k] = fmt.Sprintf("(%v) %s", v.Error, v.Message)
		}
		return nil, fmt.Errorf("Validation of collection entry failed: %v", m)
	}
	return nparams, nil

}

func GetCollectionEntry(params map[string]any) (*ExternalCollectionEntry, error) {
	nparams, err := GetCollectionObj(params)
	if err != nil {
		return nil, err
	}

	var entry ExternalCollectionEntry
	err = GetAny(nparams, &entry)
	if err != nil {
		return nil, err
	}
	entry.Uid, _ = strconv.ParseUint(entry.SUid, 16, 64)
	return &entry, nil

}

// ExternalCollectionEntry represents an Collection stored in metakv
type ExternalCollectionEntry struct {
	SUid              string `json:"uid"`
	Bucket            string `json:"bucket"`
	Scope             string `json:"scope"`
	Collection        string `json:"name"`
	Revision          string `json:"rev,omitempty"`
	Format            string `json:"format,omitempty"`
	Catalog           string `json:"catalog"`
	CatalogType       string `json:"catalogType"`
	CredentialId      string `json:"credentialId"`
	Namespace         string `json:"namespace"`
	TableName         string `json:"tableName"`
	SnapshotId        string `json:"snapshotId,omitempty"`
	SnapshotTimestamp string `json:"snapshotTimestamp,omitempty"`
	ParallelScans     int    `json:"parallelScans"`
	Uid               uint64
	CatalogInfo       *CatalogEntry
}
