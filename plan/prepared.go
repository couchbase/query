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
	signature value.Value
	name      string
}

func NewPrepared(operator Operator, signature value.Value) *Prepared {
	return &Prepared{
		Operator:  operator,
		signature: signature,
	}
}

func (this *Prepared) MarshalJSON() ([]byte, error) {
	r := make(map[string]interface{}, 3)
	r["operator"] = this.Operator
	r["signature"] = this.signature
	r["name"] = this.name

	return json.Marshal(r)
}

func (this *Prepared) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		Operator  json.RawMessage `json:"operator"`
		Signature json.RawMessage `json:"signature"`
		Name      string          `json:"name"`
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

func (this *preparedCache) add(prepared *Prepared) {
	this.Lock()
	if len(this.prepareds) > _MAX_SIZE {
		this.prepareds = make(map[string]*Prepared, _CACHE_SIZE)
	}
	this.prepareds[prepared.Name()] = prepared
	this.Unlock()
}

func (this *preparedCache) peek(name string) bool {
	this.RLock()
	_, has_name := this.prepareds[name]
	this.RUnlock()
	return has_name
}

func AddPrepared(prepared *Prepared) errors.Error {
	if cache.peek(prepared.Name()) {
		return errors.NewPreparedNameError("duplicate name")
	}
	cache.add(prepared)
	return nil
}

var errBadFormat = fmt.Errorf("unable to convert to prepared statment.")

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
		return nil, errors.NewUnrecognizedPreparedError(errBadFormat)
	}
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
	prepared, err := unmarshalPrepared(prepared_bytes)
	if err != nil {
		return nil, errors.NewPreparedDecodingError(err)
	}
	return prepared, nil
}

func unmarshalPrepared(bytes []byte) (*Prepared, errors.Error) {
	prepared := &Prepared{}
	err := prepared.UnmarshalJSON(bytes)
	if err != nil {
		return nil, errors.NewUnrecognizedPreparedError(errBadFormat)
	}
	if prepared.Name() != "" {
		cache.add(prepared)
	}
	return prepared, nil
}
