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
	"github.com/couchbase/query/value"
)

// Create index
type CreateIndex struct {
	ddl
	keyspace datastore.Keyspace
	node     *algebra.CreateIndex
}

func NewCreateIndex(keyspace datastore.Keyspace, node *algebra.CreateIndex) *CreateIndex {
	return &CreateIndex{
		keyspace: keyspace,
		node:     node,
	}
}

func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

func (this *CreateIndex) New() Operator {
	return &CreateIndex{}
}

func (this *CreateIndex) Keyspace() datastore.Keyspace {
	return this.keyspace
}

func (this *CreateIndex) Node() *algebra.CreateIndex {
	return this.node
}

func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateIndex) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateIndex"}
	this.node.Keyspace().MarshalKeyspace(r)
	r["index"] = this.node.Name()
	k := make([]interface{}, len(this.node.Keys()))
	for i, term := range this.node.Keys() {
		q := make(map[string]interface{}, 2)
		q["expr"] = term.Expression().String()

		if term.HasAttribute(algebra.IK_MISSING) {
			q["missing"] = true
		}
		if term.HasAttribute(algebra.IK_DESC) {
			q["desc"] = true
		}

		if vectorName := term.VectorName(); vectorName != "" {
			q["vectorType"] = vectorName
		}

		k[i] = q
	}
	r["keys"] = k
	r["using"] = this.node.Using()

	if this.node.Include() != nil {
		r["include"] = this.node.Include().Expressions()
	}

	if this.node.Partition() != nil && this.node.Partition().Strategy() != datastore.NO_PARTITION {
		q := make(map[string]interface{}, 2)
		q["exprs"] = this.node.Partition().Expressions()
		q["strategy"] = this.node.Partition().Strategy()
		r["partition"] = q
	}

	if this.node.Where() != nil {
		r["where"] = this.node.Where()
	}

	if this.node.With() != nil {
		r["with"] = this.node.With()
	}
	// invert so the default if not present is to fail if exists
	r["ifNotExists"] = !this.node.FailIfExists()
	r["vector"] = this.node.Vector()

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateIndex) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
		Keyspace  string `json:"keyspace"`
		Index     string `json:"index"`
		Keys      []struct {
			Expr       string `json:"expr"`
			Desc       bool   `json:"desc"`
			Missing    bool   `json:"missing"`
			Vector     bool   `json:"vector"`
			VectorType string `json:"vectorType"`
		} `json:"keys"`
		Using     datastore.IndexType `json:"using"`
		Include   []string            `json:"include"`
		Partition *struct {
			Exprs    []string                `json:"exprs"`
			Strategy datastore.PartitionType `json:"strategy"`
		} `json:"partition"`
		Where       string          `json:"where"`
		With        json.RawMessage `json:"with"`
		IfNotExists bool            `json:"ifNotExists"`
		Vector      bool            `json:"vector"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	ksref := algebra.NewKeyspaceRefFromPath(algebra.NewPathShortOrLong(_unmarshalled.Namespace, _unmarshalled.Bucket,
		_unmarshalled.Scope, _unmarshalled.Keyspace), "")
	this.keyspace, err = datastore.GetKeyspace(ksref.Path().Parts()...)
	if err != nil {
		return err
	}

	var expr expression.Expression
	keys := make(algebra.IndexKeyTerms, len(_unmarshalled.Keys))

	for i, term := range _unmarshalled.Keys {
		expr, err = parser.Parse(term.Expr)
		if err != nil {
			return err
		}
		attributes := uint32(0)
		if term.Desc {
			attributes |= algebra.IK_DESC
		}
		if term.Missing {
			attributes |= algebra.IK_MISSING
		}
		if term.VectorType != "" {
			attributes |= algebra.VectorAttribute(term.VectorType)
		} else if term.Vector {
			attributes |= algebra.IK_DENSE_VECTOR
		}

		keys[i] = algebra.NewIndexKeyTerm(expr, attributes)
	}

	var include *algebra.IndexIncludeTerm
	if len(_unmarshalled.Include) > 0 {
		exprs := make(expression.Expressions, len(_unmarshalled.Include))
		for i, p := range _unmarshalled.Include {
			exprs[i], err = parser.Parse(p)
			if err != nil {
				return err
			}
		}
		include = algebra.NewIndexIncludeTerm(exprs)
	}

	if keys.HasDescending() || keys.HasVector() || include != nil {
		indexer, err1 := this.keyspace.Indexer(_unmarshalled.Using)
		if err1 != nil {
			return err1
		}
		if keys.HasDescending() {
			if _, ok := indexer.(datastore.Indexer2); !ok {
				return errors.NewIndexerDescCollationError()
			}
		}
		if _, ok := indexer.(datastore.Indexer6); !ok {
			if keys.HasVector() {
				return errors.NewIndexerVersionError(datastore.INDEXER6_VERSION, "Index key has vector attribute")
			}
			if include != nil {
				return errors.NewIndexerVersionError(datastore.INDEXER6_VERSION, "Include clause present")
			}
		}
	}

	var partition *algebra.IndexPartitionTerm
	if _unmarshalled.Partition != nil {
		exprs := make(expression.Expressions, len(_unmarshalled.Partition.Exprs))
		for i, p := range _unmarshalled.Partition.Exprs {
			exprs[i], err = parser.Parse(p)
			if err != nil {
				return err
			}
		}
		partition = algebra.NewIndexPartitionTerm(_unmarshalled.Partition.Strategy, exprs)
	}

	var where expression.Expression
	if _unmarshalled.Where != "" {
		where, err = parser.Parse(_unmarshalled.Where)
		if err != nil {
			return err
		}
	}

	var with value.Value
	if len(_unmarshalled.With) > 0 {
		with = value.NewValue([]byte(_unmarshalled.With))
	}

	// invert IfNotExists to obtain FailIfExists
	this.node = algebra.NewCreateIndex(_unmarshalled.Index, ksref,
		keys, include, partition, where, _unmarshalled.Using, with, !_unmarshalled.IfNotExists,
		_unmarshalled.Vector)
	return nil
}

func (this *CreateIndex) verify(prepared *Prepared) errors.Error {
	var err errors.Error

	this.keyspace, err = verifyKeyspace(this.keyspace, prepared)
	return err
}
