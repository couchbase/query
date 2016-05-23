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
	"sync"
	"time"

	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type Prepared struct {
	Operator
	signature    value.Value
	name         string
	encoded_plan string
	text         string
}

func NewPrepared(operator Operator, signature value.Value) *Prepared {
	return &Prepared{
		Operator:  operator,
		signature: signature,
	}
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 5)
	r["operator"] = this.Operator
	r["signature"] = this.signature
	r["name"] = this.name
	r["encoded_plan"] = this.encoded_plan
	r["text"] = this.text

	return json.Marshal(r)
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operator    json.RawMessage `json:"operator"`
		Signature   json.RawMessage `json:"signature"`
		Name        string          `json:"name"`
		EncodedPlan string          `json:"encoded_plan"`
		Text        string          `json:"text"`
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
	sync.RWMutex
	prepareds map[string]*CacheEntry
}

type CacheEntry struct {
	Prepared    *Prepared
	LastUse     time.Time
	Uses        int32
	ServiceTime atomic.AlignedUint64
	RequestTime atomic.AlignedUint64
	// FIXME add moving averages, latency
	// This requires the use of metrics
}

const (
	_CACHE_SIZE = 1 << 10
	_MAX_SIZE   = _CACHE_SIZE * 16
)

var cache = &preparedCache{
	prepareds: make(map[string]*CacheEntry, _CACHE_SIZE),
}

func (this *preparedCache) get(name value.Value, track bool) *Prepared {
	if name.Type() != value.STRING || !name.Truth() {
		return nil
	}
	this.RLock()
	defer this.RUnlock()
	rv := this.prepareds[name.Actual().(string)]
	if rv != nil {
		if track {
			atomic.AddInt32(&rv.Uses, 1)
			rv.LastUse = time.Now()
		}
		return rv.Prepared
	}
	return nil
}

func (this *preparedCache) add(prepared *Prepared, process func(*CacheEntry) bool) {
	this.Lock()
	defer this.Unlock()

	ce, ok := this.prepareds[prepared.Name()]

	// build one if missing
	if !ok {
		ce = &CacheEntry{
			Prepared: prepared,
		}
	}
	if process != nil {
		if cont := process(ce); !cont {
			return
		}
	}

	// amend existing one if cleared to proceed
	if ok {
		ce.Prepared = prepared
	}
	if len(this.prepareds) > _MAX_SIZE {
		this.prepareds = make(map[string]*CacheEntry, _CACHE_SIZE)
	}
	this.prepareds[prepared.Name()] = ce
}

func (this *preparedCache) peek(prepared *Prepared) bool {
	this.RLock()
	cached := this.prepareds[prepared.Name()]
	this.RUnlock()
	if cached != nil && cached.Prepared.Text() != prepared.Text() {
		return true
	}
	return false
}

func (this *preparedCache) peekName(name string) bool {
	this.RLock()
	cached := this.prepareds[name]
	this.RUnlock()
	return cached != nil
}

func (this *preparedCache) snapshot() []map[string]interface{} {
	this.RLock()
	defer this.RUnlock()
	data := make([]map[string]interface{}, len(this.prepareds))
	i := 0
	for _, d := range this.prepareds {
		data[i] = map[string]interface{}{}
		data[i]["name"] = d.Prepared.Name()
		data[i]["encoded_plan"] = d.Prepared.EncodedPlan()
		data[i]["statement"] = d.Prepared.Text()
		data[i]["uses"] = d.Uses
		if d.Uses > 0 {
			data[i]["lastUse"] = d.LastUse.String()
		}
		i++
	}
	return data
}

func (this *preparedCache) size() int {
	this.RLock()
	defer this.RUnlock()
	return len(this.prepareds)
}

func (this *preparedCache) entry(name string) *CacheEntry {
	this.RLock()
	defer this.RUnlock()
	return this.prepareds[name]
}

func (this *preparedCache) remove(name string) {
	this.Lock()
	defer this.Unlock()
	delete(this.prepareds, name)
}

func (this *preparedCache) names() []string {
	i := 0
	this.RLock()
	defer this.RUnlock()
	n := make([]string, len(this.prepareds))
	for k := range this.prepareds {
		n[i] = k
		i++
	}
	return n
}

func SnapshotPrepared() []map[string]interface{} {
	return cache.snapshot()
}

func CountPrepareds() int {
	return cache.size()
}

func NamePrepareds() []string {
	return cache.names()
}

func PreparedEntry(name string) struct {
	Uses    int
	LastUse string
	Text    string
	Plan    string
} {
	ce := cache.entry(name)
	return struct {
		Uses    int
		LastUse string
		Text    string
		Plan    string
	}{
		Uses:    int(ce.Uses),
		LastUse: ce.LastUse.String(),
		Text:    ce.Prepared.Text(),
		Plan:    ce.Prepared.EncodedPlan(),
	}
}

func PreparedDo(name string, f func(*CacheEntry)) {
	cache.RLock()
	defer cache.RUnlock()
	ce := cache.prepareds[name]
	if ce != nil {
		f(ce)
	}
}

func AddPrepared(prepared *Prepared) errors.Error {
	if cache.peek(prepared) {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", prepared.Name()))
	}
	cache.add(prepared, nil)
	return nil
}

func DeletePrepared(name string) errors.Error {
	if !cache.peekName(name) {
		return errors.NewNoSuchPreparedError(name)
	}
	cache.remove(name)
	return nil
}

var errBadFormat = fmt.Errorf("unable to convert to prepared statment.")

func doGetPrepared(prepared_stmt value.Value, track bool) (*Prepared, errors.Error) {
	switch prepared_stmt.Type() {
	case value.STRING:
		prepared := cache.get(prepared_stmt, track)
		if prepared == nil {
			return nil, errors.NewNoSuchPreparedError(prepared_stmt.Actual().(string))
		}
		return prepared, nil
	case value.OBJECT:
		name_value, has_name := prepared_stmt.Field("name")
		if has_name {
			if prepared := cache.get(name_value, track); prepared != nil {
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
	cache.RLock()
	defer cache.RUnlock()
	ce := cache.prepareds[name]
	if ce != nil {
		atomic.AddUint64(&ce.ServiceTime, uint64(serviceTime))
		atomic.AddUint64(&ce.RequestTime, uint64(requestTime))
	}
}

func DecodePrepared(prepared_name string, prepared_stmt string, track bool) (*Prepared, errors.Error) {
	var cacheErr errors.Error = nil

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
	cache.add(prepared,
		func(oldEntry *CacheEntry) bool {

			// MB-19509: if the entry exists already, the new plan must
			// also be for the same statement as we have in the cache
			if oldEntry.Prepared != prepared &&
				oldEntry.Prepared.text != prepared.text {
				cacheErr = errors.NewPreparedEncodingMismatchError(prepared_name)
				return false
			}

			// track the entry if required, whether we amend the plan or
			// not, as at the end of the statement we will record the
			// metrics anyway
			if track {
				atomic.AddInt32(&oldEntry.Uses, 1)
				oldEntry.LastUse = time.Now()
			}

			// MB-19659: this is where we decide plan conflict.
			// the current behaviour is to always use the new plan
			// and amend the cache
			// This is still to be finalized
			return true
		})
	if cacheErr == nil {
		return prepared, nil
	} else {
		return nil, cacheErr
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
