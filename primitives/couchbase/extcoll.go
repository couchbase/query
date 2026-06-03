//  Copyright 2026-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with Business Source License, use of the
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package couchbase

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/couchbase/cbauth"
	"github.com/couchbase/query/extparams"
)

const _CREATE_EXTERNAL_COLLECTION_PATH = "/pools/default/buckets/%s/scopes/%s/collections/?external=1"
const _ALTER_EXTERNAL_COLLECTION_PATH = "/pools/default/buckets/%s/scopes/%s/collections/%s/?external=1"
const _DROP_EXTERNAL_COLLECTION_PATH = "/pools/default/buckets/%s/scopes/%s/collections/%s/?external=1"
const _MANIFEST_EXTERNAL_COLLECTION_PATH = "/pools/default/buckets/%s/scopes/?external=1"

func (b *Bucket) CreateExternalCollection(cred cbauth.Creds, scope, name, catalog, credential string, params map[string]any) error {
	b.RLock()
	client := b.pool.client
	b.RUnlock()

	if catalog == "" {
		return fmt.Errorf("catalog name is empty '%s'", catalog)
	}
	catalogEntry, err := client.GetCatalog(nil, catalog)
	if err != nil || catalogEntry == nil {
		return fmt.Errorf("failed to get catalog '%s': %v", catalog, err)
	}

	nparams := extparams.SetExternalCollectionInfo(name, catalog, catalogEntry.CatalogType, credential, params)

	args, err := extparams.GetCollectionObj(nparams)
	if err != nil {
		return err
	}
	target := fmt.Sprintf(_CREATE_EXTERNAL_COLLECTION_PATH, uriAdj(b.Name), uriAdj(scope))
	return client.parsePostURLResponseTerse(target, cred, args, nil)
}

func (b *Bucket) AlterExternalCollection(cred cbauth.Creds, scope, name string, params map[string]any) error {
	b.RLock()
	client := b.pool.client
	b.RUnlock()

	var nparams map[string]any
	err := extparams.GetAny(params, &nparams)
	if err != nil {
		return err
	}
	oParms, err := b.GetExternalCollectionObj(scope, name)
	if err != nil {
		return err
	}

	for k, v := range nparams {
		oParms[k] = v
	}

	_, err = extparams.GetCollectionObj(oParms)
	if err != nil {
		return err
	}
	//	nparams[extparams.CollectionRevison] = oParms[extparams.CollectionRevison]
	target := fmt.Sprintf(_ALTER_EXTERNAL_COLLECTION_PATH, uriAdj(b.Name), uriAdj(scope), uriAdj(name))
	return client.parsePatchURLResponseTerse(target, cred, nparams, nil)
}

func (b *Bucket) DropExternalCollection(cred cbauth.Creds, scope, name string) error {
	b.RLock()
	client := b.pool.client
	b.RUnlock()

	target := fmt.Sprintf(_DROP_EXTERNAL_COLLECTION_PATH, uriAdj(b.Name), uriAdj(scope), uriAdj(name))
	return client.parseDeleteURLResponseTerse(target, cred, nil, nil)
}

func (b *Bucket) GetExternalCollectionObj(scope, name string) (map[string]any, error) {
	entry, err := b.GetExternalCollectionEntry(scope, name)
	if err != nil || entry == nil {
		return nil, nil
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return nil, nil
	}
	var rv map[string]any
	if err = json.Unmarshal(data, &rv); err != nil {
		return nil, err
	}

	return rv, nil
}

func (b *Bucket) GetExternalCollectionEntry(scope, name string) (*extparams.ExternalCollectionEntry, error) {

	mani, err := b.GetExternalCollectionsManifest()
	if err != nil || mani == nil || len(mani.Scopes) == 0 {
		return nil, err
	}
	sc, sok := mani.Scopes[scope]
	if !sok || sc == nil {
		return nil, nil
	}
	col, _ := sc.Collections[name]
	return col, nil

}

type InputExternalManifest struct {
	Uid    string
	Scopes []InputExternalScope
}

type InputExternalScope struct {
	Name        string
	Uid         string
	Collections []extparams.ExternalCollectionEntry
}

type ExternalManifest struct {
	Uid    uint64
	Scopes map[string]*ExternalScope // map by name
}

type ExternalScope struct {
	Name        string
	Uid         uint64
	Collections map[string]*extparams.ExternalCollectionEntry // map by name
}

var _EMPTY_EXTERNAL_MANIFEST *ExternalManifest = &ExternalManifest{Uid: 0, Scopes: map[string]*ExternalScope{}}

// ExternalCollectionsCapable is set at startup by higher-level packages to gate
// external collections support based on cluster capability negotiation.
var ExternalCollectionsCapable = func() bool { return true }

func (b *Bucket) GetExternalCollectionsManifest() (*ExternalManifest, error) {
	if !ExternalCollectionsCapable() {
		return nil, nil
	}
	b.RLock()
	client := b.pool.client
	b.RUnlock()

	var im InputExternalManifest
	target := fmt.Sprintf(_MANIFEST_EXTERNAL_COLLECTION_PATH, uriAdj(b.Name))
	err := client.parseURLResponse(target, nil, &im)
	if err != nil {
		if strings.Contains(err.Error(), HTTP_404) || strings.Contains(err.Error(), HTTP_400) {
			return nil, nil
		}
		return nil, err
	}
	uid, err := strconv.ParseUint(im.Uid, 16, 64)
	if err != nil {
		return nil, err
	}
	mani := &ExternalManifest{Uid: uid, Scopes: make(map[string]*ExternalScope, len(im.Scopes))}
	for _, iscope := range im.Scopes {
		scope_uid, err := strconv.ParseUint(iscope.Uid, 16, 64)
		if err != nil {
			return nil, err
		}
		scope := &ExternalScope{Uid: scope_uid, Name: iscope.Name,
			Collections: make(map[string]*extparams.ExternalCollectionEntry, len(iscope.Collections))}

		for _, icoll := range iscope.Collections {
			scope.Collections[icoll.Collection] = &icoll
		}
		if len(scope.Collections) > 0 {
			mani.Scopes[iscope.Name] = scope
		}
	}

	return mani, nil
}
