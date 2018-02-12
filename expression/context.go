//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package expression

import (
	"time"
)

/*
It imports the time package that provides the functionality
to measure and display the time. The type Context is an
interface that has a method Now that returns the Time that
returns the instant it time with a nanosecond precision.
*/
type Context interface {
	Now() time.Time
	AuthenticatedUsers() []string
	DatastoreVersion() string
}

type CurlContext interface {
	Context
	GetWhitelist() map[string]interface{}
}
