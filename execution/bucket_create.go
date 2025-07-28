//  Copyright 2023-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package execution

import (
	"encoding/json"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

const (
	_RAM_QUOTA           = "ramQuota"
	_STORAGE_BACKEND     = "storageBackend"
	_NUM_VBUCKETS        = "numVBuckets"
	_MAGMA               = "magma"
	_MIN_RAM_QUOTA       = 100
	_MIN_RAM_QUOTA_MAGMA = 1024
)

type CreateBucket struct {
	base
	plan *plan.CreateBucket
}

func NewCreateBucket(plan *plan.CreateBucket, context *Context) *CreateBucket {
	rv := &CreateBucket{
		plan: plan,
	}

	newRedirectBase(&rv.base, context)
	rv.output = rv
	return rv
}

func (this *CreateBucket) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitCreateBucket(this)
}

func (this *CreateBucket) Copy() Operator {
	rv := &CreateBucket{plan: this.plan}
	this.base.copy(&rv.base)
	return rv
}

func (this *CreateBucket) PlanOp() plan.Operator {
	return this.plan
}

func (this *CreateBucket) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover(&this.base) // Recover from any panic
		active := this.active()
		defer this.close(context)
		this.switchPhase(_EXECTIME)
		defer this.switchPhase(_NOTIME)
		defer this.notify() // Notify that I have stopped

		if !active || context.Readonly() {
			return
		}

		this.switchPhase(_SERVTIME)
		with := this.plan.Node().With()
		if with != nil {
			// add a default ramQuota if not specified, according to the storage type
			if _, ok := with.Field(_RAM_QUOTA); !ok {
				// Check if storage is magma and if numVBuckets is specified. If it is not specified, default numVBuckets is 128 and need to use the MIN_RAM_QUOTA.
				useMagmaQuota := false
				if v, ok := with.Field(_STORAGE_BACKEND); ok && v.Type() == value.STRING && v.ToString() == _MAGMA {
					// If numVBuckets is specified and is 1024
					if nv, ok := with.Field(_NUM_VBUCKETS); ok {
						if numVBuckets, ok := value.IsIntValue(nv); ok && numVBuckets == 1024 {
							useMagmaQuota = true
						}
					}
				}

				if useMagmaQuota {
					with.SetField(_RAM_QUOTA, _MIN_RAM_QUOTA_MAGMA)
				} else {
					with.SetField(_RAM_QUOTA, _MIN_RAM_QUOTA)
				}
			}
		}
		err := context.datastore.CreateBucket(this.plan.Node().Name(), with)
		if err != nil {
			ae := errors.IsExistsError("Bucket", err)
			if !ae || this.plan.Node().FailIfExists() {
				if ae {
					err = errors.NewCbBucketExistsError(this.plan.Node().Name())
				}
				context.Error(err)
			}
		}
	})
}

func (this *CreateBucket) MarshalJSON() ([]byte, error) {
	r := this.plan.MarshalBase(func(r map[string]interface{}) {
		this.marshalTimes(r)
	})
	return json.Marshal(r)
}
