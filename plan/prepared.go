//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

type Prepared struct {
	Operator
	signature    value.Value
	name         string
	encoded_plan string
	text         string
	reqType      string
}

func NewPrepared(operator Operator, signature value.Value) *Prepared {
	return &Prepared{
		Operator:  operator,
		signature: signature,
	}
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Prepared) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := make(map[string]interface{}, 5)
	r["operator"] = this.Operator
	r["signature"] = this.signature
	r["name"] = this.name
	r["encoded_plan"] = this.encoded_plan
	r["text"] = this.text

	if f != nil {
		f(r)
	}
	return r
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operator    json.RawMessage `json:"operator"`
		Signature   json.RawMessage `json:"signature"`
		Name        string          `json:"name"`
		EncodedPlan string          `json:"encoded_plan"`
		Text        string          `json:"text"`
		ReqType     string          `json:"reqType"`
	}

	var op_type struct {
		Operator string `json:"#operator"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	err = json.Unmarshal(_unmarshalled.Operator, &op_type)
	if err != nil {
		return err
	}

	this.signature = value.NewValue(_unmarshalled.Signature)
	this.name = _unmarshalled.Name
	this.encoded_plan = _unmarshalled.EncodedPlan
	this.text = _unmarshalled.Text
	this.reqType = _unmarshalled.ReqType
	this.Operator, err = MakeOperator(op_type.Operator, _unmarshalled.Operator)

	return err
}

func (this *Prepared) Signature() value.Value {
	return this.signature
}

func (this *Prepared) Name() string {
	return this.name
}

func (this *Prepared) SetName(name string) {
	this.name = name
}

func (this *Prepared) Text() string {
	return this.text
}

func (this *Prepared) SetText(text string) {
	this.text = text
}

func (this *Prepared) Type() string {
	return this.reqType
}

func (this *Prepared) SetType(reqType string) {
	this.reqType = reqType
}

func (this *Prepared) EncodedPlan() string {
	return this.encoded_plan
}

func (this *Prepared) SetEncodedPlan(encoded_plan string) {
	this.encoded_plan = encoded_plan
}

func (this *Prepared) MismatchingEncodedPlan(encoded_plan string) bool {
	return this.encoded_plan != encoded_plan
}

type preparedCache struct {
	cache *util.GenCache
}

type CacheEntry struct {
	Prepared       *Prepared
	LastUse        time.Time
	Uses           int32
	ServiceTime    atomic.AlignedUint64
	RequestTime    atomic.AlignedUint64
	MinServiceTime atomic.AlignedUint64
	MinRequestTime atomic.AlignedUint64
	MaxServiceTime atomic.AlignedUint64
	MaxRequestTime atomic.AlignedUint64
	// FIXME add moving averages, latency
	// This requires the use of metrics
}

var prepareds = &preparedCache{}

// init prepareds cache

func PreparedsInit(limit int) {
	prepareds.cache = util.NewGenCache(limit)
}

// configure prepareds cache

func PreparedsLimit() int {
	return prepareds.cache.Limit()
}

func PreparedsSetLimit(limit int) {
	prepareds.cache.SetLimit(limit)
}

func (this *preparedCache) get(name value.Value, track bool) *Prepared {
	var cv interface{}

	if name.Type() != value.STRING || !name.Truth() {
		return nil
	}

	n := name.Actual().(string)
	if track {
		cv = prepareds.cache.Use(n, nil)
	} else {
		cv = prepareds.cache.Get(n, nil)
	}
	rv, ok := cv.(*CacheEntry)
	if ok {
		if track {
			atomic.AddInt32(&rv.Uses, 1)

			// this is not exactly accurate, but since the MRU queue is
			// managed properly, we'd rather be inaccurate and make the
			// change outside of the lock than take a performance hit
			rv.LastUse = time.Now()
		}
		return rv.Prepared
	}
	return nil
}

func (this *preparedCache) add(prepared *Prepared, process func(*CacheEntry) bool) {

	// prepare a new entry, if statement does not exist
	ce := &CacheEntry{
		Prepared:       prepared,
		MinServiceTime: math.MaxUint64,
		MinRequestTime: math.MaxUint64,
	}
	prepareds.cache.Add(ce, prepared.Name(), func(entry interface{}) util.Operation {
		var op util.Operation = util.AMEND
		var cont bool = true

		// check existing entry, amend if all good, ignore otherwise
		oldEntry := entry.(*CacheEntry)
		if process != nil {
			cont = process(oldEntry)
		}
		if cont {
			oldEntry.Prepared = prepared
		} else {
			op = util.IGNORE
		}
		return op
	})
}

func CountPrepareds() int {
	return prepareds.cache.Size()
}

func NamePrepareds() []string {
	return prepareds.cache.Names()
}

func PreparedsForeach(nonBlocking func(string, *CacheEntry) bool,
	blocking func() bool) {
	dummyF := func(id string, r interface{}) bool {
		return nonBlocking(id, r.(*CacheEntry))
	}
	prepareds.cache.ForEach(dummyF, blocking)
}

func PreparedDo(name string, f func(*CacheEntry)) {
	var process func(interface{}) = nil

	if f != nil {
		process = func(entry interface{}) {
			ce := entry.(*CacheEntry)
			f(ce)
		}
	}
	_ = prepareds.cache.Get(name, process)
}

func AddPrepared(prepared *Prepared) errors.Error {
	added := true

	prepareds.add(prepared, func(ce *CacheEntry) bool {
		if ce.Prepared.Text() != prepared.Text() {
			added = false
		}
		return added
	})
	if !added {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", prepared.Name()))
	} else {
		return nil
	}
}

func DeletePrepared(name string) errors.Error {
	if prepareds.cache.Delete(name, nil) {
		return nil
	}
	return errors.NewNoSuchPreparedError(name)
}

var errBadFormat = fmt.Errorf("unable to convert to prepared statment.")

func doGetPrepared(prepared_stmt value.Value, track bool) (*Prepared, errors.Error) {
	switch prepared_stmt.Type() {
	case value.STRING:
		prepared := prepareds.get(prepared_stmt, track)
		if prepared == nil {
			return nil, errors.NewNoSuchPreparedError(prepared_stmt.Actual().(string))
		}
		return prepared, nil
	case value.OBJECT:
		name_value, has_name := prepared_stmt.Field("name")
		if has_name {
			if prepared := prepareds.get(name_value, track); prepared != nil {
				return prepared, nil
			}
		}
		prepared_bytes, err := prepared_stmt.MarshalJSON()
		if err != nil {
			return nil, errors.NewUnrecognizedPreparedError(err)
		}
		return unmarshalPrepared(prepared_bytes)
	default:
		return nil, errors.NewUnrecognizedPreparedError(fmt.Errorf("Invalid prepared stmt %v", prepared_stmt))
	}
}

func GetPrepared(prepared_stmt value.Value) (*Prepared, errors.Error) {
	return doGetPrepared(prepared_stmt, false)
}

func TrackPrepared(prepared_stmt value.Value) (*Prepared, errors.Error) {
	return doGetPrepared(prepared_stmt, true)
}

func RecordPreparedMetrics(prepared *Prepared, requestTime, serviceTime time.Duration) {
	if prepared == nil {
		return
	}
	name := prepared.Name()
	if name == "" {
		return
	}

	// cache get had already moved this entry to the top of the LRU
	// no need to do it again
	_ = prepareds.cache.Get(name, func(entry interface{}) {
		ce := entry.(*CacheEntry)
		atomic.AddUint64(&ce.ServiceTime, uint64(serviceTime))
		util.TestAndSetUint64(&ce.MinServiceTime, uint64(serviceTime),
			func(old, new uint64) bool { return old > new }, 0)
		util.TestAndSetUint64(&ce.MaxServiceTime, uint64(serviceTime),
			func(old, new uint64) bool { return old < new }, 0)
		atomic.AddUint64(&ce.RequestTime, uint64(requestTime))
		util.TestAndSetUint64(&ce.MinRequestTime, uint64(requestTime),
			func(old, new uint64) bool { return old > new }, 0)
		util.TestAndSetUint64(&ce.MaxRequestTime, uint64(requestTime),
			func(old, new uint64) bool { return old < new }, 0)
	})
}

func DecodePrepared(prepared_name string, prepared_stmt string, track bool) (*Prepared, errors.Error) {
	added := true

	decoded, err := base64.StdEncoding.DecodeString(prepared_stmt)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	var buf bytes.Buffer
	buf.Write(decoded)
	reader, err := gzip.NewReader(&buf)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	prepared_bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	prepared, err := unmarshalPrepared(prepared_bytes)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}

	prepared.SetEncodedPlan(prepared_stmt)

	// MB-19509 we now have to check that the encoded plan matches
	// the prepared statement named in the rest API
	if prepared.Name() != "" && prepared_name != "" &&
		prepared_name != prepared.Name() {
		return nil, errors.NewEncodingNameMismatchError(prepared_name)
	}

	if prepared.Name() == "" {
		return prepared, nil
	}

	when := time.Now()
	prepareds.add(prepared,
		func(oldEntry *CacheEntry) bool {

			// MB-19509: if the entry exists already, the new plan must
			// also be for the same statement as we have in the cache
			if oldEntry.Prepared != prepared &&
				oldEntry.Prepared.text != prepared.text {
				added = false
				return added
			}

			// track the entry if required, whether we amend the plan or
			// not, as at the end of the statement we will record the
			// metrics anyway
			if track {
				atomic.AddInt32(&oldEntry.Uses, 1)
				oldEntry.LastUse = when
			}

			// MB-19659: this is where we decide plan conflict.
			// the current behaviour is to always use the new plan
			// and amend the cache
			// This is still to be finalized
			return added
		})

	if added {
		return prepared, nil
	} else {
		return nil, errors.NewPreparedEncodingMismatchError(prepared_name)
	}
}

func unmarshalPrepared(bytes []byte) (*Prepared, errors.Error) {
	prepared := &Prepared{}
	err := prepared.UnmarshalJSON(bytes)
	if err != nil {
		return nil, errors.NewUnrecognizedPreparedError(fmt.Errorf("JSON unmarshalling error: %v", err))
	}
	return prepared, nil
}
