//  Copyright 2018-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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
	"query.settings.curl_whitelist": "curl_whitelist",
	"query.settings.curl_allowlist": "curl_allowlist",
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

var lastConfig Config

func SetupSettingsNotifier(callb func(Config), cancelCh chan struct{}) {
	// Callback function that processes the input key value given by metakv.

	metaKvCallback := func(kve metakv.KVEntry) error {
		if kve.Path == QuerySettingsMetaPath {
			logging.Debuga(func() string { return fmt.Sprintf("kve.Value: %s", string(kve.Value)) })
			// To be able to process these settings correctly, convert to a map
			// from string to value.Value.

			// This function will also type check each input value.
			newConfig, err := valConvert(kve.Value)

			if err != nil {
				// Invalid values log this
				logging.Errorf(" ERROR: The values to be set are invalid.")
				return err
			}

			// Commenting out this call as we do not allow propogating settings unless it is
			// NS server doing the call
			// Un-comment if we need to propagate settings set by query.

			// Do a metakv.Set for the values
			// Set the updates for the given key-value pair for each parameter.
			// This will enable all other query nodes in the cluster to also get
			// updated settings.
			//if err := metakv.Set(QuerySettingsMetaPath, val, rev); err != nil {
			//	logging.Errorf("ERROR: metakv.Set. Error : %v", err)
			//}

			// Callback function defined by the caller where you can
			// manipilate the input values.
			if lastConfig == nil || !newConfig.Equals(lastConfig).Truth() {
				logging.Infof("New settings received: %s", string(kve.Value))
				callb(newConfig)
			}
			lastConfig = newConfig
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
