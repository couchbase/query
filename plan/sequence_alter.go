//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/value"
)

type AlterSequence struct {
	ddl
	node *algebra.AlterSequence
}

func NewAlterSequence(node *algebra.AlterSequence) *AlterSequence {
	return &AlterSequence{
		node: node,
	}
}

func (this *AlterSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAlterSequence(this)
}

func (this *AlterSequence) New() Operator {
	return &AlterSequence{}
}

func (this *AlterSequence) Node() *algebra.AlterSequence {
	return this.node
}

func (this *AlterSequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *AlterSequence) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "AlterSequence"}
	this.node.MarshalName(r)

	r["with"] = this.node.With()

	if f != nil {
		f(r)
	}
	return r
}

func (this *AlterSequence) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string          `json:"#operator"`
		Namespace string          `json:"namespace"`
		Bucket    string          `json:"bucket"`
		Scope     string          `json:"scope"`
		Keyspace  string          `json:"keyspace"`
		With      json.RawMessage `json:"with"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	if _unmarshalled.Scope != "" {
		_, err = datastore.GetScope(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope)
		if err != nil {
			return err
		}
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}
	path := algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Keyspace)
	this.node = algebra.NewAlterSequence(path, with)
	return nil
}

func (this *AlterSequence) verify(prepared *Prepared) bool {
	res := true
	if this.node.Name().Scope() != "" {
		scope, err := datastore.GetScope(this.node.Name().Namespace(), this.node.Name().Bucket(), this.node.Name().Scope())
		if err != nil {
			return false
		}
		_, res = verifyScope(scope, prepared)
	}
	return res
}
