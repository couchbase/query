//  Copyright 2026-Present Couchbase, Inc.
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

type DropKnowledge struct {
	ddl
	node *algebra.DropKnowledge
}

func NewDropKnowledge(node *algebra.DropKnowledge) *DropKnowledge {
	return &DropKnowledge{
		node: node,
	}
}

func (this *DropKnowledge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitDropKnowledge(this)
}

func (this *DropKnowledge) New() Operator {
	return &DropKnowledge{}
}

func (this *DropKnowledge) Node() *algebra.DropKnowledge {
	return this.node
}

func (this *DropKnowledge) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *DropKnowledge) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "DropKnowledge"}
	this.node.MarshalName(r)

	// invert so the default if not present is to fail if not exists
	r["ifExists"] = !this.node.FailIfNotExists()

	if f != nil {
		f(r)
	}
	return r
}

func (this *DropKnowledge) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string   `json:"#operator"`
		Namespace string   `json:"namespace"`
		Bucket    string   `json:"bucket"`
		Scope     string   `json:"scope"`
		Keyspace  string   `json:"keyspace"`
		Names     []string `json:"names"`
		IfExists  bool     `json:"ifExists"`
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
	// invert IfExists to obtain FailIfNotExists
	this.node = algebra.NewDropKnowledge(path, _unmarshalled.Names, !_unmarshalled.IfExists)
	return nil
}

func (this *DropKnowledge) verify(prepared *Prepared) errors.Error {
	var err errors.Error
	if this.node.Keyspace().Scope() != "" {
		var scope datastore.Scope
		scope, err = datastore.GetScope(this.node.Keyspace().Namespace(), this.node.Keyspace().Bucket(), this.node.Keyspace().Scope())
		if err != nil {
			return errors.NewPlanVerificationError(
				fmt.Sprintf("Scope: %s.%s not found", this.node.Keyspace().Bucket(), this.node.Keyspace().Scope()), err)
		}
		_, err = verifyScope(scope, prepared)
	}
	return err
}
