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
	"time"

	"github.com/couchbaselabs/query/value"
)

type ClockNowMillis struct {
	ExpressionBase
}

func NewClockNowMillis() Function {
	return &ClockNowMillis{}
}

func (this *ClockNowMillis) Evaluate(item value.Value, context Context) (value.Value, error) {
	nanos := time.Now().UnixNano()
	return value.NewValue(float64(nanos) / (1000000.0)), nil
}

func (this *ClockNowMillis) MinArgs() int { return 0 }

func (this *ClockNowMillis) MaxArgs() int { return 0 }

func (this *ClockNowMillis) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}

type ClockNowStr struct {
	ExpressionBase
}

func NewClockNowStr() Function {
	return &ClockNowStr{}
}

func (this *ClockNowStr) Evaluate(item value.Value, context Context) (value.Value, error) {
	str := time.Now().String()
	return value.NewValue(str), nil
}

func (this *ClockNowStr) MinArgs() int { return 0 }

func (this *ClockNowStr) MaxArgs() int { return 0 }

func (this *ClockNowStr) Constructor() FunctionConstructor {
	return func(Expressions) Function { return this }
}
