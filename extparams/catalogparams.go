//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package extparams

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strconv"
	"strings"
)

type ExternalParamsError struct {
	Error   string
	Message string
}

// Valid catalog types (stored and compared in uppercase, but accept lowercase input)
const (
	CatalogTypeIceberg = "ICEBERG"
)

// Valid source types (stored and compared in uppercase, but accept lowercase input)
const (
	CatalogSourceAWSGlue          = "AWS_GLUE"
	CatalogSourceAWSGlueRest      = "AWS_GLUE_REST"
	CatalogSourceS3Tables         = "S3_TABLES"
	CatalogSourceBiglakeMetastore = "BIGLAKE_METASTORE"
	CatalogSourceNessie           = "NESSIE"
	CatalogSourceNessieRest       = "NESSIE_REST"
)

// Catalog parameter names
const (
	_catalogName                    = "name"
	_catalogType                    = "catalogType"
	_catalogSource                  = "catalogSource"
	_catalogCredentialId            = "credentialId"
	_catalogRevison                 = "rev"
	_catalogUid                     = "uid"
	_catalogParamURI                = "uri"
	_catalogParamWarehouse          = "warehouse"
	_catalogParamSigV4SigningName   = "sigv4SigningName"
	_catalogParamSigV4SigningRegion = "sigv4SigningRegion"
	_catalogParamQuotaProjectID     = "quotaProjectId"
)

var catalogParamsTypes = map[string]any{_catalogName: "", _catalogType: "", _catalogSource: "", _catalogCredentialId: "",
	_catalogRevison: "", _catalogUid: "", _catalogParamURI: "", _catalogParamWarehouse: "", _catalogParamSigV4SigningName: "",
	_catalogParamSigV4SigningRegion: "", _catalogParamQuotaProjectID: ""}

var catalogMandatoryTypeParams = map[string][]string{
	CatalogTypeIceberg: {_catalogName, _catalogType, _catalogSource, _catalogCredentialId},
}

var catalogOptionalTypeParams = map[string][]string{
	CatalogTypeIceberg: {_catalogRevison, _catalogUid},
}

// Valid parameters for each source type
var catalogSourceTypeParams = map[string][]string{
	CatalogSourceAWSGlue:          {},
	CatalogSourceAWSGlueRest:      {_catalogParamURI, _catalogParamSigV4SigningRegion},
	CatalogSourceS3Tables:         {_catalogParamURI, _catalogParamSigV4SigningRegion, _catalogParamSigV4SigningName, _catalogParamWarehouse},
	CatalogSourceBiglakeMetastore: {_catalogParamURI, _catalogParamWarehouse, _catalogParamQuotaProjectID},
	CatalogSourceNessie:           {_catalogParamURI, _catalogParamWarehouse},
	CatalogSourceNessieRest:       {_catalogParamURI},
}

var validCatalogTypes = map[string]bool{
	CatalogTypeIceberg: true,
}

func validCatalogKeyValue(key, v string) (bool, []string) {
	var typeMandatoryParams map[string][]string
	var typeOptionalParams map[string][]string

	switch key {
	case _catalogType:
		typeMandatoryParams = catalogMandatoryTypeParams
		typeOptionalParams = catalogOptionalTypeParams
	case _catalogSource:
		typeMandatoryParams = catalogSourceTypeParams
	}

	if _, ok := typeMandatoryParams[v]; ok {
		return true, nil
	}
	if _, ok := typeOptionalParams[v]; ok {
		return true, nil
	}
	types := make([]string, 0, len(typeMandatoryParams))

	for t, _ := range typeMandatoryParams {
		types = append(types, t)
	}
	for t, _ := range typeOptionalParams {
		if _, ok := typeMandatoryParams[t]; !ok {
			types = append(types, t)
		}
	}

	return false, types
}

