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
)

type ExpressionPool struct {
	pool *sync.Pool
	size int
}

func NewExpressionPool(size int) *ExpressionPool {
	rv := &ExpressionPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return make([]Expression, 0, size)
			},
		},
		size: size,
	}

	return rv
}

func (this *ExpressionPool) Get() []Expression {
	return this.pool.Get().([]Expression)
}

func (this *ExpressionPool) GetSized(length int) []Expression {
	if length > this.size {
		return make([]Expression, length)
	}

	rv := this.Get()
	rv = rv[0:length]
	for i := 0; i < length; i++ {
		rv[i] = nil
	}

	return rv
}

func (this *ExpressionPool) Put(s []Expression) {
	if cap(s) < this.size || cap(s) > 2*this.size {
		return
	}

	this.pool.Put(s[0:0])
}

func (this *ExpressionPool) Size() int {
	return this.size
}
