//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extparams

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strconv"
)

const (
	_collectionCatalog           = "catalog"
	_collectionCatalogType       = "catalogType"
	_collectionCredentialId      = "credentialId"
	_collectionFormat            = "format"
	_collectionNamespace         = "namespace"
	_collectionTableName         = "tableName"
	_collectionSnapshotId        = "snapshotId"
	_collectionSnapshotTimestamp = "snapshotTimestamp"
	_collectionParallelScans     = "parallelScans"
	_collectionDecimalToDouble   = "decimal-to-double"
	_collectionUid               = "uid"
	_collectionBucket            = "bucket"
	_collectionScope             = "scope"
	_collectionName              = "name"
	_collectionCompatVersion     = "compat_version"
	CollectionRevison            = "rev"
)

var collectionParamsTypes = map[string]any{CollectionRevison: "", _collectionFormat: "", _collectionNamespace: "",
	_collectionTableName: "", _collectionSnapshotId: "", _collectionSnapshotTimestamp: "",
	_collectionParallelScans: 1, _collectionDecimalToDouble: false,
	_collectionCatalog: "", _collectionCatalogType: "", _collectionCredentialId: "",
	_collectionUid: "", _collectionBucket: "", _collectionScope: "", _collectionName: "", _collectionCompatVersion: 1}

// Valid parameters for each catalog type
var collectionMandatoryTypeParams = map[string][]string{
	CatalogTypeIceberg: {_collectionCatalog, _collectionCatalogType, _collectionCredentialId, _collectionNamespace, _collectionTableName},
}

var collectionOptinalTypeParams = map[string][]string{
	CatalogTypeIceberg: {CollectionRevison, _collectionSnapshotId, _collectionSnapshotTimestamp,
		_collectionParallelScans, _collectionDecimalToDouble,
		_collectionFormat, _collectionUid, _collectionBucket, _collectionScope, _collectionName, _collectionCompatVersion},
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
			sv := getValType(v)
			stv := getValType(tv)
			if sv != stv {
				msg := fmt.Sprintf("Parameter value type: '%s'. Expected type: '%s'", sv, stv)
				rv[k] = &ExternalParamsError{msg, "Parameter value type not matched."}
			}
		}
	}
	for k, _ := range typeParams {
		if _, exists := params[k]; !exists {
			rv[k] = &ExternalParamsError{"not specified", "Parameter must be specified."}
		}
	}

	return rv
}

func SetExternalCollectionInfo(collection, catalog, catalogType, credential string, params map[string]any) map[string]any {
	nparams := maps.Clone(params)
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
	Collection        string `json:"name"`
	Revision          int    `json:"rev,omitempty"`
	Format            string `json:"format,omitempty"`
	Catalog           string `json:"catalog"`
	CatalogType       string `json:"catalogType"`
	CredentialId      string `json:"credentialId"`
	Namespace         string `json:"namespace"`
	TableName         string `json:"tableName"`
	SnapshotId        string `json:"snapshotId,omitempty"`
	SnapshotTimestamp string `json:"snapshotTimestamp,omitempty"`
	ParallelScans     int    `json:"parallelScans,omitempty"`
	DecimalToDouble   bool   `json:"decimal-to-double,omitempty"`
	Uid               uint64
	CatalogInfo       *CatalogEntry `json:"-"`
}

// UnmarshalJSON handles ns_server storing int/bool fields as JSON strings
// (e.g. parallelScans:"2" instead of parallelScans:2) due to form-encoding.
func (e *ExternalCollectionEntry) UnmarshalJSON(data []byte) error {
	type Alias ExternalCollectionEntry
	aux := &struct {
		ParallelScans   json.RawMessage `json:"parallelScans,omitempty"`
		DecimalToDouble json.RawMessage `json:"decimal-to-double,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.ParallelScans) > 0 {
		var n int
		if err := json.Unmarshal(aux.ParallelScans, &n); err == nil {
			e.ParallelScans = n
		} else {
			var s string
			if err := json.Unmarshal(aux.ParallelScans, &s); err != nil {
				return fmt.Errorf("invalid parallelScans: %s", aux.ParallelScans)
			}
			n, err := strconv.Atoi(s)
			if err != nil {
				return fmt.Errorf("invalid parallelScans value %q: %v", s, err)
			}
			e.ParallelScans = n
		}
	}
	if len(aux.DecimalToDouble) > 0 {
		var b bool
		if err := json.Unmarshal(aux.DecimalToDouble, &b); err == nil {
			e.DecimalToDouble = b
		} else {
			var s string
			if err := json.Unmarshal(aux.DecimalToDouble, &s); err != nil {
				return fmt.Errorf("invalid decimal-to-double: %s", aux.DecimalToDouble)
			}
			switch s {
			case "true":
				e.DecimalToDouble = true
			case "false":
				e.DecimalToDouble = false
			default:
				return fmt.Errorf("invalid decimal-to-double value %q", s)
			}
		}
	}
	return nil
}
