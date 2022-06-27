//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

//go:build enterprise

package tenant

import (
	"strconv"
	"time"

	"github.com/couchbase/cbauth/service"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/regulator"
	"github.com/couchbase/regulator/factory"
	"github.com/couchbase/regulator/metering"
	"github.com/gorilla/mux"
)

var isServerless bool
var resourceManagers []ResourceManager

type Unit atomic.AlignedUint64
type Service int
type Services [_SIZER]Unit
type ResourceManager func(string)

const (
	QUERY_CU = Service(iota)
	JS_CU
	GSI_RU
	FTS_RU
	KV_RU
	KV_WU
	_SIZER
)

var toReg = [_SIZER]struct {
	service regulator.Service
	unit    regulator.UnitType
}{
	{regulator.Query, regulator.Compute},
	{regulator.Query, regulator.Compute},
	{regulator.Index, regulator.Read},
	{regulator.Search, regulator.Read},
	{regulator.Data, regulator.Read},
	{regulator.Data, regulator.Read},
}

func Init(serverless bool) {
	isServerless = serverless
}

func Start(mux *mux.Router, nodeid string, cafile string) {
	if !isServerless {
		return
	}
	handle := factory.InitRegulator(regulator.InitSettings{NodeID: service.NodeID(nodeid), TlsCAFile: cafile, Service: regulator.Query})
	mux.Handle(regulator.MeteringEndpoint, handle).Methods("GET")
}

func RegisterResourceManager(m ResourceManager) {
	if !isServerless {
		return
	}
	resourceManagers = append(resourceManagers, m)
}

func IsServerless() bool {
	return isServerless
}

func AddUnit(dest *Unit, u Unit) {
	atomic.AddUint64((*atomic.AlignedUint64)(dest), uint64(u))
}

func (this Unit) String() string {
	return strconv.FormatUint(uint64(this), 10)
}

func (this Unit) NonZero() bool {
	return this > 0
}

func Throttle(user, bucket string) error {
	tenant := findTenant(user, bucket)
	if tenant == "" {
		return errors.NewServiceTenantInvalidError()
	}
	ctx := regulator.NewUserCtx(tenant, user)
	r, d, e := regulator.CheckQuota(ctx, &regulator.CheckQuotaOpts{
		MaxThrottle:       time.Duration(0),
		NoThrottle:        false,
		NoReject:          true,
		EstimatedDuration: time.Duration(0),
		EstimatedUnits:    []regulator.Units{},
	})
	switch r {
	case regulator.CheckResultNormal:
		return nil
	case regulator.CheckResultThrottle:
		time.Sleep(d)
		return nil
	default:
		return e
	}
}

type Context regulator.UserCtx

func NewTenantCtx(user, bucket string) Context {
	tenant := findTenant(user, bucket)
	return regulator.NewUserCtx(tenant, user).(Context)
}

// TODO define units for query and js-evaluator
func RecordCU(ctx Context, d time.Duration, m uint64) Unit {
	units, _ := metering.QueryEvalComputeToCU(d, m)
	regulator.RecordUnits(ctx, units)
	return Unit(units.Whole())
}

func RecordJsCU(ctx Context, d time.Duration, m uint64) Unit {
	units, _ := metering.QueryUDFComputeToCU(d, m)
	regulator.RecordUnits(ctx, units)
	return Unit(units.Whole())
}

func RefundUnits(ctx Context, units Services) error {
	for s, u := range units {
		if u > 0 {
			ru, err := regulator.NewUnits(toReg[s].service, toReg[s].unit, uint64(u))
			if err != nil {
				return err
			}
			err = regulator.RefundUnits(ctx, ru)
			if err != nil {
				return nil
			}
		}
	}
	return nil
}

// TODO collect from regulator
func QueryCUName() string {
	return "queryCU"
}

func JsCUName() string {
	return "jsCU"
}

func GsiRUName() string {
	return "gsiRU"
}

func FtsRUName() string {
	return "ftsRU"
}

func KvRUName() string {
	return "kvRU"
}

func KvWUName() string {
	return "kvWU"
}

func findTenant(user, bucket string) string {

	// TODO temporary until cbauth gives us buckets associated with users
	if len(bucket) > 0 {
		return bucket
	}
	return ""
}
