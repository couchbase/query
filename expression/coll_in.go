//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

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

	rv.expr = rv
	return rv
}

func (this *In) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitIn(this)
}

func (this *In) Type() value.Type { return value.BOOLEAN }

func (this *In) Evaluate(item value.Value, context Context) (value.Value, error) {
	return this.BinaryEval(this, item, context)
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

func (this *In) MayOverlapSpans() bool {
	return this.Second().Value() == nil
}

/*
IN evaluates to TRUE if the right-hand-side first value is an array
and directly contains the left-hand-side second value.
*/
func (this *In) Apply(context Context, first, second value.Value) (value.Value, error) {
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
	if inlistContext, ok := context.(InlistContext); ok {
		inlistHash = inlistContext.GetInlistHash(this)
	}

	if inlistHash != nil {
		inlistHash.hashLock.Lock()
		if inlistHash.ChkHash() {
			inlistHash.SetHashChecked()
			if inlistHash.IsHashEnabled() && len(sa) >= _INLIST_HASH_THRESHOLD {
				if this.Second().Static() != nil {
					buildHT = true
				} else if sq, ok := this.Second().(Subquery); ok && !sq.IsCorrelated() {
					buildHT = true
				}
				if buildHT {
					hashTab = util.NewHashTable(util.HASH_TABLE_FOR_INLIST)
					inlistHash.hashTab = hashTab
				}
			}
			// lock is not released until hash table is built
		} else {
			hashTab = inlistHash.hashTab
			inlistHash.hashLock.Unlock()
		}
	}

	if hashTab == nil || buildHT {
		for _, s := range sa {
			v := value.NewValue(s)
			if first.Type() > value.NULL && v.Type() > value.NULL {
				if buildHT {
					err := hashTab.Put(v, true, value.MarshalValue, value.EqualValue, 0)
					if err != nil {
						return nil, errors.NewHashTablePutError(err)
					}
				} else {
					if first.Equals(v).Truth() {
						return value.TRUE_VALUE, nil
					}
				}
			} else if v.Type() == value.MISSING {
				if buildHT {
					inlistHash.SetMissing()
				} else {
					missing = true
				}
			} else {
				// first.Type() == value.NULL || v.Type() == value.NULL
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
		outVal, err := hashTab.Get(first, value.MarshalValue, value.EqualValue)
		if err != nil {
			return nil, errors.NewHashTableGetError(err)
		}
		if outVal != nil {
			return value.TRUE_VALUE, nil
		}
	}

	if null || (buildHT && inlistHash.HasNull()) {
		return value.NULL_VALUE, nil
	} else if missing || (buildHT && inlistHash.HasMissing()) {
		return value.MISSING_VALUE, nil
	} else {
		return value.FALSE_VALUE, nil
	}
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
	if inlistContext, ok := context.(InlistContext); ok {
		inlistContext.EnableInlistHash(this)
		for _, child := range this.expr.Children() {
			child.EnableInlistHash(context)
		}
	}
}

func (this *In) ResetMemory(context Context) {
	if inlistContext, ok := context.(InlistContext); ok {
		inlistContext.RemoveInlistHash(this)
		for _, child := range this.expr.Children() {
			child.ResetMemory(context)
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

func (this *InlistHash) ChkHash() bool {
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
