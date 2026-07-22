//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

// package couchbase provides low level access to the KV store and the orchestrator
package couchbase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/extparams"
)

const _CATALOG_PATH = "/pools/default/externalCatalogs"

func (c *Client) CreateCatalog(cred cbauth.Creds, name, catalogType, source, credential string, params map[string]any, ctx context.Context) error {
	nparams := extparams.SetCatalogInfo(name, catalogType, source, credential, params)
	args, err := extparams.GetCatalogObj(nparams)
	if err != nil {
		return err
	}

	return c.parsePostURLResponseTerse(_CATALOG_PATH, cred, args, nil, ctx)
}

func (c *Client) AlterCatalog(cred cbauth.Creds, name string, params map[string]any, ctx context.Context) error {
	var nparams map[string]any
	err := extparams.GetAny(params, &nparams)
	if err != nil {
		return err
	}
	oParms, err := c.GetCatalogObj(cred, name, ctx)
	if err != nil {
		return err
	}

	for k, v := range nparams {
		oParms[k] = v
	}

	_, err = extparams.GetCatalogObj(oParms)
	if err != nil {
		return err
	}
	//	nparams[extparams.CatalogRevison] = oParms[extparams.CatalogRevison]
	target := fmt.Sprintf("%s/%s", _CATALOG_PATH, uriAdj(name))
	return c.parsePatchURLResponseTerse(target, cred, nparams, nil, ctx)
}

func (c *Client) DropCatalog(cred cbauth.Creds, name string, ctx context.Context) error {
	target := fmt.Sprintf("%s/%s", _CATALOG_PATH, uriAdj(name))
	return c.parseDeleteURLResponseTerse(target, cred, nil, nil, ctx)
}

func (c *Client) GetCatalogObj(cred cbauth.Creds, name string, ctx context.Context) (rv map[string]any, err error) {
	target := fmt.Sprintf("%s/%s", _CATALOG_PATH, uriAdj(name))
	err = c.parseURLResponse(target, cred, &rv, ctx)
	if err != nil {
		return
	}
	rv["name"] = name
	return
}

func (c *Client) GetCatalog(cred cbauth.Creds, name string, ctx context.Context) (entry *extparams.CatalogEntry, err error) {
	rv, err := c.GetCatalogObj(cred, name, ctx)
	if err != nil || rv == nil {
		return nil, err
	}
	entry, err = extparams.GetCatalogEntry(rv)
	return
}

func (c *Client) getCatalogsRaw(cred cbauth.Creds, ctx context.Context) (map[string]json.RawMessage, error) {
	var ret map[string]json.RawMessage
	err := c.parseURLResponse(_CATALOG_PATH, cred, &ret, ctx)
	return ret, err
}

func (c *Client) ListCatalogs(cred cbauth.Creds, ctx context.Context) (rv []any, err error) {
	// {"uid":5,"aws_glue":{"rev":1,"credentialId":"awsid","catalogType":"ICEBERG","catalogSource":"AWS_GLUE"}}
	ret, err := c.getCatalogsRaw(cred, ctx)
	if err != nil {
		return nil, err
	}
	rv = make([]any, 0, len(ret))
	for n, v := range ret {
		if n == "uid" {
			continue
		}
		var vm map[string]any
		if json.Unmarshal(v, &vm) == nil {
			vm["name"] = n
			rv = append(rv, vm)
		}
	}
	return
}
