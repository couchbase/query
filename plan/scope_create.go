//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package plan

import (
	"encoding/json"
	"github.com/couchbase/query/algebra"
	"github.com/couchbase/query/datastore"
)

// Create scope
type CreateScope struct {
	ddl
	bucket datastore.Bucket
	node   *algebra.CreateScope
}

func NewCreateScope(bucket datastore.Bucket, node *algebra.CreateScope) *CreateScope {
	return &CreateScope{
		bucket: bucket,
		node:   node,
	}
}

func (this *CreateScope) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateScope(this)
}

func (this *CreateScope) New() Operator {
	return &CreateScope{}
}

func (this *CreateScope) Bucket() datastore.Bucket {
	return this.bucket
}

func (this *CreateScope) Node() *algebra.CreateScope {
	return this.node
}

func (this *CreateScope) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *CreateScope) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "CreateScope"}
	this.node.Scope().MarshalKeyspace(r)

	if f != nil {
		f(r)
	}
	return r
}

func (this *CreateScope) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_         string `json:"#operator"`
		Namespace string `json:"namespace"`
		Bucket    string `json:"bucket"`
		Scope     string `json:"scope"`
	}

	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}

	scpref := algebra.NewScopeRefFromPath(algebra.NewPathScope(_unmarshalled.Namespace, _unmarshalled.Bucket, _unmarshalled.Scope), "")
	this.bucket, err = datastore.GetBucket(_unmarshalled.Namespace, _unmarshalled.Bucket)
	if err != nil {
		return err
	}

	this.node = algebra.NewCreateScope(scpref)
	return nil
}

func (this *CreateScope) verify(prepared *Prepared) bool {
	var res bool

	this.bucket, res = verifyBucket(this.bucket, prepared)
	return res
}
