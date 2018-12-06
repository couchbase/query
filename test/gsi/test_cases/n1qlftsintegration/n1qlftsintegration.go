//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package n1qlFts

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/test/gsi"
)

var IndexName = "fts_index"

func start_cs() *gsi.MockServer {
	return gsi.Start_cs(false)
}

func runStmt(mockServer *gsi.MockServer, q string) ([]interface{}, []errors.Error, errors.Error) {
	return gsi.RunStmt(mockServer, q)
}

func runMatch(filename string, prepared, explain bool, qc *gsi.MockServer, t *testing.T) {
	gsi.RunMatch(filename, prepared, explain, qc, t)
}

func setupftsIndex() error {

	reader := strings.NewReader(
		`{
	"type": "fulltext-index",
		"name": "fts_index",
		"uuid": "1901bb2f07f79808",
		"sourceType": "couchbase",
		"sourceName": "product",
		"sourceUUID": "6715244cdfcf057eb5164a71512a785e",
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

	request, err := http.NewRequest("PUT", gsi.Site_CBS+gsi.Auth_param+"@"+gsi.FTS_CBS+gsi.FTS_API_PATH+IndexName, reader)
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
		return fmt.Errorf(" Failed to create FTS index ")
	}

	time.Sleep(time.Millisecond * 10)
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

	time.Sleep(time.Millisecond * 10)
	return nil
}
