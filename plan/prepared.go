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
	prepareds map[string]*Prepared
}

const (
	_CACHE_SIZE = 1 << 10
	_MAX_SIZE   = _CACHE_SIZE * 16
)

var cache = &preparedCache{
	prepareds: make(map[string]*Prepared, _CACHE_SIZE),
}

func (this *preparedCache) get(name value.Value) *Prepared {
	if name.Type() != value.STRING || !name.Truth() {
		return nil
	}
	this.RLock()
	rv := this.prepareds[name.Actual().(string)]
	this.RUnlock()
	return rv
}


func (this *preparedCache) add(prepared *Prepared, process func(*Prepared) bool) {
	this.Lock()
	defer this.Unlock()

	pr, ok := this.prepareds[prepared.Name()]

	// build one if missing
	if !ok {
		pr = prepared
	}
	if process != nil {
		if cont := process(pr); !cont {
			return
		}
	}

	if len(this.prepareds) > _MAX_SIZE {
		this.prepareds = make(map[string]*Prepared, _CACHE_SIZE)
	}
	this.prepareds[prepared.Name()] = pr
}

func (this *preparedCache) peek(prepared *Prepared) bool {
	this.RLock()
	cached := this.prepareds[prepared.Name()]
	this.RUnlock()
	if cached != nil && cached.Text() != prepared.Text() {
		return true
	}
	return false
}

func AddPrepared(prepared *Prepared) errors.Error {
	if cache.peek(prepared) {
		return errors.NewPreparedNameError(
			fmt.Sprintf("duplicate name: %s", prepared.Name()))
	}
	cache.add(prepared, nil)
	return nil
}

func GetPrepared(prepared_stmt value.Value) (*Prepared, errors.Error) {
	switch prepared_stmt.Type() {
	case value.STRING:
		prepared := cache.get(prepared_stmt)
		if prepared == nil {
			return nil, errors.NewNoSuchPreparedError(prepared_stmt.Actual().(string))
		}
		return prepared, nil
	case value.OBJECT:
		name_value, has_name := prepared_stmt.Field("name")
		if has_name {
			if prepared := cache.get(name_value); prepared != nil {
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

func DecodePrepared(prepared_name string, prepared_stmt string) (*Prepared, errors.Error) {
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
		func(oldEntry *Prepared) bool {

			// MB-19509: if the entry exists already, the new plan must
			// also be for the same statement as we have in the cache
			if oldEntry != prepared &&
				oldEntry.text != prepared.text {
				cacheErr = errors.NewPreparedEncodingMismatchError(prepared_name)
				return false
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
