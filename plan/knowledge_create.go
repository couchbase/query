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
	"github.com/couchbase/query/expression/parser"
)

type CreateKnowledge struct {
	ddl
	node *algebra.CreateKnowledge
}

func NewCreateKnowledge(node *algebra.CreateKnowledge) *CreateKnowledge {
	return &CreateKnowledge{
		node: node,
	}
}

func (this *CreateKnowledge) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateKnowledge(this)
}

func (this *CreateKnowledge) New() Operator {
	return &CreateKnowledge{}
}

func (this *CreateKnowledge) Node() *algebra.CreateKnowledge {
	return this.node
}

func (this *CreateKnowledge) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateKnowledge) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateKnowledge"}
	this.node.MarshalName(r)
	r["value"] = this.node.Value()
	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateKnowledge) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		KnowName  string `json:"know_name"`
		Value     string `json:"value"`
		Bulk      bool   `json:"bulk"`
		Replace   bool   `json:"replace"`
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

	value, err := parser.Parse(_unmarshalled.Value)
	if err != nil {
		return err
	}

	path := algebra.NewPathLong(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope, _unmarshalled.Keyspace)
	if _unmarshalled.Bulk {
		this.node = algebra.NewCreateKnowledgeBulk("", path, value, _unmarshalled.Replace)
	} else {
		this.node = algebra.NewCreateKnowledge(_unmarshalled.KnowName, path, value, _unmarshalled.Replace)
	}
	return nil
}

func (this *CreateKnowledge) verify(prepared *Prepared) errors.Error {
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
