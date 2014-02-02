//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

/*

Package execute provides query execution.

*/
package execute

import (
	_ "fmt"

	_ "github.com/couchbaselabs/query/err"
	"github.com/couchbaselabs/query/value"
)

type Operator interface {
	Accept(visitor Visitor) (interface{}, error)
	Source() Operator
	SetSource(source Operator)
	Handle() *Handle
	SetHandle(handle *Handle)
	Copy() Operator
	Run(context *Context)
}

type Handle struct {
	Chan value.ValueChannel
}

type operatorBase struct {
	source Operator
	handle *Handle
}

func (this *operatorBase) Source() Operator {
	return this.source
}

func (this *operatorBase) SetSource(source Operator) {
	this.source = source
}

func (this *operatorBase) Handle() *Handle {
	return this.handle
}

func (this *operatorBase) SetHandle(handle *Handle) {
	this.handle = handle
}

func (this *operatorBase) copy() operatorBase {
	return operatorBase{this.source, this.handle}
}
