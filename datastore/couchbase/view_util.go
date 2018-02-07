//  Copyright (c) 2013 Couchbase, Inc.
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
	"runtime/debug"
	"sort"
	"strings"

	cb "github.com/couchbase/go-couchbase"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/value"
)

const NETWORK_CHANNEL = "NETWORK"

const TYPE_NULL = 64
const TYPE_BOOLEAN = 96
const TYPE_NUMBER = 128
const TYPE_STRING = 160
const TYPE_ARRAY = 192
const TYPE_OBJECT = 224

var MIN_ID = cb.DocID("")
var MAX_ID = cb.DocID(strings.Repeat(string([]byte{0xff}), 251))

func ViewTotalRows(bucket *cb.Bucket, ddoc string, view string, options map[string]interface{}) (int64, errors.Error) {
	options["limit"] = 0

	logURL, err := bucket.ViewURL(ddoc, view, options)
	if err == nil {
		logging.Debugf("Request View: %v", logURL)
	}
	vres, err := bucket.View(ddoc, view, options)
	if err != nil {
		return 0, errors.NewCbViewsAccessError(err, "View Name"+view)
	}

	return int64(vres.TotalRows), nil
}

func WalkViewInBatches(result chan cb.ViewRow, errs chan errors.Error, stop chan bool, bucket *cb.Bucket,
	ddoc string, view string, isPrimary bool, options map[string]interface{}, batchSize int64, limit int64) {

	if limit != 0 && limit < batchSize {
		batchSize = limit
	}

	defer close(result)
	defer close(errs)

	defer func() {
		r := recover()
		if r != nil {
			logging.Errorf("View Walking Panic: %v\n%s", r, debug.Stack())
			errs <- errors.NewCbViewsAccessError(nil, "Panic In walking view "+view)
		}
	}()

	options["limit"] = batchSize + 1

	numRead := int64(0)
	numSent := int64(0)
	keysSent := map[string]bool{}
	ok := true
	for ok {

		logURL, err := bucket.ViewURL(ddoc, view, options)
		if err == nil {
			logging.Debugf("Request View: %v", logURL)
		}
		vres, err := bucket.View(ddoc, view, options)
		if err != nil {
			errs <- errors.NewCbViewsAccessError(err, "View name "+view)
			return
		}

		for i, row := range vres.Rows {
			// dont process the last row, its just used to see if we
			// need to continue processing
			if int64(i) < batchSize {
				// Send the row if its primary key has not been sent
				if isPrimary || !keysSent[row.ID] {
					select {
					case result <- row:
						numSent += 1
					case <-stop:
						ok = false
						break
					}
				}
				// For non primary views, mark the row's primary key as sent
				if !isPrimary {
					keysSent[row.ID] = true
				}
				numRead += 1
			}
		}

		if (int64(len(vres.Rows)) > batchSize) && (limit == 0 || (limit != 0 && numRead < limit)) {
			// prepare for next run
			skey := vres.Rows[batchSize].Key
			skeydocid := vres.Rows[batchSize].ID
			options["startkey"] = skey
			options["startkey_docid"] = cb.DocID(skeydocid)
		} else {
			// stop
			ok = false
		}
	}
	logging.Debugf("WalkViewInBatches <ud>%s</ud>: %d rows fetched, %d rows sent", view, numRead, numSent)
}

func generateViewOptions(cons datastore.ScanConsistency, span *datastore.Span, isPrimary bool) map[string]interface{} {
	viewOptions := map[string]interface{}{}

	if span != nil {

		logging.Debugf("Scan range. <ud>%v</ud>", span)
		low := span.Range.Low
		high := span.Range.High
		inclusion := span.Range.Inclusion
		if low != nil {
			viewOptions["startkey"] = encodeValuesAsMapKey(low, isPrimary)
			if inclusion == datastore.NEITHER || inclusion == datastore.HIGH {
				viewOptions["startkey_docid"] = MAX_ID
			}
		}

		if high != nil {
			viewOptions["endkey"] = encodeValuesAsMapKey(high, isPrimary)
			if inclusion == datastore.NEITHER || inclusion == datastore.LOW {
				viewOptions["endkey_docid"] = MIN_ID
			}
		}

		if inclusion == datastore.BOTH || inclusion == datastore.HIGH {
			viewOptions["inclusive_end"] = true
		}
	}

	if cons == datastore.SCAN_PLUS || cons == datastore.AT_PLUS {
		viewOptions["stale"] = "false"
	} else if cons == datastore.UNBOUNDED {
		viewOptions["stale"] = "ok"
	}

	return viewOptions
}

