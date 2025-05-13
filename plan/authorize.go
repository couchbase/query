//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package plan

import (
	"encoding/json"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/datastore"
)

type Authorize struct {
	readonly
	privs   *auth.Privileges `json:"privileges"`
	child   Operator         `json:"~child"`
	dynamic bool             `json:"dynamic"`
}

func NewAuthorize(privs *auth.Privileges, child Operator) *Authorize {
	rv := &Authorize{
		privs: privs,
		child: child,
	}

	if privs != nil {
		privs.ForEach(func(pp auth.PrivilegePair) {
			if (pp.Props & auth.PRIV_PROPS_DYNAMIC_TARGET) != 0 {
				rv.dynamic = true
			}
		})
	}
	if !rv.dynamic {
		datastore.GetDatastore().PreAuthorize(rv.privs)
	}

	return rv
}

func (this *Authorize) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitAuthorize(this)
}

func (this *Authorize) New() Operator {
	return &Authorize{}
}

func (this *Authorize) Privileges() *auth.Privileges {
	return this.privs
}

func (this *Authorize) Readonly() bool {
	return this.child.Readonly()
}

func (this *Authorize) Child() Operator {
	return this.child
}

func (this *Authorize) Dynamic() bool {
	return this.dynamic
}

func (this *Authorize) Cost() float64 {
	return this.child.Cost()
}

func (this *Authorize) Cardinality() float64 {
	return this.child.Cardinality()
}

func (this *Authorize) Size() int64 {
	return this.child.Size()
}

func (this *Authorize) FrCost() float64 {
	return this.child.FrCost()
}

func (this *Authorize) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.MarshalBase(nil))
}

func (this *Authorize) MarshalBase(f func(map[string]interface{})) map[string]interface{} {
	r := map[string]interface{}{"#operator": "Authorize"}
	r["privileges"] = this.privs
	if this.dynamic {
		r["dynamic"] = this.dynamic
	}
	if f != nil {
		f(r)
	} else {
		r["~child"] = this.child
	}
	return r
}

func (this *Authorize) UnmarshalJSON(body []byte) error {
	var _unmarshalled struct {
		_       string           `json:"#operator"`
		Privs   *auth.Privileges `json:"privileges"`
		Child   json.RawMessage  `json:"~child"`
		Dynamic bool             `json:"Dynamic"`
	}
	var child_type struct {
		Operator string `json:"#operator"`
	}
	err := json.Unmarshal(body, &_unmarshalled)
	if err != nil {
		return err
	}
	this.privs = _unmarshalled.Privs
	this.dynamic = _unmarshalled.Dynamic
	if !this.dynamic {
		datastore.GetDatastore().PreAuthorize(this.privs)
	}

	err = json.Unmarshal(_unmarshalled.Child, &child_type)
	if err != nil {
		return err
	}
	this.child, err = MakeOperator(child_type.Operator, _unmarshalled.Child, this.PlanContext())
	return err
}

func (this *Authorize) verify(prepared *Prepared) bool {
	return this.child.verify(prepared)
}
