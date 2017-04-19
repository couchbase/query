//  Copyright (c) 2017 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package util

// simple implementation of a SQL92 Interval like duration representation

import (
	"fmt"
	"time"
)

type Qualifier int

const (
	YEAR Qualifier = iota
	MONTH
	DAY
	HOUR
	MINUTE
	SECOND
	FRACTION
)

const (
	fraction = 1
	second   = fraction * 1000000000
	minute   = second * 60
	hour     = minute * 60
	day      = hour * 24
	month    = day * 30  // ...not to be taken literally
	year     = day * 365 // ditto
)

var multipliers = [...]int64{
	year,
	month,
	day,
	hour,
	minute,
	second,
	fraction,
}

var formats = [...]string{
	"%04d",
	"%02d",
	"%02d",
	"%02d",
	"%02d",
	"%02d",
	"%09d",
}

func ToTiming(d time.Duration) string {
	return ToQualifiedInterval(d, HOUR, FRACTION, 5, true)
}

// we can't import errors because of circular dependencies, so bad qualifier ranges
// are signified by empty strings
func ToQualifiedInterval(d time.Duration, start Qualifier, end Qualifier, precision int, capped bool) string {

	// check for valid qualifiers
	// we only accept YEAR to MONTH or DAY or lower to anything lower
	if end < start {
		return ""
	}
	if start < DAY && end > MONTH {
		return ""
	}

	if end == FRACTION && (precision < 1 || precision > 9) {
		return ""
	}

	useSeparator := false
	res := ""
	digits := ""
	intvl := int64(d)

	// intervals offer a way to make meaningful duration comparisons
	// to achieve that, we need to ensure that the output format is
	// fixed, so that string comparisons match duration comparison results
	// to that end, if a duration exceeds what would fit in an interval
	// of specific qualifiers, will cap it to the maximum actually fits
	// although, this has the potential to lose data, for specific
	// applications (eg monitoring, profiling), the risk is non existent,
	// and even then capped timings still convey that something is not
	// performing correctly, and the advantage of sorting outweighs the
	// risk.
	if capped && start != YEAR {
		limit := multipliers[start-1]
		if intvl > limit {
			intvl = limit
		}
	}

	if intvl < 0 {
		res = "-"
		intvl = -intvl
	}

	// go through each interval element
	for ; start <= end; start++ {

		// prepend a separator
		if useSeparator {
			switch start {
			case MONTH:
				res = res + "-"
			case HOUR:
				res = res + " "
			case FRACTION:
				res = res + "."
			default:
				res = res + ":"
			}
		}
		element := intvl / multipliers[start]
		intvl = intvl % multipliers[start]
		if start == FRACTION {
			digits = fmt.Sprintf(formats[start], element)
			digits = digits[0:precision]
		} else if capped || useSeparator {
			digits = fmt.Sprintf(formats[start], element)
		} else {
			digits = fmt.Sprintf("%d", element)
		}

		res = res + digits
		useSeparator = true
	}

	return res
}
