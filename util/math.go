//  Copyright (c) 2014 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

import (
	"math"

	atomic "github.com/couchbase/go-couchbase/platform"
)

// Comparisons

func MinInt(x, y int) int {
	return int(math.Min(float64(x), float64(y)))
}

func MaxInt(x, y int) int {
	return int(math.Max(float64(x), float64(y)))
}

// Rounding

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

func RoundPlaces(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return Round(f*shift) / shift
}

type Tristate int

const (
	FAILURE = Tristate(iota)
	NOT_DONE
	DONE
)

// atomic test and set
func TestAndSetUint64(loc *atomic.AlignedUint64, val uint64, test func(locVal, val uint64) bool, limit int) Tristate {

	// This works like Power's store with reservation or ARM's conditional store
	// except that rather than only allowing an equality comparison, we offer an
	// arbitrary comparison function
	// This allows to implement various lockless functions such as Min, Max, etc
	// We also give the option of trying no more than a specific number of times,
	// and report a completion status, so that alternative actions can be taken on
	// failure
	if limit <= 0 {
		limit = math.MaxInt32
	}
	for limit > 0 {
		limit--
		oldVal := uint64(*loc)
		if test(oldVal, val) {
			if atomic.CompareAndSwapUint64(loc, oldVal, val) {
				return DONE
			}
		} else {
			return NOT_DONE
		}
	}
	return FAILURE
}
