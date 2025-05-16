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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type CreateSequence struct {
	ddl
	node *algebra.CreateSequence
}

func NewCreateSequence(node *algebra.CreateSequence) *CreateSequence {
	return &CreateSequence{
		node: node,
	}
}

func (this *CreateSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateSequence(this)
}

func (this *CreateSequence) New() Operator {
	return &CreateSequence{}
}

func (this *CreateSequence) Node() *algebra.CreateSequence {
	return this.node
}

func (this *CreateSequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateSequence) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateSequence"}
	this.node.MarshalName(r)

	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()

	r["with"] = this.node.With()

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateSequence) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_           string          `json:"#operator"`
		Namespace   string          `json:"namespace"`
		Bucket      string          `json:"bucket"`
		Scope       string          `json:"scope"`
		Keyspace    string          `json:"keyspace"`
		IfNotExists bool            `json:"ifNotExists"`
		With        json.RawMessage `json:"with"`
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
	// invert IfNotExists to obtain FailIfExists
	this.node = algebra.NewCreateSequence(path, !_unmarshalled.IfNotExists, with)
	return nil
}

func (this *CreateSequence) verify(prepared *Prepared) errors.Error {
	var err errors.Error
	if this.node.Name().Scope() != "" {
		scope, err := datastore.GetScope(this.node.Name().Namespace(), this.node.Name().Bucket(), this.node.Name().Scope())
		if err != nil {
			return errors.NewPlanVerificationError(fmt.Sprintf("Scope: %s.%s not found", this.node.Name().Bucket(), this.node.Name().Scope()), err)
		}
		_, err = verifyScope(scope, prepared)
	}
	return err
}
