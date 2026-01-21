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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type PrimaryScan struct {
	legacy
	index            datastore.PrimaryIndex
	indexer          datastore.Indexer
	keyspace         datastore.Keyspace
	term             *algebra.KeyspaceTerm
	limit            expression.Expression
	hasDeltaKeyspace bool
}

func NewPrimaryScan(index datastore.PrimaryIndex, keyspace datastore.Keyspace,
	term *algebra.KeyspaceTerm, limit expression.Expression, hasDeltaKeyspace bool) *PrimaryScan {
	return &PrimaryScan{
		index:            index,
		indexer:          index.Indexer(),
		keyspace:         keyspace,
		term:             term,
		limit:            limit,
		hasDeltaKeyspace: hasDeltaKeyspace,
	}
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) New() Operator {
	return &PrimaryScan{}
}

func (this *PrimaryScan) Index() datastore.PrimaryIndex {
	return this.index
}

func (this *PrimaryScan) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *PrimaryScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *PrimaryScan) Limit() expression.Expression {
	return this.limit
}

func (this *PrimaryScan) HasDeltaKeyspace() bool {
	return this.hasDeltaKeyspace
}

func (this *PrimaryScan) GetIndex() datastore.Index {
	return this.index
}

func (this *PrimaryScan) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *PrimaryScan) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "PrimaryScan"}
	r["index"] = this.index.Name()
	this.term.MarshalKeyspace(r)
	r["using"] = this.index.Type()

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.limit != nil {
		r["limit"] = this.limit.String()
	}

	if this.hasDeltaKeyspace {
		r["has_delta_keyspace"] = this.hasDeltaKeyspace
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *PrimaryScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_                string              `json:"#operator"`
		Index            string              `json:"index"`
		Namespace        string              `json:"namespace"`
		Bucket           string              `json:"bucket"`
		Scope            string              `json:"scope"`
		Keyspace         string              `json:"keyspace"`
		As               string              `json:"as"`
		Using            datastore.IndexType `json:"using"`
		Limit            string              `json:"limit"`
		HasDeltaKeyspace bool                `json:"has_delta_keyspace"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}
	this.hasDeltaKeyspace = _unmarshalled.HasDeltaKeyspace

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.term = algebra.NewKeyspaceTermFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), _unmarshalled.As, nil, nil)
	this.keyspace, err = datastore.GetKeyspace(this.term.Path().Parts()...)
	if err != nil {
		return err
	}

	this.indexer, err = this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := this.indexer.IndexByName(_unmarshalled.Index)
	if err != nil {
		return err
	}

	primary, ok := index.(datastore.PrimaryIndex)
	if !ok {
		return fmt.Errorf("Unable to unmarshal %s as primary index.", _unmarshalled.Index)
	}
	this.index = primary

	planContext := this.PlanContext()
	if planContext != nil {
		if this.limit != nil {
			_, err = planContext.Map(this.limit)
			if err != nil {
				return err
			}
		}
		planContext.addKeyspaceAlias(this.term.Alias())
	}

	return nil
}

func (this *PrimaryScan) verify(prepared *Prepared) errors.Error {
	return verifyIndex(this.index, this.indexer, verifyCoversAndSeqScan(nil, this.keyspace, this.indexer), prepared)
}

func (this *PrimaryScan) keyspaceReferences(prepared *Prepared) {
	prepared.addKeyspaceReference(this.keyspace.QualifiedName())
}
