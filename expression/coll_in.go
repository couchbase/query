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
	rv := &In{}
	rv.Init("in", first, second)

	if secondArr, ok := second.(*ArrayConstruct); ok {
		secondArr.SetExprFlag(EXPR_ARRAY_IS_SET)
	}
	rv.expr = rv
	return rv
}

func (this *In) Accept(visitor Visitor) (any, error) {
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

	sa := second.Actual().([]any)

	if first.Type() == value.NULL {
		if len(sa) == 0 {
			return value.FALSE_VALUE, nil
		}
		return value.NULL_VALUE, nil
	}

	if context != nil {
		if inlistContext, ok := context.(InlistContext); ok {
			if inlistHash := inlistContext.GetInlistHash(this); inlistHash != nil {
				hashTab, flags, err := inlistHash.getHashTab(this, sa)
				if err != nil {
					return nil, err
				}
				if hashTab != nil {
					found, err := hashTab.Contains(first, value.MarshalValue, value.EqualValue)
					if err != nil {
						return nil, errors.NewHashTableGetError(err)
					}
					if found {
						return value.TRUE_VALUE, nil
					} else if (flags & INEXPR_HAS_NULL) != 0 {
						return value.NULL_VALUE, nil
					} else if (flags & INEXPR_HAS_MISSING) != 0 {
						return value.MISSING_VALUE, nil
					}
					return value.FALSE_VALUE, nil
				}
			}
		}
	}

	var missing, null bool
	for _, s := range sa {
		v := value.NewValue(s)
		if v.Type() > value.NULL {
			if first.Equals(v).Truth() {
				return value.TRUE_VALUE, nil
			}
		} else if v.Type() == value.MISSING {
			missing = true
		} else {
			// v.Type() == value.NULL
			null = true
		}
	}

	if null {
		return value.NULL_VALUE, nil
	} else if missing {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
}

/*
Whether a hash table can be built for the IN-list, i.e. the IN-list does not
change from one evaluation to the next.
*/
func (this *In) inlistHashable() bool {
	right := this.Second()
	if right.Static() != nil {
		ident, ok := right.(*Identifier)
		return !ok || !ident.IsCorrelated()
	} else if sq, ok := right.(Subquery); ok && !sq.IsCorrelated() {
		return true
	}
	return false
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

/*
The hash table is built once, by the first evaluation, while holding hashLock.
hashTab is set before INEXPR_HASHTAB_CHECKED is (atomically) set in hashFlags,
so once INEXPR_HASHTAB_CHECKED is set subsequent evaluations can read hashTab
and the other flags without taking hashLock. The hash table is not modified
after it's built, and lookups use HashTable.Contains(), which is read-only, so
the hash table can be shared among parallel instances of an operator.
*/
type InlistHash struct {
	hashTab   *util.HashTable
	hashFlags uint32
	hashCnt   int32
	hashLock  sync.Mutex
}

func NewInlistHash() *InlistHash {
	return &InlistHash{}
}

func (this *InlistHash) IsHashEnabled() bool {
	return atomic.LoadInt32(&this.hashCnt) > 0
}

func (this *InlistHash) EnableHash() {
	// in case of parallelism, keep number of instances
	atomic.AddInt32(&(this.hashCnt), 1)
}

func (this *InlistHash) HasMissing() bool {
	return (atomic.LoadUint32(&this.hashFlags) & INEXPR_HAS_MISSING) != 0
}

func (this *InlistHash) SetMissing() {
	atomic.OrUint32(&this.hashFlags, INEXPR_HAS_MISSING)
}

func (this *InlistHash) HasNull() bool {
	return (atomic.LoadUint32(&this.hashFlags) & INEXPR_HAS_NULL) != 0
}

func (this *InlistHash) SetNull() {
	atomic.OrUint32(&this.hashFlags, INEXPR_HAS_NULL)
}

/*
Returns the hash table for the IN-list (nil if no hash table is used) and the hash flags.
The hash table is built on first call.
*/
func (this *InlistHash) getHashTab(in *In, sa []any) (*util.HashTable, uint32, error) {
	flags := atomic.LoadUint32(&this.hashFlags)
	if (flags & INEXPR_HASHTAB_CHECKED) == 0 {
		this.hashLock.Lock()
		flags = atomic.LoadUint32(&this.hashFlags)
		if (flags & INEXPR_HASHTAB_CHECKED) == 0 {
			var err error
			if this.IsHashEnabled() && len(sa) >= _INLIST_HASH_THRESHOLD && in.inlistHashable() {
				err = this.buildHashTab(sa)
			}
			// INEXPR_HAS_NULL/INEXPR_HAS_MISSING set during build are not looked at
			// until INEXPR_HASHTAB_CHECKED is set, which must be set last
			// if hash table build fails, the error is returned for this evaluation,
			// and subsequent evaluations do a linear scan of the IN-list
			flags = atomic.OrUint32(&this.hashFlags, INEXPR_HASHTAB_CHECKED) | INEXPR_HASHTAB_CHECKED
			if err != nil {
				this.hashLock.Unlock()
				return nil, flags, err
			}
		}
		this.hashLock.Unlock()
	}
	return this.hashTab, flags, nil
}

// caller must hold hashLock; hashTab is only set if the build is successful
func (this *InlistHash) buildHashTab(sa []any) error {
	hashTab := util.NewHashTable(util.HASH_TABLE_FOR_INLIST, float64(len(sa)), 1)
	for _, s := range sa {
		v := value.NewValue(s)
		if v.Type() > value.NULL {
			err := hashTab.Put(v, true, value.MarshalValue, value.EqualValue, 0)
			if err != nil {
				hashTab.Drop()
				atomic.StoreUint32(&this.hashFlags, 0)
				return errors.NewHashTablePutError(err)
			}
		} else if v.Type() == value.MISSING {
			this.SetMissing()
		} else {
			// v.Type() == value.NULL
			this.SetNull()
		}
	}
	this.hashTab = hashTab
	return nil
}

// returns true if this is the last instance, i.e. the hash table is dropped
func (this *InlistHash) DropHashTab() bool {
	// let the last instance drop the hash table
	if atomic.AddInt32(&(this.hashCnt), -1) == 0 {
		this.hashLock.Lock()
		if this.hashTab != nil {
			this.hashTab.Drop()
			this.hashTab = nil
		}
		atomic.StoreUint32(&this.hashFlags, 0)
		this.hashLock.Unlock()
		return true
	}
	return false
}

/*
This function implements the NOT IN collection operation.
*/
func NewNotIn(first, second Expression) Expression {
	return NewNot(NewIn(first, second))
}

const _INLIST_HASH_THRESHOLD = 16
