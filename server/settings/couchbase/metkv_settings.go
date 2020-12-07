//  Copyright (c) 2018 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package couchbase

import (
	"fmt"
	"time"

	"github.com/couchbase/cbauth/metakv"
	"github.com/couchbase/indexing/secondary/common"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const MAX_METAKV_RETRIES = 100

const (
	QueryMetaDir          = "/query/"
	QuerySettingsMetaDir  = QueryMetaDir + "settings/"
	QuerySettingsMetaPath = QuerySettingsMetaDir + "config"
	FTSMetaDir            = "/fts/cbgt/cfg/"
)

// List of parameters to be sent to the indexer
var INDEXERPARAM = map[string]string{
	"query.settings.tmp_space_dir":  "query_tmpspace_dir",
	"query.settings.tmp_space_size": "query_tmpspace_limit",
}

var GLOBALPARAM = map[string]string{
	"query.settings.curl_whitelist":   "curl_whitelist",
	"query.settings.curl_allowedlist": "curl_allowedlist",
}

type Config value.Value

func Subscribe(callb metakv.Callback, path string, cancelCh chan struct{}) {
	go func() {
		fn := func(r int, err error) error {
			if r > 0 {
				logging.Errorf("ERROR: metakv notifier failed (%v)..Retrying %v", err, r)
			}
			// cancelCh is the cancel channel to return contril back to metakv.
			err = metakv.RunObserveChildren(path, callb, cancelCh)
			if err != nil {
				logging.Infof("New susbscription %s done:%v", path, err)
			}
			return err
		}

		// Retry mechanism for above function - no of retries - 100
		rh := common.NewRetryHelper(MAX_METAKV_RETRIES, time.Second, 2, fn)
		err := rh.Run()
		if err != nil {
			logging.Fatalf("ERROR: Settings metakv notifier failed (%v).. Exiting", err)
			return
		}
	}()
}

func SetupSettingsNotifier(callb func(Config), cancelCh chan struct{}) {
	// Callback function that processes the input key value given by metakv.

	metaKvCallback := func(path string, val []byte, rev interface{}) error {
		if path == QuerySettingsMetaPath {
			logging.Infof("New settings received: %s", string(val))

			// To be able to process these settings correctly, convert to a map
			// from string to value.Value.

			// This function will also type check each input value.
			newConfig, err := valConvert(val)

			if err != nil {
				// Invalid values log this
				logging.Errorf(" ERROR: The values to be set are invalid.")
				return err
			}

			// Commenting out this call as we do not allow propogating settings unless it is
			// NS server doing the call
			// Un-comment if we need to propogate settings set by query.

			// Do a metakv.Set for the values
			// Set the updates for the given key-value pair for each parameter.
			// This will enable all other query nodes in the cluster to also get
			// updated settings.
			//if err := metakv.Set(QuerySettingsMetaPath, val, rev); err != nil {
			//	logging.Errorf("ERROR: metakv.Set. Error : %v", err)
			//}

			// Callback function defined by the caller where you can
			// manipilate the input values.
			callb(newConfig)
		}
		return nil
	}

	Subscribe(metaKvCallback, QuerySettingsMetaDir, cancelCh)

	return
}

func valConvert(val []byte) (Config, error) {
	nval := value.NewValue(val)

	if nval.Type() != value.OBJECT {
		return nil, fmt.Errorf(" ERROR: Invalid value type. Expected OBJECT, actual %v", nval.Type().String())
	}
	return nval, nil
}
