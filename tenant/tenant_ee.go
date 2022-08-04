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
	"net/http"
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

type Context regulator.UserCtx
type Endpoint interface {
	Mux() *mux.Router
	Authorize(req *http.Request) errors.Error
	WriteError(err errors.Error, w http.ResponseWriter, req *http.Request)
}

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
	service  regulator.Service
	unit     regulator.UnitType
	billable bool
}{
	{regulator.Query, regulator.Compute, false}, // query, not billable
	{regulator.Query, regulator.Compute, true},  // js, billable
	{regulator.Index, regulator.Read, true},     // gsi, billable
	{regulator.Search, regulator.Read, true},    // fts, billable
	{regulator.Data, regulator.Read, true},      // kv ru, billable
	{regulator.Data, regulator.Write, true},     // kv wu, billable
}

func Init(serverless bool) {
	isServerless = serverless
}

func Start(endpoint Endpoint, nodeid string, regulatorsettingsfile string) {
	if !isServerless {
		return
	}
	handle := factory.InitRegulator(regulator.InitSettings{NodeID: service.NodeID(nodeid),
		SettingsFile: regulatorsettingsfile, Service: regulator.Query,
		ServiceCheckMask: regulator.Index | regulator.Search})
	mux := endpoint.Mux()
	tenantHandler := func(w http.ResponseWriter, req *http.Request) {
		err := endpoint.Authorize(req)
		if err != nil {
			endpoint.WriteError(err, w, req)
			return
		}
		handle.WriteMetrics(w)
	}
	mux.HandleFunc(regulator.MeteringEndpoint, tenantHandler).Methods("GET")
	mux.HandleFunc("/_prometheusMetricsHigh", tenantHandler).Methods("GET")
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

func Throttle(isAdmin bool, user, bucket string, buckets []string, timeout time.Duration) (Context, error) {

	if isAdmin && len(buckets) == 0 {
		return regulator.NewUserCtx("", user), nil
	}
	tenant := bucket
	if tenant == "" {
		if len(buckets) == 0 {
			return nil, errors.NewServiceTenantMissingError()
		}
		tenant = buckets[0]
	} else {
		found := false
		for _, b := range buckets {
			if b == tenant {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.NewServiceTenantNotAuthorizedError(bucket)
		}
	}
	ctx := regulator.NewUserCtx(tenant, user)
	r, d, e := regulator.CheckQuota(ctx, &regulator.CheckQuotaOpts{
		MaxThrottle:       timeout,
		NoThrottle:        false,
		NoReject:          false,
		EstimatedDuration: time.Duration(0),
		EstimatedUnits:    []regulator.Units{},
	})
	switch r {
	case regulator.CheckResultNormal:
		return ctx, nil
	case regulator.CheckResultThrottle:
		time.Sleep(d)
		return ctx, nil
	case regulator.CheckResultReject:
		return nil, errors.NewRejectRequestError(d)
	default:
		return ctx, e
	}
}

func Bucket(ctx Context) string {
	if ctx != nil {
		return ctx.Bucket()
	}
	return ""
}
func User(ctx Context) string {
	if ctx != nil {
		return ctx.User()
	}
	return ""
}

// TODO define units for query and js-evaluator
func RecordCU(ctx Context, d time.Duration, m uint64) Unit {
	units, _ := metering.QueryEvalComputeToCU(d, m)
	if ctx.Bucket() != "" {
		regulator.RecordUnits(ctx, units)
	}
	return Unit(units.Whole())
}

func RecordJsCU(ctx Context, d time.Duration, m uint64) Unit {
	units, _ := metering.QueryUDFComputeToCU(d, m)
	if ctx.Bucket() != "" {
		regulator.RecordUnits(ctx, units)
	}
	return Unit(units.Whole())
}

func RefundUnits(ctx Context, units Services) error {

	// no refund needed for full admin
	if ctx.Bucket() == "" {
		return nil
	}
	for s, u := range units {
		if u.NonZero() && toReg[s].billable {
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

func Units2Map(serv Services) map[string]interface{} {
	var out []regulator.Units

	for s, u := range serv {
		if u.NonZero() && toReg[s].billable {
			ru, err := regulator.NewUnits(toReg[s].service, toReg[s].unit, uint64(u))
			if err != nil {
				continue
			}
			out = append(out, ru)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return regulator.UnitsToMap(false, out...)
}
