//  Copyright (c) 2016 Couchbase, Inc.
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
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/expression/parser"
)

type IndexCountScan struct {
	readonly
	index  datastore.CountIndex
	term   *algebra.KeyspaceTerm
	spans  Spans
	covers expression.Covers
}

func NewIndexCountScan(index datastore.CountIndex, term *algebra.KeyspaceTerm,
	spans Spans, covers expression.Covers) *IndexCountScan {
	return &IndexCountScan{
		index:  index,
		term:   term,
		spans:  spans,
		covers: covers,
	}
}

func (this *IndexCountScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIndexCountScan(this)
}

func (this *IndexCountScan) New() Operator {
	return &IndexCountScan{}
}

func (this *IndexCountScan) Index() datastore.CountIndex {
	return this.index
}

func (this *IndexCountScan) Term() *algebra.KeyspaceTerm {
	return this.term
}

func (this *IndexCountScan) Spans() Spans {
	return this.spans
}

func (this *IndexCountScan) Covers() expression.Covers {
	return this.covers
}

func (this *IndexCountScan) Covering() bool {
	return len(this.covers) > 0
}

func (this *IndexCountScan) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"#operator": "IndexCountScan"}
	r["index"] = this.index.Name()
	r["namespace"] = this.term.Namespace()
	r["keyspace"] = this.term.Keyspace()
	r["using"] = this.index.Type()
	r["spans"] = this.spans

	if len(this.covers) > 0 {
		r["covers"] = this.covers
	}

	if this.duration != 0 {
		r["#time"] = this.duration.String()
	}

	return json.Marshal(r)
}

func (this *IndexCountScan) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string              `json:"#operator"`
		index     string              `json:"index"`
		namespace string              `json:"namespace"`
		keyspace  string              `json:"keyspace"`
		using     datastore.IndexType `json:"using"`
		spans     Spans               `json:"spans"`
		covers    []string            `json:"covers"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	k, err := datastore.GetKeyspace(_unmarshalled.namespace, _unmarshalled.keyspace)
	if err != nil {
		return err
	}

	this.term = algebra.NewKeyspaceTerm(
		_unmarshalled.namespace, _unmarshalled.keyspace,
		nil, "", nil, nil)

	this.spans = _unmarshalled.spans

	if _unmarshalled.covers != nil {
		this.covers = make(expression.Covers, len(_unmarshalled.covers))
		for i, c := range _unmarshalled.covers {
			expr, err := parser.Parse(c)
			if err != nil {
				return err
			}

			this.covers[i] = expression.NewCover(expr)
		}
	}

	indexer, err := k.Indexer(_unmarshalled.using)
	if err != nil {
		return err
	}

	index, err := indexer.IndexByName(_unmarshalled.index)
	if err != nil {
		return err
	}
	countIndex, ok := index.(datastore.CountIndex)
	if !ok {
		return errors.NewError(nil, "Unable to find Count() for index")
	}

	this.index = countIndex
	return nil
}
