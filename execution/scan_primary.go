//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package execution

import (
	"math"
	"time"

	"github.com/couchbase/query/datastore"
	"github.com/couchbase/query/logging"
	"github.com/couchbase/query/plan"
	"github.com/couchbase/query/value"
)

type PrimaryScan struct {
	base
	plan *plan.PrimaryScan
}

func NewPrimaryScan(plan *plan.PrimaryScan) *PrimaryScan {
	rv := &PrimaryScan{
		base: newBase(),
		plan: plan,
	}

	rv.output = rv
	return rv
}

func (this *PrimaryScan) Accept(visitor Visitor) (interface{}, error) {
	return visitor.VisitPrimaryScan(this)
}

func (this *PrimaryScan) Copy() Operator {
	return &PrimaryScan{this.base.copy(), this.plan}
}

func (this *PrimaryScan) RunOnce(context *Context, parent value.Value) {
	this.once.Do(func() {
		defer context.Recover()       // Recover from any panic
		defer close(this.itemChannel) // Broadcast that I have stopped
		defer this.notify()           // Notify that I have stopped

		this.scanPrimary(context, parent)
	})
}

func (this *PrimaryScan) scanPrimary(context *Context, parent value.Value) {
	conn := this.newIndexConnection(context)
	defer notifyConn(conn) // Notify index that I have stopped

	var duration time.Duration
	timer := time.Now()
	defer context.AddPhaseTime("scan", time.Since(timer)-duration)

	go this.scanEntries(context, conn)

	var entry *datastore.IndexEntry
	ok := true
	for ok {
		select {
		case <-this.stopChannel:
			return
		default:
		}

		select {
		case entry, ok = <-conn.EntryChannel():
			t := time.Now()

			if ok {
				cv := value.NewScopeValue(make(map[string]interface{}), parent)
				av := value.NewAnnotatedValue(cv)
				av.SetAttachment("meta", map[string]interface{}{"id": entry.PrimaryKey})
				ok = this.sendItem(av)
			}

			duration += time.Since(t)
		case <-this.stopChannel:
			return
		}

	}
}

func (this *PrimaryScan) scanEntries(context *Context, conn *datastore.IndexConnection) {
	defer context.Recover() // Recover from any panic
	this.plan.Index().ScanEntries(context.RequestId(), math.MaxInt64,
		context.ScanConsistency(), context.ScanVector(), conn)
}

func (this *PrimaryScan) newIndexConnection(context *Context) *datastore.IndexConnection {
	var conn *datastore.IndexConnection

	// Use keyspace count to create a sized index connection
	keyspace := this.plan.Keyspace()
	size, err := keyspace.Count()
	if err == nil {
		conn, err = datastore.NewSizedIndexConnection(size, context)
	}

	// Use non-sized API and log error
	if err != nil {
		conn = datastore.NewIndexConnection(context)
		logging.Errorp("PrimaryScan.newIndexConnection ", logging.Pair{"error", err})
	}

	return conn
}
