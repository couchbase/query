//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

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

	name         string                `json:"name"`
	keyspace     *KeyspaceRef          `json:"keyspace"`
	keys         IndexKeyTerms         `json:"keys"`
	include      *IndexIncludeTerm     `json:"include"`
	partition    *IndexPartitionTerm   `json:"partition"`
	where        expression.Expression `json:"where"`
	using        datastore.IndexType   `json:"using"`
	with         value.Value           `json:"with"`
	failIfExists bool                  `json:"failIfExists"`
	vector       bool                  `json:"vector"`
}

/*
The function NewCreateIndex returns a pointer to the
CreateIndex struct with the input argument values as fields.
*/
func NewCreateIndex(name string, keyspace *KeyspaceRef, keys IndexKeyTerms, include *IndexIncludeTerm,
	partition *IndexPartitionTerm, where expression.Expression, using datastore.IndexType, with value.Value,
	failIfExists, vector bool) *CreateIndex {
	rv := &CreateIndex{
		name:         name,
		keyspace:     keyspace,
		keys:         keys,
		include:      include,
		partition:    partition,
		where:        where,
		using:        using,
		with:         with,
		failIfExists: failIfExists,
		vector:       vector,
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

	if this.include != nil {
		err = this.include.MapExpressions(mapper)
		if err != nil {
			return
		}
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

	if this.include != nil && len(this.include.Expressions()) > 0 {
		exprs = append(exprs, this.include.Expressions()...)
	}

	if this.partition != nil && len(this.partition.Expressions()) > 0 {
		exprs = append(exprs, this.partition.Expressions()...)
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
	if this.using == datastore.FTS {
		privs.Add(fullName, auth.PRIV_SEARCH_CREATE_INDEX, auth.PRIV_PROPS_NONE)
	} else {
		privs.Add(fullName, auth.PRIV_QUERY_CREATE_INDEX, auth.PRIV_PROPS_NONE)
	}

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

func (this *CreateIndex) Include() *IndexIncludeTerm {
	return this.include
}

/*
Returns the Partition expression of the create index statement.
*/
func (this *CreateIndex) Partition() *IndexPartitionTerm {
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

func (this *CreateIndex) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateIndex) Vector() bool {
	return this.vector
}

func (this *CreateIndex) SeekKeys() expression.Expressions {
	return nil
}

func (this *CreateIndex) RangeKeys() expression.Expressions {
	return this.keys.Expressions()
}

/*
Marshals input receiver into byte array.
*/
func (this *CreateIndex) MarshalJSON() ([]byte, error) {
	r := map[string]interface{}{"type": "createIndex"}
	r["keyspaceRef"] = this.keyspace
	r["name"] = this.name
	r["keys"] = this.keys
	if this.include != nil {
		r["include"] = this.include
	}
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
	r["failIfExists"] = this.failIfExists

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
	attributes uint32                `json:"attributes"`
}

const (
	IK_ASC = 1 << iota
	IK_DESC
	IK_MISSING
	IK_VECTOR
	IK_NONE = 0
)

func NewIndexKeyTermAttributes(attributes ...uint32) (v uint32, b bool) {
	for i := 0; i < len(attributes); i++ {
		for j := i + 1; j < len(attributes); j++ {
			// Don't allow repeat of same attribute, ASC/DESC together.
			if (attributes[i]&attributes[j]) != 0 ||
				((attributes[i]&(IK_ASC|IK_DESC)) != 0 && (attributes[j]&(IK_ASC|IK_DESC)) != 0) {
				return uint32(0), false
			}
		}
		v |= attributes[i]
	}
	return v, true
}

/*
The function NewIndexKeyTerm returns a pointer to the IndexKeyTerm
struct that has its fields set to the input arguments.
*/
func NewIndexKeyTerm(expr expression.Expression, attributes uint32) *IndexKeyTerm {
	return &IndexKeyTerm{
		expr:       expr,
		attributes: attributes,
	}
}

/*
Representation as a N1QL string.
*/
func (this *IndexKeyTerm) String(pos int) string {
	s := this.expr.String()

	if pos == 0 && this.HasAttribute(IK_MISSING) {
		s += " INCLUDE MISSING"
	}

	if this.HasAttribute(IK_DESC) {
		s += " DESC"
	}

	if this.HasAttribute(IK_VECTOR) {
		s += " VECTOR"
	}

	return s
}

/*
Return the expression that is create index
*/
func (this *IndexKeyTerm) Expression() expression.Expression {
	return this.expr
}

func (this *IndexKeyTerm) Attributes() uint32 {
	return this.attributes
}

func (this *IndexKeyTerm) SetAttribute(attr uint32, add bool) {
	if add {
		this.attributes |= attr
	} else {
		this.attributes = attr
	}
}

func (this *IndexKeyTerm) UnsetAttribute(attr uint32) {
	this.attributes &^= attr
}

func (this *IndexKeyTerm) HasAttribute(attr uint32) bool {
	return (this.attributes & attr) != 0
}

func (this *IndexKeyTerm) HasDesc() bool {
	return (this.attributes & IK_DESC) != 0
}

func (this *IndexKeyTerm) HasMissing() bool {
	return (this.attributes & IK_MISSING) != 0
}

func (this *IndexKeyTerm) HasVector() bool {
	return (this.attributes & IK_VECTOR) != 0
}

/*
Return bool value representing Index Leading Missing
*/
func (this IndexKeyTerms) Missing() bool {
	return this[0].HasAttribute(IK_MISSING)
}

/*
Return bool value representing ASC or DESC collation order.
*/
func (this IndexKeyTerms) HasDescending() bool {
	for _, term := range this {
		if term.HasAttribute(IK_DESC) {
			return true
		}
	}
	return false
}

func (this IndexKeyTerms) HasVector() bool {
	for _, term := range this {
		if term.HasAttribute(IK_VECTOR) {
			return true
		}
		all, ok := term.Expression().(*expression.All)
		if ok && all.Flatten() {
			fk := all.FlattenKeys()
			for pos, _ := range fk.Operands() {
				if fk.HasVector(pos) {
					return true
				}
			}
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

func (this IndexKeyTerms) Attributes() []uint32 {
	attrs := make([]uint32, len(this))

	for i, term := range this {
		attrs[i] = term.attributes
	}

	return attrs
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

		s += term.String(i)
	}

	return s
}

type IndexIncludeTerm struct {
	exprs expression.Expressions `json:"exprs"`
}

func NewIndexIncludeTerm(exprs expression.Expressions) *IndexIncludeTerm {
	return &IndexIncludeTerm{
		exprs: exprs,
	}
}

func (this *IndexIncludeTerm) Expressions() expression.Expressions {
	return this.exprs
}

func (this *IndexIncludeTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if len(this.exprs) > 0 {
		err = this.exprs.MapExpressions(mapper)
		if err != nil {
			return
		}
	}
	return
}

func (this *IndexIncludeTerm) String() (s string) {
	if this != nil {
		s += " INCLUDE("
		for i, expr := range this.exprs {
			if i > 0 {
				s += ", "
			}
			s += expr.String()
		}
		s += ") "
	}
	return
}

/*
Represents the Partition term in create index.
*/
type IndexPartitionTerm struct {
	strategy datastore.PartitionType `json:"strategy"`
	exprs    expression.Expressions  `json:"exprs"`
}

/*
The function NewIndexPartitionTerm returns a pointer to the IndexPartitionTerm
struct that has its fields set to the input arguments.
*/
func NewIndexPartitionTerm(strategy datastore.PartitionType, exprs expression.Expressions) *IndexPartitionTerm {
	return &IndexPartitionTerm{
		strategy: strategy,
		exprs:    exprs,
	}
}

/*
Returns all contained Expressions.
*/
func (this *IndexPartitionTerm) Expressions() expression.Expressions {
	return this.exprs
}

/*
Returns Partition Strategy
*/
func (this *IndexPartitionTerm) Strategy() datastore.PartitionType {
	return this.strategy
}

/*
This method maps the partition expressions
*/
func (this *IndexPartitionTerm) MapExpressions(mapper expression.Mapper) (err error) {
	if len(this.exprs) > 0 {
		err = this.exprs.MapExpressions(mapper)
		if err != nil {
			return
		}
	}
	return
}

func (this *IndexPartitionTerm) String() (s string) {
	if this.strategy == datastore.HASH_PARTITION {
		s += " PARTITION BY HASH("
		for i, expr := range this.exprs {
			if i > 0 {
				s += ", "
			}
			s += expr.String()
		}
		s += ") "
	}
	return
}