func encodeValuesAsMapKey(keys value.Values, isPrimary bool) interface{} {
	if isPrimary {
		if len(keys) > 1 {
			panic(fmt.Sprintf("Key value for a primary index should be length 1, found: %d", len(keys)))
		} else {
			return encodeValue(keys[0].Actual())
		}
	}
	rv := make([]interface{}, len(keys))
	for i, lv := range keys {
		val := lv.Actual()
		rv[i] = encodeValue(val)
	}
	return rv
}

func encodeValue(val interface{}) interface{} {
	switch val := val.(type) {
	case nil:
		return []interface{}{TYPE_NULL}
	case bool:
		return []interface{}{TYPE_BOOLEAN, val}
	case float64:
		return []interface{}{TYPE_NUMBER, val}
	case string:
		return []interface{}{TYPE_STRING, encodeStringAsNumericArray(val)}
	case []interface{}:
		return []interface{}{TYPE_ARRAY, val}
	case map[string]interface{}:
		return []interface{}{TYPE_OBJECT, encodeObjectAsCompoundArray(val)}
	default:
		panic(fmt.Sprintf("Unable to encode type %T to map key", val))
	}
}

func encodeStringAsNumericArray(str string) []float64 {
	rv := make([]float64, len(str))
	for i, rune := range str {
		rv[i] = float64(rune)
	}
	return rv
}

func decodeNumericArrayAsString(na []interface{}) (string, error) {
	rv := ""
	for _, num := range na {
		switch num := num.(type) {
		case float64:
			rv = rv + string(rune(num))
		default:
			return "", fmt.Errorf("numeric array contained non-number")
		}
	}
	return rv, nil
}

func encodeObjectAsCompoundArray(obj map[string]interface{}) []interface{} {
	keys := make([]string, len(obj))
	counter := 0
	for k, _ := range obj {
		keys[counter] = k
		counter++
	}
	sort.Strings(keys)
	vals := make([]interface{}, len(obj))
	for i, key := range keys {
		vals[i] = encodeValue(obj[key])
	}
	return []interface{}{keys, vals}
}

func decodeCompoundArrayAsObject(ca []interface{}) (map[string]interface{}, error) {
	rv := map[string]interface{}{}

	if len(ca) != 2 {
		return nil, fmt.Errorf("Incorrectly formatted compound array object")
	}

	key_array, ok := ca[0].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Key array is not an array but type %T", ca[0], ca[0])
	}

	val_array, ok := ca[1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("Value array is not an array but type %T", val_array)
	}

	for i, key := range key_array {

		key := key.(string)
		val := val_array[i]
		decodedVal, err := convertCouchbaseViewKeyEntryToValue(val)
		if err != nil {
			return nil, err
		}
		rv[key] = decodedVal

	}
	return rv, nil
}

func convertCouchbaseViewKeyToLookupValue(key interface{}) (value.Values, error) {

	switch key := key.(type) {
	case []interface{}:
		// top-level key MUST be an array
		rv := make(value.Values, len(key))
		for i, keyEntry := range key {
			val, err := convertCouchbaseViewKeyEntryToValue(keyEntry)
			if err != nil {
				return nil, err
			}
			rv[i] = val
		}
		return rv, nil
	}
	return nil, fmt.Errorf("Couchbase view key top-level MUST be an array")
}

func convertCouchbaseViewKeyEntryToValue(keyEntry interface{}) (value.Value, error) {

	switch keyEntry := keyEntry.(type) {
	case []interface{}:
		keyEntryType, ok := keyEntry[0].(float64)
		if !ok {
			return nil, fmt.Errorf("Key entry type must be number")
		}
		switch keyEntryType {
		case TYPE_NULL:
			return value.NewValue(nil), nil
		case TYPE_BOOLEAN, TYPE_NUMBER, TYPE_ARRAY:
			return value.NewValue(keyEntry[1]), nil
		case TYPE_STRING:
			keyStringValue, ok := keyEntry[1].([]interface{})
			if !ok {
				return nil, fmt.Errorf("key entry type string value must be array")
			}
			decodedString, err := decodeNumericArrayAsString(keyStringValue)
			if err != nil {
				return nil, err
			}
			return value.NewValue(decodedString), nil
		case TYPE_OBJECT:
			keyObjectValue, ok := keyEntry[1].([]interface{})
			if !ok {
				return nil, fmt.Errorf("key entry type object value must be array")
			}
			decodedObject, err := decodeCompoundArrayAsObject(keyObjectValue)
			if err != nil {
				return nil, err
			}
			return value.NewValue(decodedObject), nil
		}
		return nil, fmt.Errorf("Unkown type of key entry")
	}
	return nil, fmt.Errorf("Key entries top-level MUST be an array")
}
