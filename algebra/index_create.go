//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package algebra

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents the Create index ddl statement. Type CreateIndex is
a struct that contains fields mapping to each clause in the
create index statement. The fields refer to the index name,
keyspace ref, expression, partition, where clause and using clause
(IndexType string).

The partition expression is used to compute the hash value for
partitioning the index across multiple nodes. When a document
is indexed, the expression is evaluated for that document, and
the resulting value determines which index node will contain an
index value into the document.
*/
type CreateIndex struct {
	statementBase

	name      string                 `json:"name"`
	keyspace  *KeyspaceRef           `json:"keyspace"`
	keys      IndexKeyTerms          `json:"keys"`
	partition expression.Expressions `json:"partition"`
	where     expression.Expression  `json:"where"`
	using     datastore.IndexType    `json:"using"`
	with      value.Value            `json:"with"`
}

/*
The function NewCreateIndex returns a pointer to the
CreateIndex struct with the input argument values as fields.
*/
func NewCreateIndex(name string, keyspace *KeyspaceRef, keys IndexKeyTerms, partition expression.Expressions,
	where expression.Expression, using datastore.IndexType, with value.Value) *CreateIndex {
	rv := &CreateIndex{
		name:      name,
		keyspace:  keyspace,
		keys:      keys,
		partition: partition,
		where:     where,
		using:     using,
		with:      with,
	}

	rv.stmt = rv
	return rv
}

/*
It calls the VisitCreateIndex method by passing
in the receiver and returns the interface. It is a
visitor pattern.
*/
func (this *CreateIndex) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateIndex(this)
}

/*
Returns nil.
*/
func (this *CreateIndex) Signature() value.Value {
	return nil
}

/*
Returns nil.
*/
func (this *CreateIndex) Formalize() error {
	f := expression.NewKeyspaceFormalizer(this.keyspace.Keyspace(), nil)
	return this.MapExpressions(f)
}

/*
This method maps all the constituent clauses, namely the expression,
partition and where clause within a create index statement.
*/
func (this *CreateIndex) MapExpressions(mapper expression.Mapper) (err error) {
	err = this.keys.MapExpressions(mapper)
	if err != nil {
		return
	}

	if this.partition != nil {
		err = this.partition.MapExpressions(mapper)
		if err != nil {
			return
		}
	}

	if this.where != nil {
		this.where, err = mapper.Map(this.where)
		if err != nil {
			return
		}
	}

	return
}

/*
Return expr from the create index statement.
*/
func (this *CreateIndex) Expressions() expression.Expressions {
	exprs := this.keys.Expressions()

	if this.partition != nil {
		exprs = append(exprs, this.partition...)
	}

	if this.where != nil {
		exprs = append(exprs, this.where)
	}

	return exprs
}

/*
Returns all required privileges.
*/
func (this *CreateIndex) Privileges() (*auth.Privileges, errors.Error) {
	privs := auth.NewPrivileges()
	fullName := this.keyspace.FullName()
	privs.Add(fullName, auth.PRIV_QUERY_CREATE_INDEX)

	for _, expr := range this.Expressions() {
		privs.AddAll(expr.Privileges())
	}
	return privs, nil
}

/*
Returns the name of the index.
*/
func (this *CreateIndex) Name() string {
	return this.name
}

/*
Returns the bucket (keyspace) that the index is created on.
*/
func (this *CreateIndex) Keyspace() *KeyspaceRef {
	return this.keyspace
}

/*
Return keys from the create index statement.
*/
func (this *CreateIndex) Keys() IndexKeyTerms {
	return this.keys
}

/*
Returns the Partition expression of the create index statement.
*/
func (this *CreateIndex) Partition() expression.Expressions {
	return this.partition
}

/*
Returns the where condition in the create index statement.
*/
func (this *CreateIndex) Where() expression.Expression {
	return this.where
}

/*
Returns the index type string for the using clause.
*/
func (this *CreateIndex) Using() datastore.IndexType {
	return this.using
}

/*
Returns the WITH deployment plan.
*/
func (this *CreateIndex) With() value.Value {
	return this.with
}

func (this *CreateIndex) SeekKeys() expression.Expressions {
	return this.partition
}

func (this *CreateIndex) RangeKeys() expression.Expressions {
	exprs := this.keys.Expressions()
	if this.partition != nil {
		return exprs[len(this.partition):]
	} else {
		return exprs
	}
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["keys"] = this.keys
	if this.partition != nil {
		r["partition"] = this.partition
	}
	if this.where != nil {
		r["where"] = this.where
	}
	r["using"] = this.using
	if this.with != nil {
		r["with"] = this.with
	}

	return json.Marshal(r)
}

func (this *CreateIndex) Type() string {
	return "CREATE_INDEX"
}

/*
It represents multiple IndexKey terms.
Type IndexKeyTerms is a slice of IndexKeyTerm.
*/

type IndexKeyTerms []*IndexKeyTerm

/*
Represents the index key term in create index. Type
IndexKeyTerm is a struct containing the expression and a bool
value that decides the IndexKey collation (ASC or DESC).
*/
type IndexKeyTerm struct {
	expr       expression.Expression `json:"expr"`
	descending bool                  `json:"desc"`
}

/*
The function NewIndexKeyTerm returns a pointer to the IndexKeyTerm
struct that has its fields set to the input arguments.
*/
func NewIndexKeyTerm(expr expression.Expression, descending bool) *IndexKeyTerm {
	return &IndexKeyTerm{
		expr:       expr,
		descending: descending,
	}
}

/*
   Representation as a N1QL string.
*/
func (this *IndexKeyTerm) String() string {
	s := this.expr.String()

	if this.descending {
		s += " desc"
	}

	return s
}

/*
Return the expression that is create index
*/
func (this *IndexKeyTerm) Expression() expression.Expression {
	return this.expr
}

/*
Return bool value representing ASC or DESC collation order.
*/
func (this *IndexKeyTerm) Descending() bool {
	return this.descending
}

/*
Return bool value representing ASC or DESC collation order.
*/
func (this IndexKeyTerms) HasDescending() bool {
	for _, term := range this {
		if term.Descending() {
			return true
		}
	}
	return false
}

/*
Map Expressions for all IndexKey terms in the receiver.
*/
func (this IndexKeyTerms) MapExpressions(mapper expression.Mapper) (err error) {
	for _, term := range this {
		term.expr, err = mapper.Map(term.expr)
		if err != nil {
			return
		}
	}

	return
}

/*
   Returns all contained Expressions.
*/
func (this IndexKeyTerms) Expressions() expression.Expressions {
	exprs := make(expression.Expressions, len(this))

	for i, term := range this {
		exprs[i] = term.expr
	}

	return exprs
}

/*
   Representation as a N1QL string.
*/
func (this IndexKeyTerms) String() string {
	s := ""

	for i, term := range this {
		if i > 0 {
			s += ", "
		}

		s += term.String()
	}

	return s
}
