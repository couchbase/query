//  Copyright (c) 2019 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

// noop is a placeholder operator that does nothing
// used to signify to sequences that we had received a plan from a node running an older version,
// which contains an operator we no longer use
package execution

import (
	"github.com/couchbase/query/value"
)

type Noop struct {
	base
}

var _noop = &Noop{}

func NewNoop() *Noop {
	return _noop
}

func (this *Noop) Accept(visitor Visitor) (interface{}, error) {
	return nil, nil
}

func (this *Noop) Copy() Operator {
	return _noop
}

func (this *Noop) RunOnce(context *Context, parent value.Value) {
}

func (this *Noop) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}
