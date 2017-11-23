//  Copyright (c) 2017 Couchbase, Inc.
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
	"fmt"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type PrimaryScan3 struct {
	readonly
	index      datastore.PrimaryIndex3
	keyspace   datastore.Keyspace
	term       *algebra.KeyspaceTerm
	projection *IndexProjection
	orderTerms IndexKeyOrders
	offset     expression.Expression
	limit      expression.Expression
}

func NewPrimaryScan3(index datastore.PrimaryIndex3, keyspace datastore.Keyspace,
	term *algebra.KeyspaceTerm, offset, limit expression.Expression,
	projection *IndexProjection, orderTerms IndexKeyOrders) *PrimaryScan3 {
	return &PrimaryScan3{
		index:      index,
		keyspace:   keyspace,
		term:       term,
		projection: projection,
		orderTerms: orderTerms,
		offset:     offset,
		limit:      limit,
	}
}

func (this *PrimaryScan3) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan3(this)
}

func (this *PrimaryScan3) New() Operator {
	return &PrimaryScan3{}
}

func (this *PrimaryScan3) Index() datastore.PrimaryIndex3 {
	return this.index
}

func (this *PrimaryScan3) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *PrimaryScan3) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *PrimaryScan3) Projection() *IndexProjection {
	return this.projection
}

func (this *PrimaryScan3) OrderTerms() IndexKeyOrders {
	return this.orderTerms
}

func (this *PrimaryScan3) Offset() expression.Expression {
	return this.offset
}

func (this *PrimaryScan3) Limit() expression.Expression {
	return this.limit
}

func (this *PrimaryScan3) SetLimit(limit expression.Expression) {
	this.limit = limit
}

func (this *PrimaryScan3) SetOffset(offset expression.Expression) {
	this.offset = offset
}

func (this *PrimaryScan3) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *PrimaryScan3) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "PrimaryScan3"}
	r["index"] = this.index.Name()
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["using"] = this.index.Type()

	if this.term.As() != "" {
		r["as"] = this.term.As()
	}

	if this.projection != nil {
		r["index_projection"] = this.projection
	}

	if len(this.orderTerms) > 0 {
		r["index_order"] = this.orderTerms
	}

	if this.offset != nil {
		r["offset"] = expression.NewStringer().Visit(this.offset)
	}

	if this.limit != nil {
		r["limit"] = expression.NewStringer().Visit(this.limit)
	}

	if f != nil {
		f(r)
	}
	return r
}

func (this *PrimaryScan3) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_          string              `json:"#operator"`
		Index      string              `json:"index"`
		Names      string              `json:"namespace"`
		Keys       string              `json:"keyspace"`
		As         string              `json:"as"`
		Using      datastore.IndexType `json:"using"`
		Projection *IndexProjection    `json:"index_projection"`
		OrderTerms IndexKeyOrders      `json:"index_order"`
		Offset     string              `json:"offset"`
		Limit      string              `json:"limit"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	this.projection = _unmarshalled.Projection
	this.orderTerms = _unmarshalled.OrderTerms

	if _unmarshalled.Offset != "" {
		this.offset, err = parser.Parse(_unmarshalled.Offset)
		if err != nil {
			return err
		}
	}

	if _unmarshalled.Limit != "" {
		this.limit, err = parser.Parse(_unmarshalled.Limit)
		if err != nil {
			return err
		}
	}

	this.keyspace, err = datastore.GetKeyspace(_unmarshalled.Names, _unmarshalled.Keys)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(_unmarshalled.Names, _unmarshalled.Keys, _unmarshalled.As, nil, nil)
	indexer, err := this.keyspace.Indexer(_unmarshalled.Using)
	if err != nil {
		return err
	}

	index, err := indexer.IndexByName(_unmarshalled.Index)
	if err != nil {
		return err
	}

	if primary, ok := index.(datastore.PrimaryIndex3); ok {
		this.index = primary
		return nil
	}

	return fmt.Errorf("Unable to find Primary Index3 for %v", index.Name())
}
