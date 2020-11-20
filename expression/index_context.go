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

	"github.com/couchbase/query/auth"
	"github.com/couchbase/query/value"
)

/*
Type IndexContext is a structure containing a variable
now that is of type Time which represents an instant in
time.
*/
type IndexContext struct {
	now time.Time
}

/*
This method returns a pointer to the IndecContext
structure, after assigning its value now with the
current local time using the time package's Now
function.
*/
func NewIndexContext() Context {
	return &IndexContext{
		now: time.Now(),
	}
}

/*
This method allows us to access the value now in the
receiver of type IndexContext. It returns the now
value from the receiver.
*/
func (this *IndexContext) Now() time.Time {
	return this.now
}

// 5 next methods are unused and only for expression Context compatibility
func (this *IndexContext) AuthenticatedUsers() []string {
	return []string{"NEVER_USED"}
}

func (this *IndexContext) Credentials() *auth.Credentials {
	return nil
}

func (this *IndexContext) DatastoreVersion() string {
	return "BOGUS_VERSION"
}

func (this *IndexContext) EvaluateStatement(statement string, namedArgs map[string]value.Value, positionalArgs value.Values, subquery, readonly bool) (value.Value, uint64, error) {
	return nil, 0, nil
}

func (this *IndexContext) Readonly() bool {
	return true
}

func (this *IndexContext) NewQueryContext(queryContext string, readonly bool) interface{} {
	return nil
}

func (this *IndexContext) SetAdvisor() {
	// no-op
}