func validateCatalogParm(params map[string]any, key string, rv map[string]*ExternalParamsError) string {
	v, exists := params[key]
	if !exists {
		rv[key] = &ExternalParamsError{"not specified", "Parameter must be specified."}
	} else if vs, ok := v.(string); ok {
		s := strings.ToUpper(vs)
		if vok, types := validCatalogKeyValue(key, s); !vok {
			msg := fmt.Sprintf("Invalid '%s' value: '%s'. Valid values are: '%s'", key, vs, strings.Join(types, "|"))
			rv[key] = &ExternalParamsError{msg, "Parameter value invalid."}
		} else {
			return s
		}
	} else {
		msg := fmt.Sprintf("Parameter value type: '%s'. Expected type: 'string'", getValType(v))
		rv[key] = &ExternalParamsError{msg, "Parameter value type not matched."}
	}
	return ""
}

func validateCatalog(params map[string]any) map[string]*ExternalParamsError {
	rv := make(map[string]*ExternalParamsError, len(params))

	catalogType := validateCatalogParm(params, _catalogType, rv)
	if catalogType == "" {
		return rv
	}
	source := validateCatalogParm(params, _catalogSource, rv)
	if source == "" {
		return rv
	}

	typeParams := make(map[string]any)
	typeOptinalParams := make(map[string]any)

	for _, s := range catalogMandatoryTypeParams[catalogType] {
		typeParams[s] = catalogParamsTypes[s]
	}
	for _, s := range catalogOptionalTypeParams[catalogType] {
		typeOptinalParams[s] = catalogParamsTypes[s]
	}

	for _, s := range catalogSourceTypeParams[source] {
		typeParams[s] = catalogParamsTypes[s]
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

type CatalogEntry struct {
	Name               string `json:"name"`
	CatalogType        string `json:"catalogType"`
	Source             string `json:"catalogSource"`
	CredentialId       string `json:"credentialId"`
	Revision           string `json:"rev,omitempty"`
	URI                string `json:"uri,omitempty"`
	Warehouse          string `json:"warehouse,omitempty"`
	SigV4SigningName   string `json:"sigv4SigningName,omitempty"`
	SigV4SigningRegion string `json:"sigv4SigningRegion,omitempty"`
	QuotaProjectID     string `json:"quotaProjectId,omitempty"`
	SUid               string `json:"uid"`
	Uid                uint64
}

func SetCatalogInfo(name, catalogType, source, credential string, params map[string]any) map[string]any {
	nparams := maps.Clone(params)
	nparams[_catalogName] = name
	nparams[_catalogType] = strings.ToUpper(catalogType)
	nparams[_catalogSource] = strings.ToUpper(source)
	nparams[_catalogCredentialId] = credential
	return nparams
}

func GetAny(params any, rv any) error {
	data, err := json.Marshal(params)
	if err == nil {
		err = json.Unmarshal(data, &rv)
	}
	return err

}

func GetCatalogObj(params map[string]any) (map[string]any, error) {
	var nparams map[string]any

	if err := GetAny(params, &nparams); err != nil {
		return nil, err
	}

	rv := validateCatalog(nparams)

	if len(rv) > 0 {
		m := make(map[string]string)
		for k, v := range rv {
			m[k] = fmt.Sprintf("(%v) %s", v.Error, v.Message)
		}
		return nil, fmt.Errorf("Validation of catalog failed: %v", m)
	}
	return nparams, nil
}

func GetCatalogEntry(params map[string]any) (*CatalogEntry, error) {
	nparams, err := GetCatalogObj(params)
	if err != nil {
		return nil, err
	}

	var entry CatalogEntry
	err = GetAny(nparams, &entry)
	if err != nil {
		return nil, err
	}
	entry.Uid, _ = strconv.ParseUint(entry.SUid, 16, 64)
	return &entry, nil
}

func getValType(v any) string {
	switch v.(type) {
	case float64:
		if vsf, ok := v.(float64); ok && vsf == float64(int64(vsf)) {
			return "int"
		}
	}
	return reflect.TypeOf(v).String()
}
