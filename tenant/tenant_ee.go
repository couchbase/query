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
	"sync"
	"time"

	"github.com/couchbase/cbauth/service"
	atomic "github.com/couchbase/go-couchbase/platform"
	"github.com/couchbase/query/distributed"
	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/server/http/router"
	"github.com/couchbase/query/util"
	"github.com/couchbase/regulator"
	"github.com/couchbase/regulator/factory"
	"github.com/couchbase/regulator/metering"
)

var isServerless bool
var resourceManagers []ResourceManager
var throttleTimes map[string]util.Time = make(map[string]util.Time, _MAX_TENANTS)
var throttleLock sync.RWMutex
var thisNodeId service.NodeID

type Unit atomic.AlignedUint64
type Service int
type Services [_SIZER]Unit
type ResourceManager func(string)

type Context interface {
	regulator.Ctx
	User() string
}

type Endpoint interface {
	Router() router.Router
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

const _THROTTLE_DELAY = 500 * time.Millisecond

var toReg = [_SIZER]struct {
	service  regulator.Service
	unit     regulator.UnitType
	billable bool
}{
	{regulator.Query, regulator.Compute, false}, // query, not billable
	{regulator.Query, regulator.Compute, false}, // js, billable
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
	thisNodeId = service.NodeID(nodeid)
	handle := factory.InitRegulator(regulator.InitSettings{NodeID: service.NodeID(nodeid),
		SettingsFile: regulatorsettingsfile, Service: regulator.Query,
		ServiceCheckMask: regulator.Index | regulator.Search})
	router := endpoint.Router()
	tenantHandler := func(w http.ResponseWriter, req *http.Request) {
		err := endpoint.Authorize(req)
		if err != nil {
			endpoint.WriteError(err, w, req)
			return
		}
		handle.WriteMetrics(w)
	}
	router.Map(regulator.MeteringEndpoint, tenantHandler, "GET")
	router.Map("/_prometheusMetricsHigh", tenantHandler, "GET")
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

func Throttle(isAdmin bool, user, bucket string, buckets []string, timeout time.Duration) (Context, time.Duration, errors.Error) {
	var ctx Context

	tenant := bucket
	if tenant == "" {
		if isAdmin {
			ctx = regulator.NewNoBucketCtx(user)

			// currently we don't throttle requests that have no tenant associated
			return ctx, time.Duration(0), nil
		} else {
			return nil, time.Duration(0), errors.NewServiceTenantMissingError()
		}
	} else {
		found := false
		for _, b := range buckets {
			if b == tenant {
				found = true
				break
			}
		}
		if !found {
			if isAdmin {
				return nil, time.Duration(0), errors.NewServiceTenantNotFoundError(bucket)
			} else {
				return nil, time.Duration(0), errors.NewServiceTenantNotAuthorizedError(bucket)
			}
		}
		ctx = regulator.NewUserCtx(tenant, user)
	}

	quotaOpts := &regulator.CheckQuotaOpts{
		Timeout: timeout,
	}
	for {
		r, d, e := regulator.CheckQuota(ctx, quotaOpts)
		switch r {
		case regulator.CheckResultProceed:
			var d time.Duration

			// if KV is throttling this tenant slow it down before the request starts in order
			// to use the would be KV throttling to service other less active tenants
			// the query throttling will limit KV requests which in turn will lessen KV need
			// for throttling
			throttleLock.RLock()
			thottleTime, ok := throttleTimes[bucket]
			throttleLock.RUnlock()
			if ok {
				d = thottleTime.Sub(util.Now())

				if d > time.Duration(0) {
					regulator.RecordExternalThrottle(ctx, regulator.ExternalThrottleSpec{
						Duration: d,
						Timing:   regulator.Preceding,
						Service:  regulator.Query,
						NodeID:   thisNodeId,
					})
					logging.Debugf("External bucket %v throttled for %v by query", bucket, d)
					time.Sleep(d)
				} else {
					d = time.Duration(0)
				}

				// remove delay hint to minimise cost
				throttleLock.Lock()
				currThottleTime, ook := throttleTimes[bucket]
				if ook && currThottleTime == thottleTime {
					delete(throttleTimes, bucket)
				}
				throttleLock.Unlock()
			}
			return ctx, d, nil
		case regulator.CheckResultThrottleProceed:
			logging.Debugf("bucket %v throttled for %v by regulator (%v)", bucket, d, r)
			time.Sleep(d)
			return ctx, d, nil
		case regulator.CheckResultThrottleRetry:

			// this code relies on the regulator not to exceed the request or config timeout
			logging.Debugf("bucket %v throttled for %v by regulator (%v)", bucket, d, r)
			time.Sleep(d)
			logging.Debugf("retrying bucket %v previously throttled for %v by regulator (%v)", bucket, d, r)
		case regulator.CheckResultReject:
			logging.Debugf("bucket %v rejected by regulator", bucket)
			return nil, time.Duration(0), errors.NewServiceTenantRejectedError(d)
		default:
			logging.Debugf("bucket %v error by regulator (%v) ", bucket, e)
			return ctx, time.Duration(0), errors.NewServiceTenantThrottledError(e)
		}
	}
}

func Bucket(ctx Context) string {
	bucketCtx, _ := ctx.(interface{ Bucket() string })
	if bucketCtx != nil {
		return bucketCtx.Bucket()
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
	regulator.RecordUnits(ctx, units...)
	if len(units) == 0 {
		logging.Warnf("bucket %v duration %v memory %v unexpected 0-length response from regulator CU compute",
			ctx.Bucket(), d, m)
		return 0
	}
	return Unit(units[0].Whole())
}

func RecordJsCU(ctx Context, d time.Duration, m uint64) Unit {
	units, _ := metering.QueryUDFComputeToCU(d, m)
	regulator.RecordUnits(ctx, units...)
	if len(units) == 0 {
		logging.Warnf("bucket %v duration %v memory %v unexpected 0-length response from regulator CU compute",
			ctx.Bucket(), d, m)
		return 0
	}
	return Unit(units[0].Whole())
}

func NeedRefund(ctx Context, errs []errors.Error, warns []errors.Error) bool {

	// TODO extend
	for _, e := range errs {
		if e.Code() == errors.E_NODE_QUOTA_EXCEEDED {
			return true
		}
	}
	return false
}

func RefundUnits(ctx Context, units Services) error {
	bucketCtx, _ := ctx.(interface{ Bucket() string })

	// no refund needed for full admin
	if bucketCtx == nil || bucketCtx.Bucket() == "" {
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

func EncodeNodeName(name string) string {
	if isServerless {
		return distributed.RemoteAccess().NodeUUID(name)
	} else {
		return name
	}
}

func DecodeNodeName(name string) string {
	if isServerless {
		return distributed.RemoteAccess().UUIDToHost(name)
	} else {
		return name
	}
}

func Suspend(bucket string, delay time.Duration, node string) {
	t := util.Now().Add(delay)
	throttleLock.Lock()
	oldT, ok := throttleTimes[bucket]
	doLog := !ok || t.Sub(oldT) > time.Duration(0)
	if doLog {
		doLog = true
		throttleTimes[bucket] = t
	}
	throttleLock.Unlock()
	if doLog {
		logging.Debugf("bucket %v throttled to %v by KV", bucket, t)
		ctx := regulator.NewBucketCtx(bucket)
		regulator.RecordExternalThrottle(ctx, regulator.ExternalThrottleSpec{
			Duration: delay,
			Timing:   regulator.Preceding,
			Service:  regulator.Data,
			HostPort: node,
		})
	}
}

func IsSuspended(bucket string) bool {
	throttleLock.Lock()
	t, suspended := throttleTimes[bucket]
	if suspended && util.Now().Sub(t) > time.Duration(0) {
		suspended = false
	}
	throttleLock.Unlock()
	return suspended
}
