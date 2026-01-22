//  Copyright 2014-Present Couchbase, Inc.
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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

// Build indexes
type BuildIndexes struct {
	ddl
	keyspace datastore.Keyspace
	node     *algebra.BuildIndexes
}

func NewBuildIndexes(keyspace datastore.Keyspace, node *algebra.BuildIndexes) *BuildIndexes {
	return &BuildIndexes{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *BuildIndexes) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitBuildIndexes(this)
}

func (this *BuildIndexes) New() Operator {
	return &BuildIndexes{}
}

func (this *BuildIndexes) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *BuildIndexes) Node() *algebra.BuildIndexes {
	return this.node
}

func (this *BuildIndexes) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *BuildIndexes) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "BuildIndexes"}
	this.node.Keyspace().MarshalKeyspace(r)
	r["using"] = this.node.Using()

	if len(this.node.Names()) > 0 {
		r["indexes"] = this.node.Names()
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *BuildIndexes) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		Namespace string              `json:"namespace"`
		Bucket    string              `json:"bucket"`
		Scope     string              `json:"scope"`
		Keyspace  string              `json:"keyspace"`
		Using     datastore.IndexType `json:"using"`
		Indexes   []string            `json:"indexes"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	exprs := make(expression.Expressions, 0, len(_unmarshalled.Indexes))
	for _, s := range _unmarshalled.Indexes {
		expr, err := parser.Parse(s)
		if err != nil {
			return err
		}
		exprs = append(exprs, expr)
	}

	this.node = algebra.NewBuildIndexes(ksref, _unmarshalled.Using, exprs...)

	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	return err
}

func (this *BuildIndexes) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.keyspace, err = verifyKeyspace(this.keyspace, prepared)
	return err
}

func (this *BuildIndexes) keyspaceReferences(prepared *Prepared) {
	prepared.addKeyspaceReference(this.keyspace)
}
