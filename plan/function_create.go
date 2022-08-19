//  Copyright 2019-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/functions"
	"github.com/couchbase/query/functions/resolver"
	"github.com/couchbase/query/functions/storage"
)

// Create function
type CreateFunction struct {
	ddl
	name         functions.FunctionName
	body         functions.FunctionBody
	replace      bool
	failIfExists bool
}

func NewCreateFunction(node *algebra.CreateFunction) *CreateFunction {
	return &CreateFunction{
		name:         node.Name(),
		body:         node.Body(),
		replace:      node.Replace(),
		failIfExists: node.FailIfExists(),
	}
}

func (this *CreateFunction) Name() functions.FunctionName {
	return this.name
}

func (this *CreateFunction) Body() functions.FunctionBody {
	return this.body
}

func (this *CreateFunction) Replace() bool {
	return this.replace
}

func (this *CreateFunction) FailIfExists() bool {
	return this.failIfExists
}

func (this *CreateFunction) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateFunction(this)
}

func (this *CreateFunction) New() Operator {
	return &CreateFunction{}
}

func (this *CreateFunction) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateFunction) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateFunction"}
	identity := make(map[string]interface{})
	definition := make(map[string]interface{})
	this.name.Signature(identity)
	this.body.Body(definition)
	r["identity"] = identity
	r["definition"] = definition
	r["replace"] = this.replace
	r["fail_if_exists"] = this.failIfExists

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateFunction) UnmarshalJSON(bytes []byte) error {
	var _unmarshalled struct {
		_            string          `json:"#operator"`
		Identity     json.RawMessage `json:"identity"`
		Definition   json.RawMessage `json:"definition"`
		Replace      bool            `json:"replace"`
		FailIfExists bool            `json:"fail_if_exists"`
	}
	var newErr errors.Error

	err := json.Unmarshal(bytes, &_unmarshalled)
	if err != nil {
		return err
	}

	this.name, err = storage.MakeName(_unmarshalled.Identity)
	if err != nil {
		return err
	}
	this.body, newErr = resolver.MakeBody(this.name.Name(), _unmarshalled.Definition)
	if newErr != nil {
		return newErr.GetICause()
	}
	this.replace = _unmarshalled.Replace
	this.failIfExists = _unmarshalled.FailIfExists
	return nil
}
