//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package n1qlFts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/test/gsi"
)

var IndexName = "fts_index"

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(true)
}

func runStmt(mockServer *gsi.MockServer, q string) *gsi.RunResult {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.SetConsistencyParam(datastore.AT_PLUS)
	gsi.RunMatch(filename, prepared, explain, qc, t)
	gsi.SetConsistencyParam(datastore.AT_PLUS)
}

func isFTSPresent() bool {
	request, err := http.NewRequest("GET", gsi.Site_CBS+gsi.Auth_param+"@"+gsi.Pool_CBS+gsi.NodeServices, strings.NewReader(""))
	if err != nil {
		return false
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return false
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return false
	}

	_, ok := data["nodesExt"].([]interface{})[0].(map[string]interface{})["services"].(map[string]interface{})["fts"]

	return ok
}

func setupftsIndex() error {

	b := []byte(
		`{
	"type": "fulltext-index",
		"name": "fts_index",
		"sourceType": "couchbase",
		"sourceName": "product",
		"planParams": {
	"maxPartitionsPerPIndex": 171
	},
	"params": {
	"doc_config": {
	"docid_prefix_delim": "",
	"docid_regexp": "",
	"mode": "type_field",
	"type_field": "name"
	},
	"mapping": {
	"analysis": {},
	"default_analyzer": "standard",
	"default_datetime_parser": "dateTimeOptional",
	"default_field": "_all",
	"default_mapping": {
	"default_analyzer": "",
	"dynamic": false,
	"enabled": true,
	"properties": {
	"name": {
	"default_analyzer": "",
	"dynamic": false,
	"enabled": true,
	"fields": [
	{
	"include_in_all": true,
	"include_term_vectors": true,
	"index": true,
	"name": "name",
	"store": true,
	"type": "text"
	}
	]
	}
	}
	},
	"default_type": "_default",
	"docvalues_dynamic": true,
	"index_dynamic": true,
	"store_dynamic": false,
	"type_field": "_type"
	},
	"store": {
	"indexType": "upside_down",
	"kvStoreName": "mossStore"
	}
	},
	"sourceParams": {}
}`)

	body := bytes.NewReader(b)

	request, err := http.NewRequest("PUT", gsi.Site_CBS+gsi.FTS_CBS+gsi.FTS_API_PATH+IndexName, body)
	if err != nil {
		return err
	}

	request.SetBasicAuth("Administrator", "password")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data map[string]interface{}

	err = json.Unmarshal(respbody, &data)
	if err != nil {
		return err
	}

	if data["status"] != "ok" {
		return fmt.Errorf(" Failed to create FTS index ")
	}

	return nil
}

func deleteFTSIndex() error {
	request, err := http.NewRequest("DELETE", gsi.Site_CBS+gsi.Auth_param+"@"+gsi.FTS_CBS+gsi.FTS_API_PATH+IndexName, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data map[string]interface{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	if data["status"] != "ok" {
		return fmt.Errorf(" Failed to delete FTS index ")
	}

	return nil
}
