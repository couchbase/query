//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package expression

import (
	"sync"
	"sync/atomic"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/util"
	"github.com/couchbase/query/value"
)

/*
Represents the collection expression IN.
*/
type In struct {
	BinaryFunctionBase
}

func NewIn(first, second Expression) Function {
	rv := &In{
		*NewBinaryFunctionBase("in", first, second),
	}

	if secondArr, ok := second.(*ArrayConstruct); ok {
		secondArr.SetExprFlag(EXPR_ARRAY_IS_SET)
	}
	rv.expr = rv
	return rv
}

func (this *In) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIn(this)
}

func (this *In) Type() value.Type { return value.BOOLEAN }

/*
IN evaluates to TRUE if the right-hand-side first value is an array
and directly contains the left-hand-side second value.
*/
func (this *In) Evaluate(item value.Value, context Context) (value.Value, error) {
	first, err := this.operands[0].Evaluate(item, context)
	if err != nil {
		return nil, err
	}
	second, err := this.operands[1].Evaluate(item, context)
	if err != nil {
		return nil, err
	}

	if first.Type() == value.MISSING || second.Type() == value.MISSING {
		return value.MISSING_VALUE, nil
	} else if second.Type() != value.ARRAY {
		return value.NULL_VALUE, nil
	}

	var hashTab *util.HashTable
	var missing, null bool
	buildHT := false
	sa := second.Actual().([]interface{})

	var inlistHash *InlistHash
	if context != nil {
		if inlistContext, ok := context.(InlistContext); ok {
			inlistHash = inlistContext.GetInlistHash(this)
		}
	}

	if inlistHash != nil {
		inlistHash.hashLock.Lock()
		if !inlistHash.HashChecked() {
			inlistHash.SetHashChecked()
			if inlistHash.IsHashEnabled() && len(sa) >= _INLIST_HASH_THRESHOLD {
				if this.Second().Static() != nil {
					buildHT = true
				} else if sq, ok := this.Second().(Subquery); ok && !sq.IsCorrelated() {
					buildHT = true
				}
			}
		}
		if buildHT {
			hashTab = util.NewHashTable(util.HASH_TABLE_FOR_INLIST)
			inlistHash.hashTab = hashTab
			// lock is not released until hash table is built
		} else {
			hashTab = inlistHash.hashTab
			inlistHash.hashLock.Unlock()
		}
	}

	if hashTab == nil || buildHT {
		for _, s := range sa {
			v := value.NewValue(s)
			if v.Type() > value.NULL {
				if buildHT {
					err := hashTab.Put(v, true, value.MarshalValue, value.EqualValue, 0)
					if err != nil {
						inlistHash.hashLock.Unlock()
						return nil, errors.NewHashTablePutError(err)
					}
				} else if first.Type() > value.NULL {
					if first.Equals(v).Truth() {
						return value.TRUE_VALUE, nil
					}
				} else {
					// first.Type() == value.NULL
					null = true
				}
			} else if v.Type() == value.MISSING {
				if buildHT {
					inlistHash.SetMissing()
				} else {
					missing = true
				}
			} else {
				// v.Type() == value.NULL
				if buildHT {
					inlistHash.SetNull()
				} else {
					null = true
				}
			}
		}
		if buildHT {
			inlistHash.hashLock.Unlock()
		}
	}

	if hashTab != nil {
		if first.Type() > value.NULL {
			outVal, err := hashTab.Get(first, value.MarshalValue, value.EqualValue)
			if err != nil {
				return nil, errors.NewHashTableGetError(err)
			}
			if outVal != nil {
				return value.TRUE_VALUE, nil
			}
		} else {
			null = true
		}
	}

	if null || (inlistHash != nil && inlistHash.HasNull()) {
		return value.NULL_VALUE, nil
	} else if missing || (inlistHash != nil && inlistHash.HasMissing()) {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
}

/*
If this expression is in the WHERE clause of a partial index, lists
the Expressions that are implicitly covered.

For IN, simply list this expression.
*/
func (this *In) FilterCovers(covers map[string]value.Value) map[string]value.Value {
	covers[this.String()] = value.TRUE_VALUE
	return covers
}

func (this *In) FilterExpressionCovers(covers map[Expression]value.Value) map[Expression]value.Value {
	covers[this] = value.TRUE_VALUE
	return covers
}

func (this *In) MayOverlapSpans() bool {
	return this.Second().Value() == nil
}

/*
Factory method pattern.
*/
func (this *In) Constructor() FunctionConstructor {
	return func(operands ...Expression) Function {
		return NewIn(operands[0], operands[1])
	}
}

func (this *In) EnableInlistHash(context Context) {
	if context != nil {
		if inlistContext, ok := context.(InlistContext); ok {
			inlistContext.EnableInlistHash(this)
			for _, child := range this.expr.Children() {
				child.EnableInlistHash(context)
			}
		}
	}
}

func (this *In) ResetMemory(context Context) {
	if context != nil {
		if inlistContext, ok := context.(InlistContext); ok {
			inlistContext.RemoveInlistHash(this)
			for _, child := range this.expr.Children() {
				child.ResetMemory(context)
			}
		}
	}
}

/*
  Hash tables used in IN-list evaluation.
  Note that during execution the expression itself comes from
  the plan, which may be shared among multiple executions, thus
  no change to the expression itself should be allowed during
  execution. Therefore hash table should be put somewhere else
  and not inside the expression itself.
  We'll use a hash map in the execution context to store a
  structure (InlistHash) which includes a hash table pointer.
*/
const (
	INEXPR_HASHTAB_CHECKED = 1 << iota
	INEXPR_HAS_NULL
	INEXPR_HAS_MISSING
)

type InlistHash struct {
	hashTab   *util.HashTable
	hashFlags uint32
	hashCnt   int32
	hashLock  sync.Mutex
}

func NewInlistHash() *InlistHash {
	return &InlistHash{}
}

func (this *InlistHash) HashChecked() bool {
	return (this.hashFlags & INEXPR_HASHTAB_CHECKED) != 0
}

func (this *InlistHash) SetHashChecked() {
	this.hashFlags |= INEXPR_HASHTAB_CHECKED
}

func (this *InlistHash) IsHashEnabled() bool {
	return this.hashCnt > 0
}

func (this *InlistHash) EnableHash() {
	// in case of parallelism, keep number of instances
	atomic.AddInt32(&(this.hashCnt), 1)
}

func (this *InlistHash) HasMissing() bool {
	return (this.hashFlags & INEXPR_HAS_MISSING) != 0
}

func (this *InlistHash) SetMissing() {
	this.hashFlags |= INEXPR_HAS_MISSING
}

func (this *InlistHash) HasNull() bool {
	return (this.hashFlags & INEXPR_HAS_NULL) != 0
}

func (this *InlistHash) SetNull() {
	this.hashFlags |= INEXPR_HAS_NULL
}

func (this *InlistHash) DropHashTab() {
	// let the last instance drop the hash table
	if atomic.AddInt32(&(this.hashCnt), -1) == 0 {
		if this.hashTab != nil {
			this.hashTab.Drop()
			this.hashTab = nil
		}
		this.hashFlags = 0
	}
}

/*
This function implements the NOT IN collection operation.
*/
func NewNotIn(first, second Expression) Expression {
	return NewNot(NewIn(first, second))
}

const _INLIST_HASH_THRESHOLD = 16
