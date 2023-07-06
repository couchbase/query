//  Copyright 2022-Present Couchbase, Inc.
//
//  Use of this software is governed by the Business Source License included
//  in the file licenses/BSL-Couchbase.txt.  As of the Change Date specified
//  in that file, in accordance with the Business Source License, use of this
//  software will be governed by the Apache License, Version 2.0, included in
//  the file licenses/APL2.txt.

package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type DurationStyle int

const (
	LEGACY DurationStyle = iota
	INTERVAL
	COMPATIBLE
	SECONDS
	DEFAULT // should always be last
)

var styles = map[string]DurationStyle{
	"legacy":     LEGACY,
	"interval":   INTERVAL,
	"compatible": COMPATIBLE,
	"seconds":    SECONDS,
	"default":    DEFAULT,
}

func (this DurationStyle) String() string {
	for k, v := range styles {
		if v == this {
			return k
		}
	}
	return "<!invalid DurationStyle>"
}

func IsDurationStyle(style string) (DurationStyle, bool) {
	if len(style) == 0 {
		return durationStyle, true
	}
	s, ok := styles[strings.ToLower(style)]
	if !ok {
		return durationStyle, false
	}
	return s, true
}

var durationStyle DurationStyle = LEGACY

func SetDurationStyle(style DurationStyle) {
	if style >= DEFAULT {
		style = LEGACY
	}
	durationStyle = style
}

func GetDurationStyle() DurationStyle {
	return durationStyle
}

func OutputDuration(d time.Duration) string {
	return FormatDuration(d, durationStyle)
}

func FormatDuration(d time.Duration, style DurationStyle) string {
	switch style {
	case INTERVAL:
		return trimTrailingZeros(ToQualifiedInterval(d, HOUR, FRACTION, 9, false))
	case COMPATIBLE:
		return trimTrailingZeros(fmt.Sprintf("%.9f", d.Seconds())) + "s"
	case SECONDS:
		return trimTrailingZeros(fmt.Sprintf("%.9f", d.Seconds()))
	case LEGACY:
		return d.String()
	default:
		if durationStyle == DEFAULT {
			panic("durationStyle is DEFAULT")
		}
		return FormatDuration(d, durationStyle)
	}
}

func trimTrailingZeros(s string) string {
	if len(s) > 6 && s[len(s)-6:] == "000000" {
		return s[:len(s)-6]
	} else if len(s) > 3 && s[len(s)-3:] == "000" {
		return s[:len(s)-3]
	}
	return s
}

func ParseDuration(str string) (time.Duration, error) {
	return ParseDurationStyle(str, durationStyle)
}

func ParseDurationStyle(str string, style DurationStyle) (time.Duration, error) {
	var d time.Duration
	var e error
	var all bool

	if style == DEFAULT {
		all = true
		if strings.IndexByte(str, ':') != -1 {
			style = INTERVAL
		} else {
			style = SECONDS
		}
	}

	switch style {
	case SECONDS:
		var f float64
		f, e = strconv.ParseFloat(str, 64)
		if e != nil {
			if all {
				return time.ParseDuration(str)
			}
			return d, e
		}
		d = time.Duration(f * float64(time.Second))
		return d, e
	case INTERVAL:
		var h, m, n int
		var s float64
		parts := strings.Split(str, ":")
		switch len(parts) {
		case 3:
			h, e = strconv.Atoi(parts[n])
			n++
			fallthrough
		case 2:
			if e == nil {
				m, e = strconv.Atoi(parts[n])
				n++
			}
			fallthrough
		case 1:
			if e == nil {
				s, e = strconv.ParseFloat(parts[n], 64)
				if all && e != nil && n == 0 {
					return time.ParseDuration(str)
				}
			}
		default:
			return 0, fmt.Errorf("Invalid duration")
		}
		if e == nil {
			d = time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s*float64(time.Second))
		}
		return d, e
	default: // LEGACY + COMPATIBLE
		return time.ParseDuration(str)
	}
}