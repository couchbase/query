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
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type Merge struct {
	readwrite
	keyspace datastore.Keyspace
	ref      *algebra.KeyspaceRef
	key      expression.Expression
	update   Operator
	delete   Operator
	insert   Operator
}

func NewMerge(keyspace datastore.Keyspace, ref *algebra.KeyspaceRef,
	key expression.Expression, update, delete, insert Operator) *Merge {
	return &Merge{
		keyspace: keyspace,
		ref:      ref,
		key:      key,
		update:   update,
		delete:   delete,
		insert:   insert,
	}
}

func (this *Merge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitMerge(this)
}

func (this *Merge) New() Operator {
	return &Merge{}
}

func (this *Merge) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *Merge) KeyspaceRef() *algebra.KeyspaceRef {
	return this.ref
}

func (this *Merge) Key() expression.Expression {
	return this.key
}

func (this *Merge) IsOnKey() bool {
	return this.key != nil
}

func (this *Merge) Update() Operator {
	return this.update
}

func (this *Merge) Delete() Operator {
	return this.delete
}

func (this *Merge) Insert() Operator {
	return this.insert
}

func (this *Merge) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Merge) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Merge"}
	this.ref.MarshalKeyspace(r)

	if this.key != nil {
		r["key"] = expression.NewStringer().Visit(this.key)
	}

	if this.ref.As() != "" {
		r["as"] = this.ref.As()
	}

	if f != nil {
		f(r)
	} else {
		if this.update != nil {
			r["update"] = this.update
		}
		if this.delete != nil {
			r["delete"] = this.delete
		}
		if this.insert != nil {
			r["insert"] = this.insert
		}
	}
	return r
}

func (this *Merge) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string          `json:"#operator"`
		Namespace string          `json:"namespace"`
		Bucket    string          `json:"bucket"`
		Scope     string          `json:"scope"`
		Keyspace  string          `json:"keyspace"`
		As        string          `json:"as"`
		Key       string          `json:"key"`
		Update    json.RawMessage `json:"update"`
		Delete    json.RawMessage `json:"delete"`
		Insert    json.RawMessage `json:"insert"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Keyspace)
	if err != nil {
		return err
	}

	this.ref = algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As)

	if _unmarshalled.Key != "" {
		this.key, err = parser.Parse(_unmarshalled.Key)
		if err != nil {
			return err
		}
	}

	ops := []json.RawMessage{
		_unmarshalled.Update,
		_unmarshalled.Delete,
		_unmarshalled.Insert,
	}

	for i, child := range ops {
		if len(child) == 0 {
			continue
		}

		var op_type struct {
			Operator string `json:"#operator"`
		}

		err = json.Unmarshal(child, &op_type)
		if err != nil {
			return err
		}

		switch i {
		case 0:
			this.update, err = MakeOperator(op_type.Operator, child)
		case 1:
			this.delete, err = MakeOperator(op_type.Operator, child)
		case 2:
			this.insert, err = MakeOperator(op_type.Operator, child)
		}

		if err != nil {
			return err
		}
	}

	return err
}

func (this *Merge) verify(prepared *Prepared) bool {
	var result bool

	this.keyspace, result = verifyKeyspace(this.keyspace, prepared)
	if result && this.insert != nil {
		result = this.insert.verify(prepared)
	}
	if result && this.delete != nil {
		result = this.delete.verify(prepared)
	}
	if result && this.update != nil {
		result = this.update.verify(prepared)
	}
	return result
}
