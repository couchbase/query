//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package algebra

import (
	"sync"

	"github.com/couchbase/query/expression"
	"github.com/couchbase/query/value"
)

/*
Represents a subquery statement. It inherits from
ExpressionBase since the result representation of
the subquery is an expression and contains a field
that refers to the select statement to represent
the subquery.
*/
type Subquery struct {
	expression.ExpressionBase
	query *Select
}

/*
The function NewSubquery returns a pointer to the
Subquery struct by assigning the input attributes
to the fields of the struct.
*/
func NewSubquery(query *Select) *Subquery {
	rv := &Subquery{
		query: query,
	}

	rv.SetExpr(rv)
	return rv
}

/*
   Representation as a N1QL string.
*/
func (this *Subquery) String() string {
	var s string
	if this.IsCorrelated() || this.query.subresult.IsCorrelated() {
		s += "correlated "
	}

	return s + "(" + this.query.String() + ")"
}

/*
Visitor pattern.
*/
func (this *Subquery) Accept(visitor expression.Visitor) (interface{}, error) {
	return visitor.VisitSubquery(this)
}

/*
Subqueries return a value of type ARRAY.
*/
func (this *Subquery) Type() value.Type { return value.ARRAY }

func (this *Subquery) Evaluate(item value.Value, context expression.Context) (value.Value, error) {
	return context.(Context).EvaluateSubquery(this.query, item)
}

/*
Return false. Subquery cannot be used as a secondary
index key.
*/
func (this *Subquery) Indexable() bool {
	return false
}

/*
Return false.
*/
func (this *Subquery) IndexAggregatable() bool {
	return false
}

/*
Return false.
*/
func (this *Subquery) EquivalentTo(other expression.Expression) bool {
	return false
}

/*
Return false.
*/
func (this *Subquery) SubsetOf(other expression.Expression) bool {
	return false
}

/*
Return inner query's Expressions.
*/
func (this *Subquery) Children() expression.Expressions {
	return this.query.Expressions()
}

/*
Map inner query's Expressions.
*/
func (this *Subquery) MapChildren(mapper expression.Mapper) error {
	return this.query.MapExpressions(mapper)
}

/*
Return this subquery expression.
*/
func (this *Subquery) Copy() expression.Expression {
	return this
}

/*
TODO: This is overly broad. Ideally, we would allow:

SELECT g, (SELECT d2.* FROM d2 USE KEYS d.g) AS d2
FROM d
GROUP BY g;

but not allow:

SELECT g, (SELECT d2.* FROM d2 USE KEYS d.foo) AS d2
FROM d
GROUP BY g;
*/
func (this *Subquery) SurvivesGrouping(groupKeys expression.Expressions, allowed *value.ScopeValue) (
	bool, expression.Expression) {
	return !this.query.IsCorrelated(), nil
}

/*
This method calls FormalizeSubquery to qualify all the children
of the query, and returns an error if any.
*/
func (this *Subquery) Formalize(parent *expression.Formalizer) error {
	err := this.query.FormalizeSubquery(parent, true)
	if err != nil {
		return err
	}

	// if the subquery is correlated, add the correlation reference to
	// the parent formalizer such that any nested correlation can be detected
	// at the next level
	if this.query.IsCorrelated() {
		err = parent.AddCorrelatedIdentifiers(this.query.GetCorrelation())
	}

	return err
}

/*
Returns the subquery select statement, namely the input
query.
*/
func (this *Subquery) Select() *Select {
	return this.query
}

func (this *Subquery) IsCorrelated() bool {
	return this.query.IsCorrelated()
}

func (this *Subquery) GetCorrelation() map[string]uint32 {
	return this.query.GetCorrelation()
}

// Hold Subquery Plans IN (UDF, prepare statements, context)

type SubqueryPlans struct {
	mutex    sync.RWMutex            // mutex
	expr     expression.Expression   // Valid inside udf only
	prepared interface{}             // only inline udf, this is dummy entry
	plans    map[*Select]interface{} // map subquery plan
	isks     map[*Select]interface{} // map of keyspaces inside the transction
}

func NewSubqueryPlans() *SubqueryPlans {
	return &SubqueryPlans{
		plans: make(map[*Select]interface{}),
		isks:  make(map[*Select]interface{}),
	}
}

func (this *SubqueryPlans) GetMutex() *sync.RWMutex {
	return &this.mutex
}

func (this *SubqueryPlans) GetExpression(lock bool) expression.Expression {
	if lock {
		this.mutex.RLock()
	}
	expr := this.expr
	if lock {
		this.mutex.RUnlock()
	}
	return expr
}

func (this *SubqueryPlans) Get(key *Select, lock bool) (expression.Expression, interface{}, interface{}, bool) {
	if lock {
		this.mutex.RLock()
	}

	expr := this.expr
	plan, ok := this.plans[key]
	isk, _ := this.isks[key]

	if lock {
		this.mutex.RUnlock()
	}

	return expr, plan, isk, ok
}

func (this *SubqueryPlans) Set(key *Select, expr expression.Expression, plan, isk interface{}, lock bool) {
	if lock {
		this.mutex.Lock()
	}

	if plan != nil {
		this.plans[key] = plan
	}
	if isk != nil {
		this.isks[key] = isk
	}
	if expr != this.expr {
		this.expr = expr
	}

	if lock {
		this.mutex.Unlock()
	}
}

// Used in the context of UDF, Dummy entry to detect MetadataVersion change of collections/indexes,
// Allows auto plan generation if the metadata change vs leave the unoptimized plan for ever until UDF changed

func (this *SubqueryPlans) GetPrepared(lock bool) (prepared interface{}) {
	if lock {
		this.mutex.RLock()
		prepared = this.prepared
		this.mutex.RUnlock()
	} else {
		prepared = this.prepared
	}
	return prepared
}

func (this *SubqueryPlans) SetPrepared(prepared interface{}, lock bool) {
	if lock {
		this.mutex.Lock()
		this.prepared = prepared
		this.mutex.Unlock()
	} else {
		this.prepared = prepared
	}
}

// Copy the plans via refrence from UDF to the context.
// This allows same plan/expression used repeated execution in the same statement.

func (this *SubqueryPlans) Copy(dest *SubqueryPlans, lock bool) {
	if lock {
		this.mutex.RLock()
	}

	dest.mutex.Lock()
	// don't copy this.expr
	for key, plan := range this.plans {
		dest.plans[key] = plan
		dest.isks[key] = this.isks[key]
	}

	dest.mutex.Unlock()
	if lock {
		this.mutex.RUnlock()
	}
}

// Validation of each subquery plans.

func (this *SubqueryPlans) ForEach(expr expression.Expression, options uint32, lock bool,
	verify func(key *Select, options uint32, plan, isk interface{}) (bool, bool)) (bool, bool) {

	var good, trans bool
	if lock {
		this.mutex.RLock()
		defer this.mutex.RUnlock()
	}
	if expr == this.expr {
		good = true
		for key, _ := range this.plans {
			good1, trans1 := verify(key, options, this.plans[key], this.isks[key])
			if !good1 {
				return good1, false
			}
			trans = trans || trans1
		}
	}
	return good, trans
}
