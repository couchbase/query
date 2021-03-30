//  Copyright 2014-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included in
//  the file licenses/Couchbase-BSL.txt.  As of the Change Date specified in that
//  file, in accordance with the Business Source License, use of this software will
//  be governed by the Apache License, Version 2.0, included in the file
//  licenses/APL.txt.

package datastore

import (
	"time"

	"github.com/couchbase/query/errors"
	"github.com/couchbase/query/value"
)

type StopChannel chan bool

type ValueConnection struct {
	valueChannel value.ValueChannel // Closed by the generator when the scan is completed or aborted.
	stopChannel  StopChannel        // Notifies generator  to stop generating. Never closed, just garbage-collected.
	context      Context            // Context.
	timeout      bool               // True if timed out.
}

const _VALUE_CAP = 256 // Buffer size

func NewValueConnection(context Context) *ValueConnection {
	return NewSizedValueConnection(_VALUE_CAP, context)
}

func NewSizedValueConnection(size int, context Context) *ValueConnection {
	return &ValueConnection{
		valueChannel: make(value.ValueChannel, size),
		stopChannel:  make(StopChannel, 1),
		context:      context,
	}
}

func (this *ValueConnection) ValueChannel() value.ValueChannel {
	return this.valueChannel
}

func (this *ValueConnection) StopChannel() StopChannel {
	return this.stopChannel
}

func (this *ValueConnection) Fatal(err errors.Error) {
	this.context.Fatal(err)
}

func (this *ValueConnection) Error(err errors.Error) {
	this.context.Error(err)
}

func (this *ValueConnection) Warning(wrn errors.Error) {
	this.context.Warning(wrn)
}

func (this *ValueConnection) Timeout() bool {
	return this.timeout
}

func (this *ValueConnection) SetTimeout(timeout bool) {
	this.timeout = timeout
}

func (this *ValueConnection) GetReqDeadline() time.Time {
	return this.context.GetReqDeadline()
}
