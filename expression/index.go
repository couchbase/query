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
