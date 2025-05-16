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
)

type DropSequence struct {
	ddl
	node *algebra.DropSequence
}

func NewDropSequence(node *algebra.DropSequence) *DropSequence {
	return &DropSequence{
		node: node,
	}
}

func (this *DropSequence) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropSequence(this)
}

func (this *DropSequence) New() Operator {
	return &DropSequence{}
}

func (this *DropSequence) Node() *algebra.DropSequence {
	return this.node
}

func (this *DropSequence) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropSequence) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropSequence"}
	this.node.MarshalName(r)

	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()

	if f != nil {
		f(r)
	}
	return r
}

func (this *DropSequence) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		IfExists  bool   `json:"ifExists"`
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

	path := algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Keyspace)
	// invert IfExists to obtain FailIfExists
	this.node = algebra.NewDropSequence(path, !_unmarshalled.IfExists)
	return nil
}

func (this *DropSequence) verify(prepared *Prepared) errors.Error {
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
