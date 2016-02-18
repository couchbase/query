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

type preparedCache struct {
	sync.RWMutex
	prepareds map[string]*cacheEntry
}

type cacheEntry struct {
	prepared *Prepared
	lastUse  time.Time
	uses     int32
	// FIXME add moving averages, latency
	// This requires an update method to be called from
	// server/http/service_endpoint.go:doStats()
}

const (
	_CACHE_SIZE = 1 << 10
	_MAX_SIZE   = _CACHE_SIZE * 16
)

var cache = &preparedCache{
	prepareds: make(map[string]*cacheEntry, _CACHE_SIZE),
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
			atomic.AddInt32(&rv.uses, 1)
			rv.lastUse = time.Now()
		}
		return rv.prepared
	}
	return nil
}

func (this *preparedCache) add(prepared *Prepared) {
	this.Lock()
	if len(this.prepareds) > _MAX_SIZE {
		this.prepareds = make(map[string]*cacheEntry, _CACHE_SIZE)
	}
	this.prepareds[prepared.Name()] = &cacheEntry{
		prepared: prepared,
	}
	this.Unlock()
}

func (this *preparedCache) peek(prepared *Prepared) bool {
	this.RLock()
	cached := this.prepareds[prepared.Name()]
	this.RUnlock()
	if cached != nil && cached.prepared.Text() != prepared.Text() {
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
		data[i]["name"] = d.prepared.Name()
		data[i]["statement"] = d.prepared.Text()
		data[i]["uses"] = d.uses
		data[i]["lastUse"] = d.lastUse.String()
		data[i]["plan"] = "{ TODO }"
		i++
	}
	return data
}

func (this *preparedCache) size() int {
	this.RLock()
	defer this.RUnlock()
	return len(this.prepareds)
}

func (this *preparedCache) entry(name string) *cacheEntry {
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
} {
	ce := cache.entry(name)
	return struct {
		Uses    int
		LastUse string
		Text    string
	}{
		Uses:    int(ce.uses),
		LastUse: ce.lastUse.String(),
		Text:    ce.prepared.Text(),
	}
}

func AddPrepared(prepared *Prepared) errors.Error {
	if cache.peek(prepared) {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", prepared.Name()))
	}
	cache.add(prepared)
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
		return unmarshalPrepared(prepared_bytes, "")
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

func RecordPreparedMetrics(prepared *Prepared) {
	// TODO
}

func DecodePrepared(prepared_stmt string) (*Prepared, errors.Error) {
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
	prepared, err := unmarshalPrepared(prepared_bytes, prepared_stmt)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	return prepared, nil
}

func unmarshalPrepared(bytes []byte, prepared_stmt string) (*Prepared, errors.Error) {
	prepared := &Prepared{}
	err := prepared.UnmarshalJSON(bytes)
	if err != nil {
		return nil, errors.NewUnrecognizedPreparedError(fmt.Errorf("JSON unmarshalling error: %v", err))
	}
	if prepared_stmt != "" {
		prepared.SetEncodedPlan(prepared_stmt)
	}
	if prepared.Name() != "" {
		cache.add(prepared)
	}
	return prepared, nil
}
