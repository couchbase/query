//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package datastore

import (
	"net/http"

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/errors"
)

var NULL_CONTEXT Context = &contextImpl{}

var NULL_QUERY_CONTEXT QueryContext = &queryContextImpl{}

type Context interface {
	GetScanCap() int64
	Fatal(errors.Error)
	Error(errors.Error)
	Warning(errors.Error)
}

type contextImpl struct {
}

func (ci *contextImpl) GetScanCap() int64 {
	return GetScanCap()
}

func (ci *contextImpl) Fatal(err errors.Error) {
}

func (ci *contextImpl) Error(err errors.Error) {
}

func (ci *contextImpl) Warning(err errors.Error) {
}

// A subset of execution.Context that is useful at the datastore level.
type QueryContext interface {
	Credentials() auth.Credentials
	AuthenticatedUsers() []string
	OriginalHttpRequest() *http.Request
	Warning(errors.Error)
}

type queryContextImpl struct {
}

func (ci *queryContextImpl) Credentials() auth.Credentials {
	return make(auth.Credentials, 0)
}

func (ci *queryContextImpl) AuthenticatedUsers() []string {
	return make([]string, 0, 16)
}

func (ci *queryContextImpl) OriginalHttpRequest() *http.Request {
	return nil
}

func (ci *queryContextImpl) Warning(err errors.Error) {
}
